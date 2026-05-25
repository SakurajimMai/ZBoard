package server

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/payment"
	"github.com/zboard/api-server/internal/payment/registry"
	"github.com/zboard/api-server/internal/store"
)

// paymentCallback dispatches provider webhooks and completes the paid order.
// Plan orders activate subscriptions; traffic-reset orders clear used traffic.
func paymentCallback(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		providerName := c.Param("provider")
		prov, err := d.Payments.Get(providerName)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "unknown_provider", err.Error()))
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
			_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
				ActorType: "system", Action: "payment.callback_failed",
				ResourceType: "provider", ResourceID: providerName,
				Detail: err.Error(), IP: c.ClientIP(),
			})
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "callback_verify_failed", err.Error()))
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

		// Activate: mark payment + order paid, activate user plan.
		if err := d.Biz.ActivateByCallback(c.Request.Context(), data.OrderNo, providerName, data.ProviderOrderNo); err != nil {
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

// createPaymentWithProvider handles POST /api/v1/orders/:order_no/pay with a
// real provider when a provider query param is given.
func createPaymentWithProvider(d Deps, reg *registry.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderNo := c.Param("order_no")
		uid := c.MustGet(ctxUserIDKey).(int64)
		providerName := c.DefaultQuery("provider", "")
		payType := c.DefaultQuery("pay_type", "alipay")

		if providerName == "" {
			// Fall back to the existing mock flow.
			payOrder(d)(c)
			return
		}

		prov, err := reg.Get(providerName)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "unknown_provider", err.Error()))
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

		// Build callback URLs from the request host.
		scheme := "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
		baseURL := scheme + "://" + c.Request.Host
		notifyURL := baseURL + "/api/v1/payments/" + providerName + "/callback"
		returnURL := baseURL + "/dashboard"

		resp, err := prov.CreatePayment(c.Request.Context(), payment.CreateRequest{
			OrderNo:   o.OrderNo,
			Amount:    o.Amount,
			Currency:  o.Currency,
			Subject:   "Zboard - " + o.OrderNo,
			PayType:   payType,
			NotifyURL: notifyURL,
			ReturnURL: returnURL,
			ClientIP:  c.ClientIP(),
			UserID:    uid,
		})
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadGateway, "payment_create_failed", err.Error()))
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
