package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// defaultMaxBodyBytes caps every request body to 1 MiB. JSON payloads in this
// API are tiny (orders, settings, ticket text); anything larger is either a
// misconfigured client or an attempt to exhaust memory in handlers that ReadAll
// before authenticating (e.g. agent HMAC, payment webhook signature checks).
// Endpoints that need more (file uploads, bulk imports) should override per-
// route via a dedicated middleware — none exist today.
const defaultMaxBodyBytes int64 = 1 << 20

// maxBodyBytes wraps c.Request.Body in http.MaxBytesReader. Reads beyond the
// limit return an error which the downstream handler converts to a 4xx — the
// goroutine never buffers more than `limit` bytes.
func maxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
