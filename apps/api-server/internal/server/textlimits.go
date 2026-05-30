package server

import (
	"net/http"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
)

// User-supplied free-text limits. These are deliberately generous (real users
// rarely hit them) but strict enough to block large XSS payloads, accidental
// log floods, and stored data that bloats every list response.
const (
	maxTitleRunes   = 200
	maxSummaryRunes = 1000
	maxContentRunes = 50_000
)

// validateTextLen returns nil if `s` is at most `max` runes (UTF-8 codepoints,
// not bytes — matches user-visible "characters" for CJK and emoji input). On
// over-length input it writes a 400 to the gin context and returns false so
// the caller can early-return.
func validateTextLen(c *gin.Context, field, s string, max int) bool {
	if utf8.RuneCountInString(s) > max {
		httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "field_too_long",
			field+" 超过最大长度"))
		return false
	}
	return true
}
