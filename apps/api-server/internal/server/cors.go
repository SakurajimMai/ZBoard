package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// cors returns a middleware that handles CORS based on the allowed origins list.
// If origins contains "*", all origins are allowed (and credentials are NOT sent
// per browser spec — `Access-Control-Allow-Origin: *` is incompatible with
// `Access-Control-Allow-Credentials: true`).
// If origins is empty, CORS headers are not set (same-origin only).
func cors(origins []string) gin.HandlerFunc {
	if len(origins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}

	allowAll := false
	set := make(map[string]bool, len(origins))
	for _, o := range origins {
		if o == "*" {
			allowAll = true
			continue
		}
		set[strings.TrimRight(o, "/")] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowed := allowAll || set[strings.TrimRight(origin, "/")]
		if !allowed {
			c.Next()
			return
		}

		// Always vary on Origin so caches don't poison cross-origin responses.
		c.Header("Vary", "Origin")

		if allowAll {
			// Wildcard mode: per spec, do NOT set Allow-Credentials. Echoing the
			// origin would defeat the wildcard intent (and hostile origins would
			// receive credential-bearing responses).
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Zboard-Node-Id, X-Zboard-Timestamp, X-Zboard-Nonce, X-Zboard-Body-SHA256, X-Zboard-Signature")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
