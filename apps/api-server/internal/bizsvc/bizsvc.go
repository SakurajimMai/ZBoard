package bizsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

// newClientID returns a UUIDv4-shaped client identifier used by inbound configs.
func newClientID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

const (
	IdempotencyTTL = 24 * time.Hour
)

type Service struct {
	Store *store.Store
}

func New(s *store.Store) *Service { return &Service{Store: s} }

// ===== Plans =====

func (s *Service) ListActivePlans(ctx context.Context) ([]store.Plan, error) {
	return s.Store.ListActivePlans(ctx)
}

func (s *Service) ListAllPlans(ctx context.Context) ([]store.Plan, error) {
	return s.Store.ListAllPlans(ctx)
}

func (s *Service) CreatePlan(ctx context.Context, in store.CreatePlanInput) (int64, error) {
	if in.Name == "" || in.Price == "" || in.DurationDays <= 0 {
		return 0, httpx.NewError(http.StatusBadRequest, "bad_request", "套餐字段不完整")
	}
	return s.Store.CreatePlan(ctx, in)
}

// ===== Orders =====

type OrderResult struct {
	Existing bool         `json:"existing"`
	Order    *store.Order `json:"order"`
}

// CreateOrder creates a pending order with optional Idempotency-Key support.
// When idempotencyKey is non-empty, the same key returns the same order; a
// different request body with the same key returns a 409.
func (s *Service) CreateOrder(ctx context.Context, userID, planID int64, period, idempotencyKey string) (*OrderResult, error) {
	plan, err := s.Store.FindPlanByID(ctx, planID)
	if err != nil {
		if store.IsNoRows(err) {
			return nil, httpx.NewError(http.StatusNotFound, "plan_not_found", "套餐不存在")
		}
		return nil, err
	}
	if plan.Status != "active" {
		return nil, httpx.NewError(http.StatusBadRequest, "plan_inactive", "套餐已下架")
	}
	period = store.NormalizeBillingPeriod(period)

	requestHash := hashRequest(map[string]any{"user_id": userID, "plan_id": planID, "period": period})
	scope := "orders.create"

	if idempotencyKey != "" {
		claimed, existing, err := s.Store.ClaimIdempotency(ctx, scope, idempotencyKey, requestHash, IdempotencyTTL)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if existing.RequestHash == nil || *existing.RequestHash != requestHash {
				return nil, httpx.NewError(http.StatusConflict, "idempotency_mismatch", "Idempotency-Key 与原始请求不一致")
			}
			if existing.ResponseBody != nil {
				var prior OrderResult
				if err := json.Unmarshal([]byte(*existing.ResponseBody), &prior); err == nil {
					prior.Existing = true
					return &prior, nil
				}
			}
			return nil, httpx.NewError(http.StatusConflict, "idempotency_in_progress", "请求处理中，请稍后重试")
		}
		o, err := s.insertOrder(ctx, userID, plan, period)
		if err != nil {
			return nil, err
		}
		body, _ := json.Marshal(OrderResult{Order: o})
		_ = s.Store.CompleteIdempotency(ctx, claimed.ID, string(body), "succeeded")
		return &OrderResult{Order: o}, nil
	}

	o, err := s.insertOrder(ctx, userID, plan, period)
	if err != nil {
		return nil, err
	}
	return &OrderResult{Order: o}, nil
}

func (s *Service) insertOrder(ctx context.Context, userID int64, plan *store.Plan, period string) (*store.Order, error) {
	price, credit, err := s.calculatePlanOrderAmount(ctx, userID, plan, period)
	if err != nil {
		return nil, err
	}
	amount := price - credit
	if amount < 0 {
		amount = 0
	}
	orderNo := newOrderNo()
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	o := &store.Order{
		OrderNo:        orderNo,
		UserID:         userID,
		PlanID:         plan.ID,
		Kind:           store.OrderKindPlan,
		BillingPeriod:  period,
		Amount:         centsToMoney(amount),
		OriginalAmount: centsToMoney(price),
		CreditAmount:   centsToMoney(credit),
		Currency:       "CNY",
		Status:         "pending",
		ExpiredAt:      &expiresAt,
	}
	id, err := s.Store.CreateOrder(ctx, o)
	if err != nil {
		return nil, err
	}
	o.ID = id
	return s.Store.FindOrderByNo(ctx, orderNo)
}

func (s *Service) calculatePlanOrderAmount(ctx context.Context, userID int64, plan *store.Plan, period string) (int64, int64, error) {
	priceCents, err := planPeriodPriceCents(plan, period)
	if err != nil {
		return 0, 0, httpx.NewError(http.StatusBadRequest, "bad_price", "套餐价格不合法")
	}
	u, err := s.Store.FindUserByID(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	creditCents, err := s.calculateUpgradeCreditCents(ctx, u, plan)
	if err != nil {
		return 0, 0, err
	}
	if creditCents > priceCents {
		creditCents = priceCents
	}
	return priceCents, creditCents, nil
}

func (s *Service) calculateUpgradeCreditCents(ctx context.Context, u *store.User, targetPlan *store.Plan) (int64, error) {
	now := time.Now().UTC()
	if u.PlanID == nil || *u.PlanID == 0 || *u.PlanID == targetPlan.ID || u.ExpiredAt == nil || !u.ExpiredAt.After(now) {
		return 0, nil
	}
	currentPlan, err := s.Store.FindPlanByID(ctx, *u.PlanID)
	if err != nil {
		if store.IsNoRows(err) {
			return 0, nil
		}
		return 0, err
	}
	currentPeriod := store.NormalizeBillingPeriod(u.PlanPeriod)
	currentPrice, err := planPeriodPriceCents(currentPlan, currentPeriod)
	if err != nil {
		return 0, nil
	}
	totalSeconds := int64(store.PlanDurationDays(currentPlan, currentPeriod)) * 24 * 60 * 60
	if totalSeconds <= 0 {
		return 0, nil
	}
	remainingSeconds := int64(math.Ceil(u.ExpiredAt.Sub(now).Seconds()))
	if remainingSeconds <= 0 {
		return 0, nil
	}
	if remainingSeconds > totalSeconds {
		remainingSeconds = totalSeconds
	}
	unusedTraffic := u.TrafficLimit - u.TrafficUsed
	if unusedTraffic < 0 {
		unusedTraffic = 0
	}
	if u.TrafficLimit <= 0 {
		return currentPrice * remainingSeconds / totalSeconds, nil
	}
	return currentPrice * remainingSeconds * unusedTraffic / totalSeconds / u.TrafficLimit, nil
}

func planPeriodPriceCents(plan *store.Plan, period string) (int64, error) {
	monthly, err := moneyToCents(plan.Price)
	if err != nil {
		return 0, err
	}
	switch store.NormalizeBillingPeriod(period) {
	case store.BillingPeriodQuarterly:
		price, err := moneyToCents(plan.QuarterlyPrice)
		if err == nil && price > 0 {
			return price, nil
		}
		return monthly * 3, nil
	case store.BillingPeriodYearly:
		price, err := moneyToCents(plan.YearlyPrice)
		if err == nil && price > 0 {
			return price, nil
		}
		return monthly * 12, nil
	default:
		return monthly, nil
	}
}

func moneyToCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	negative := false
	if strings.HasPrefix(value, "-") {
		negative = true
		value = strings.TrimPrefix(value, "-")
	}
	parts := strings.SplitN(value, ".", 2)
	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	var cents int64
	if len(parts) == 2 {
		frac := parts[1]
		if len(frac) > 2 {
			frac = frac[:2]
		}
		for len(frac) < 2 {
			frac += "0"
		}
		cents, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	total := yuan*100 + cents
	if negative {
		return -total, nil
	}
	return total, nil
}

func centsToMoney(cents int64) string {
	if cents < 0 {
		return "-" + centsToMoney(-cents)
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

// CreateTrafficResetOrder creates a payable order for resetting the current
// user's traffic. The traffic is not cleared until the payment callback marks
// the order as paid.
func (s *Service) CreateTrafficResetOrder(ctx context.Context, userID int64) (*OrderResult, error) {
	u, err := s.Store.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.PlanID == nil || *u.PlanID == 0 {
		return nil, httpx.NewError(http.StatusForbidden, "plan_required", "请先订阅套餐后再使用此功能")
	}
	plan, err := s.Store.FindPlanByID(ctx, *u.PlanID)
	if err != nil {
		if store.IsNoRows(err) {
			return nil, httpx.NewError(http.StatusNotFound, "plan_not_found", "套餐不存在")
		}
		return nil, err
	}
	price := strings.TrimSpace(plan.ResetTrafficPrice)
	if price == "" || price == "0" || price == "0.00" {
		return nil, httpx.NewError(http.StatusForbidden, "reset_disabled", "当前套餐未开放流量重置")
	}
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	o := &store.Order{
		OrderNo:   newOrderNo(),
		UserID:    userID,
		PlanID:    plan.ID,
		Kind:      store.OrderKindTrafficReset,
		Amount:    price,
		Currency:  "CNY",
		Status:    "pending",
		ExpiredAt: &expiresAt,
	}
	id, err := s.Store.CreateOrder(ctx, o)
	if err != nil {
		return nil, err
	}
	o.ID = id
	created, err := s.Store.FindOrderByNo(ctx, o.OrderNo)
	if err != nil {
		return nil, err
	}
	return &OrderResult{Order: created}, nil
}

func newOrderNo() string {
	rand, _ := authx.NewToken(6)
	return fmt.Sprintf("ZB%s%s", time.Now().UTC().Format("20060102150405"), rand[:8])
}

func hashRequest(payload map[string]any) string {
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ===== Payments =====

type PayResult struct {
	Existing bool           `json:"existing"`
	Payment  *store.Payment `json:"payment"`
	OrderNo  string         `json:"order_no"`
	PayURL   string         `json:"pay_url"`
}

// StartPayment creates a `pending` payment and returns a mock pay URL. Idempotent on (Idempotency-Key, scope).
func (s *Service) StartPayment(ctx context.Context, userID int64, orderNo, idempotencyKey string) (*PayResult, error) {
	o, err := s.Store.FindOrderByNo(ctx, orderNo)
	if err != nil {
		if store.IsNoRows(err) {
			return nil, httpx.NewError(http.StatusNotFound, "order_not_found", "订单不存在")
		}
		return nil, err
	}
	if o.UserID != userID {
		return nil, httpx.NewError(http.StatusForbidden, "order_owner_mismatch", "订单不属于当前用户")
	}
	if o.Status == "paid" {
		return nil, httpx.NewError(http.StatusConflict, "order_already_paid", "订单已支付")
	}

	requestHash := hashRequest(map[string]any{"user_id": userID, "order_no": orderNo})
	scope := "payments.start"

	doInsert := func() (*PayResult, error) {
		paymentNo := "PAY" + time.Now().UTC().Format("20060102150405") + strconv.FormatInt(o.ID, 10)
		p := &store.Payment{
			PaymentNo: paymentNo,
			OrderID:   o.ID,
			UserID:    o.UserID,
			Provider:  "mock",
			Amount:    o.Amount,
			Status:    "pending",
		}
		pid, err := s.Store.CreatePayment(ctx, p)
		if err != nil {
			return nil, err
		}
		p.ID = pid
		return &PayResult{
			Payment: p,
			OrderNo: o.OrderNo,
			PayURL:  fmt.Sprintf("https://example.invalid/pay/%s", paymentNo),
		}, nil
	}

	if idempotencyKey != "" {
		claimed, existing, err := s.Store.ClaimIdempotency(ctx, scope, idempotencyKey, requestHash, IdempotencyTTL)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if existing.RequestHash == nil || *existing.RequestHash != requestHash {
				return nil, httpx.NewError(http.StatusConflict, "idempotency_mismatch", "Idempotency-Key 与原始请求不一致")
			}
			if existing.ResponseBody != nil {
				var prior PayResult
				if err := json.Unmarshal([]byte(*existing.ResponseBody), &prior); err == nil {
					prior.Existing = true
					return &prior, nil
				}
			}
			return nil, httpx.NewError(http.StatusConflict, "idempotency_in_progress", "请求处理中，请稍后重试")
		}
		res, err := doInsert()
		if err != nil {
			return nil, err
		}
		body, _ := json.Marshal(res)
		_ = s.Store.CompleteIdempotency(ctx, claimed.ID, string(body), "succeeded")
		return res, nil
	}
	return doInsert()
}

// HandleMockCallback marks payment + order as paid and activates the user. It is
// keyed on (provider, provider_event_id) so duplicates are rejected.
func (s *Service) HandleMockCallback(ctx context.Context, eventID, orderNo, paymentNo, headers, body string) error {
	if eventID == "" || orderNo == "" || paymentNo == "" {
		return httpx.NewError(http.StatusBadRequest, "bad_request", "回调字段缺失")
	}
	_, dup, err := s.Store.CreatePaymentCallback(ctx, "mock", eventID, orderNo, headers, body)
	if err != nil {
		return err
	}
	if dup {
		return httpx.NewError(http.StatusConflict, "callback_duplicate", "回调事件已处理")
	}

	now := time.Now().UTC()
	if err := s.Store.MarkPaymentSuccess(ctx, paymentNo, "mock-"+eventID, now); err != nil {
		return err
	}
	if err := s.Store.MarkOrderPaid(ctx, orderNo, now); err != nil {
		return err
	}

	o, err := s.Store.FindOrderByNo(ctx, orderNo)
	if err != nil {
		return err
	}
	if o.Kind == store.OrderKindTrafficReset {
		return s.completeTrafficResetOrder(ctx, o)
	}
	plan, err := s.Store.FindPlanByID(ctx, o.PlanID)
	if err != nil {
		return err
	}
	if err := s.Store.ActivateUserPlanPeriod(ctx, o.UserID, plan, o.BillingPeriod); err != nil {
		return err
	}
	// Provision node_users for every currently active node so the new
	// subscriber sees nodes immediately.
	nodes, err := s.Store.ListActiveNodes(ctx)
	if err != nil {
		return err
	}
	clientID, err := newClientID()
	if err != nil {
		return err
	}
	for _, n := range nodes {
		if err := s.Store.EnsureNodeUserWithLimits(ctx, o.UserID, n.ID, clientID, n.Protocol, 0, plan.DeviceLimit); err != nil {
			return err
		}
	}
	return nil
}

// ActivateByCallback is called from the generic payment webhook handler. It
// marks the payment + order as paid and activates the user's plan — same logic
// as HandleMockCallback but provider-agnostic.
func (s *Service) ActivateByCallback(ctx context.Context, orderNo, provider, providerTradeNo string) error {
	o, err := s.Store.FindOrderByNo(ctx, orderNo)
	if err != nil {
		return err
	}
	if o.Status == "paid" {
		return nil // already activated, idempotent
	}
	now := time.Now().UTC()
	paymentNo := "PAY-" + provider + "-" + orderNo
	_ = s.Store.MarkPaymentSuccess(ctx, paymentNo, providerTradeNo, now)
	if err := s.Store.MarkOrderPaid(ctx, orderNo, now); err != nil {
		return err
	}
	if o.Kind == store.OrderKindTrafficReset {
		return s.completeTrafficResetOrder(ctx, o)
	}
	plan, err := s.Store.FindPlanByID(ctx, o.PlanID)
	if err != nil {
		return err
	}
	if err := s.Store.ActivateUserPlanPeriod(ctx, o.UserID, plan, o.BillingPeriod); err != nil {
		return err
	}
	nodes, err := s.Store.ListActiveNodes(ctx)
	if err != nil {
		return err
	}
	clientID, err := newClientID()
	if err != nil {
		return err
	}
	for _, n := range nodes {
		if err := s.Store.EnsureNodeUserWithLimits(ctx, o.UserID, n.ID, clientID, n.Protocol, 0, plan.DeviceLimit); err != nil {
			return err
		}
	}
	// Notify user: payment success
	s.Store.NotifyUser(ctx, o.UserID, "payment_success",
		"支付成功", "您的订单 "+orderNo+" 已支付成功，套餐已激活",
		"/dashboard")
	return nil
}

func (s *Service) completeTrafficResetOrder(ctx context.Context, o *store.Order) error {
	if err := s.Store.ResetUserTraffic(ctx, o.UserID); err != nil {
		return err
	}
	s.Store.NotifyUser(ctx, o.UserID, "traffic_reset",
		"流量重置成功", "您的流量重置订单 "+o.OrderNo+" 已支付成功，已用流量已清零。",
		"/dashboard")
	return nil
}
