package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/payment"
	"github.com/zboard/api-server/internal/payment/registry"
	"github.com/zboard/api-server/internal/store"
)

type paypalCapturer interface {
	CaptureOrder(ctx context.Context, orderID string) (*payment.CallbackData, error)
}

// paymentCallback dispatches provider webhooks and completes the paid order.
// Plan orders activate subscriptions; traffic-reset orders clear used traffic.
func paymentCallback(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		providerName := strings.TrimSpace(c.Param("provider"))
		if providerName == "" || strings.EqualFold(providerName, "mock") {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "unknown_provider", "支付方式不可用"))
			return
		}
		prov, err := d.Payments.Get(providerName)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "unknown_provider", "支付方式不可用"))
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_body", "无法读取请求体"))
			return
		}
		headers := map[string]string{}
		for k := range c.Request.Header {
			headers[k] = c.GetHeader(k)
		}

		data, err := prov.VerifyCallback(c.Request.Context(), headers, body)
		if err != nil {
			// 详细验签失败原因只写入审计日志；公开响应保持泛化，避免泄露
			// expected sign、密钥材料或内部字段名。
			_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
				ActorType: "system", Action: "payment.callback_failed",
				ResourceType: "provider", ResourceID: providerName,
				Detail: err.Error(), IP: c.ClientIP(),
			})
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "callback_verify_failed", "回调签名校验失败"))
			return
		}

		// Deduplicate: record the callback; if already processed, return OK.
		cbID, dup, err := d.Store.CreatePaymentCallback(c.Request.Context(),
			providerName, data.ProviderOrderNo, data.OrderNo, "", data.RawBody)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if dup {
			respondCallback(c, providerName)
			return
		}

		if data.Status != "success" {
			_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, "status="+data.Status)
			respondCallback(c, providerName)
			return
		}

		// 签名只能证明消息来自支付网关，不能证明用户支付了订单金额。
		// 激活前再次比对落库订单金额，阻止低金额交易冒充高金额订单。
		o, ferr := d.Store.FindOrderByNo(c.Request.Context(), data.OrderNo)
		if ferr != nil {
			if store.IsNoRows(ferr) {
				_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, "order_not_found")
				httpx.Fail(c, httpx.NewError(http.StatusNotFound, "order_not_found", "订单不存在"))
				return
			}
			httpx.Fail(c, ferr)
			return
		}
		if err := verifyCallbackAmount(o.Amount, data.Amount); err != nil {
			_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, "amount_mismatch: "+err.Error())
			_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
				ActorType: "system", Action: "payment.callback_amount_mismatch",
				ResourceType: "order", ResourceID: data.OrderNo,
				Detail: providerName + ": " + err.Error(), IP: c.ClientIP(),
			})
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "callback_amount_mismatch", "回调金额与订单不一致"))
			return
		}

		// Activate: mark payment + order paid, activate user plan.
		if err := d.Biz.ActivateByCallback(c.Request.Context(), data.OrderNo, providerName, data.ProviderOrderNo, data.Amount); err != nil {
			_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, err.Error())
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, "")
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "system", Action: "payment.callback_success",
			ResourceType: "order", ResourceID: data.OrderNo,
			Detail: providerName + ":" + data.ProviderOrderNo, IP: c.ClientIP(),
		})
		respondCallback(c, providerName)
	}
}

// paypalReturn captures an approved PayPal order after the user returns from
// the PayPal approval page. Webhooks may also complete the order; this route
// makes the browser redirect path deterministic for smaller deployments.
func paypalReturn(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "缺少 PayPal token"))
			return
		}
		providerName := strings.TrimSpace(c.DefaultQuery("provider", "paypal"))
		if providerName == "" {
			providerName = "paypal"
		}
		if strings.EqualFold(providerName, "mock") {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "unknown_provider", "支付方式不可用"))
			return
		}
		prov, err := d.Payments.Get(providerName)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "unknown_provider", "支付方式不可用"))
			return
		}
		capturer, ok := prov.(paypalCapturer)
		if !ok {
			httpx.Fail(c, httpx.NewError(http.StatusInternalServerError, "bad_provider", "PayPal provider 不支持 capture"))
			return
		}
		data, err := capturer.CaptureOrder(c.Request.Context(), token)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadGateway, "paypal_capture_failed", "PayPal 支付确认失败"))
			return
		}
		eventID := data.ProviderOrderNo
		if eventID == "" {
			eventID = token
		}
		cbID, dup, err := d.Store.CreatePaymentCallback(c.Request.Context(), providerName, eventID, data.OrderNo, "", data.RawBody)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if dup {
			c.Redirect(http.StatusFound, frontendReturnURL(c, d))
			return
		}
		if data.Status == "success" && data.OrderNo != "" {
			o, ferr := d.Store.FindOrderByNo(c.Request.Context(), data.OrderNo)
			if ferr != nil {
				if store.IsNoRows(ferr) {
					_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, "order_not_found")
					httpx.Fail(c, httpx.NewError(http.StatusNotFound, "order_not_found", "订单不存在"))
					return
				}
				httpx.Fail(c, ferr)
				return
			}
			if err := verifyCallbackAmount(o.Amount, data.Amount); err != nil {
				_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, "amount_mismatch: "+err.Error())
				_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
					ActorType: "system", Action: "payment.callback_amount_mismatch",
					ResourceType: "order", ResourceID: data.OrderNo,
					Detail: providerName + ": " + err.Error(), IP: c.ClientIP(),
				})
				httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "callback_amount_mismatch", "回调金额与订单不一致"))
				return
			}
			if err := d.Biz.ActivateByCallback(c.Request.Context(), data.OrderNo, providerName, data.ProviderOrderNo, data.Amount); err != nil {
				_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, err.Error())
				httpx.Fail(c, err)
				return
			}
		}
		_ = d.Store.MarkCallbackProcessed(c.Request.Context(), cbID, "")
		c.Redirect(http.StatusFound, frontendReturnURL(c, d))
	}
}

// respondCallback sends the provider-expected success response.
// EasyPay expects plain text "success"; others accept JSON.
func respondCallback(c *gin.Context, provider string) {
	switch provider {
	case "epay":
		c.String(http.StatusOK, "success")
	default:
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// verifyCallbackAmount 校验网关回调金额必须等于订单金额。两者都是十进制
// 字符串，解析为分后比较，避免浮点误差；空金额按缺失数据拒绝。
func verifyCallbackAmount(orderAmount, callbackAmount string) error {
	want, err := decimalToCents(orderAmount)
	if err != nil {
		return fmt.Errorf("order amount %q: %w", orderAmount, err)
	}
	got, err := decimalToCents(callbackAmount)
	if err != nil {
		return fmt.Errorf("callback amount %q: %w", callbackAmount, err)
	}
	if got != want {
		return fmt.Errorf("paid %d != order %d (cents)", got, want)
	}
	return nil
}

// decimalToCents 把 "9"、"9.9"、"9.90" 这类金额解析为分，并拒绝负数和非数字。
func decimalToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	if strings.HasPrefix(s, "-") {
		return 0, errors.New("negative")
	}
	parts := strings.SplitN(s, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	frac := "00"
	if len(parts) == 2 {
		frac = parts[1]
		if len(frac) == 1 {
			frac += "0"
		}
		if len(frac) > 2 {
			frac = frac[:2]
		}
	}
	cents, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, err
	}
	return whole*100 + cents, nil
}

// createPaymentWithProvider handles POST /api/v1/orders/:order_no/pay with a
// real provider when a provider query param is given.
func createPaymentWithProvider(d Deps, reg *registry.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderNo := c.Param("order_no")
		uid := c.MustGet(ctxUserIDKey).(int64)
		providerName := strings.TrimSpace(c.DefaultQuery("provider", ""))
		payType := c.DefaultQuery("pay_type", "alipay")

		if providerName == "" {
			// 必须显式选择真实支付渠道，避免请求落到任何开发期支付路径。
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "provider_required", "缺少 provider 参数"))
			return
		}

		if strings.EqualFold(providerName, "mock") {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "unknown_provider", "支付方式不可用"))
			return
		}
		if reg == nil {
			httpx.Fail(c, httpx.NewError(http.StatusInternalServerError, "payment_registry_missing", "支付服务未初始化"))
			return
		}

		prov, err := reg.Get(providerName)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "unknown_provider", "支付方式不可用"))
			return
		}

		o, err := d.Store.FindOrderByNo(c.Request.Context(), orderNo)
		if err != nil {
			if store.IsNoRows(err) {
				httpx.Fail(c, httpx.NewError(http.StatusNotFound, "order_not_found", "订单不存在"))
				return
			}
			httpx.Fail(c, err)
			return
		}
		if o.UserID != uid {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "order_owner_mismatch", "订单不属于当前用户"))
			return
		}
		if o.Status == "paid" {
			httpx.Fail(c, httpx.NewError(http.StatusConflict, "order_already_paid", "订单已支付"))
			return
		}
		// 过期订单不得发起真实支付:网关一旦收款,30 分钟窗口外的回调会被
		// 视为过期而无法开通套餐,用户白白损失。让用户重新下单更安全。
		if o.ExpiredAt != nil && !o.ExpiredAt.After(time.Now().UTC()) {
			httpx.Fail(c, httpx.NewError(http.StatusConflict, "order_expired", "订单已过期,请重新下单"))
			return
		}

		// Build callback URLs from the configured public origin. Real gateways
		// call NotifyURL server-to-server and redirect the browser to ReturnURL,
		// so both must be publicly reachable. Refuse to create the payment when
		// no public origin is configured rather than emitting an unreachable
		// localhost URL that strands the payment after the user is charged.
		baseURL := apiBaseURL(c, d)
		if baseURL == "" {
			httpx.Fail(c, httpx.NewError(http.StatusServiceUnavailable, "site_url_unconfigured", "未配置可公网访问的站点地址,请联系管理员在后台设置 site_url 后再支付"))
			return
		}
		notifyURL := baseURL + "/api/v1/payments/" + providerName + "/callback"
		returnURL := frontendReturnURL(c, d)
		if providerName == "paypal" || paymentProviderType(c.Request.Context(), d.Store, providerName) == "paypal" {
			returnURL = baseURL + "/api/v1/payments/paypal/return?provider=" + url.QueryEscape(providerName)
		}

		resp, err := prov.CreatePayment(c.Request.Context(), payment.CreateRequest{
			OrderNo:   o.OrderNo,
			Amount:    o.Amount,
			Currency:  o.Currency,
			Subject:   "Zboard - " + o.OrderNo,
			PayType:   payType,
			NotifyURL: notifyURL,
			ReturnURL: returnURL,
			CancelURL: frontendReturnURL(c, d),
			ClientIP:  c.ClientIP(),
			UserID:    uid,
		})
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadGateway, "payment_create_failed", "支付创建失败"))
			return
		}

		// Record the payment row.
		p := &store.Payment{
			PaymentNo: "PAY-" + providerName + "-" + orderNo,
			OrderID:   o.ID,
			UserID:    uid,
			Provider:  providerName,
			Amount:    o.Amount,
			Status:    "pending",
		}
		pid, err := d.Store.CreatePayment(c.Request.Context(), p)
		if err != nil && !store.IsUniqueViolation(err) {
			httpx.Fail(c, err)
			return
		}
		_ = pid

		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user", ActorID: ptrInt64(uid),
			Action: "payment.create", ResourceType: "order", ResourceID: orderNo,
			Detail: providerName, IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})

		httpx.OK(c, gin.H{
			"provider":   providerName,
			"pay_url":    resp.PayURL,
			"qr_code":    resp.QRCode,
			"order_no":   o.OrderNo,
			"payment_id": resp.ProviderOrderNo,
		})
	}
}

func paymentProviderType(ctx context.Context, st *store.Store, providerName string) string {
	if st == nil || strings.TrimSpace(providerName) == "" {
		return ""
	}
	row, err := st.FindPaymentProviderByName(ctx, providerName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(row.ProviderType)
}

// apiBaseURL returns the configured public origin used to build gateway
// callback/return URLs. It returns "" when no public origin is configured;
// callers must refuse to create a real payment in that case rather than
// emitting an unreachable localhost URL that would strand a paid order.
func apiBaseURL(c *gin.Context, d Deps) string {
	return configuredSiteOrigin(c.Request.Context(), d.Store)
}

func frontendReturnURL(c *gin.Context, d Deps) string {
	base := configuredSiteOrigin(c.Request.Context(), d.Store)
	if base == "" {
		// No public origin configured. Fall back to a host-relative path so the
		// browser redirect still lands on the serving host instead of an
		// unreachable localhost address.
		return "/dashboard"
	}
	return base + "/dashboard"
}

func configuredSiteOrigin(ctx context.Context, st *store.Store) string {
	if st == nil {
		return ""
	}
	value, err := st.GetSetting(ctx, "site_url", "")
	if err != nil {
		return ""
	}
	return normalizePublicOrigin(value)
}

func normalizePublicOrigin(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.Path = ""
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/")
}
