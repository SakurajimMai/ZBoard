package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFailDoesNotExposeRawInternalError(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/boom", func(c *gin.Context) {
		Fail(c, errors.New("sql: secret connection detail"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "secret connection detail") || strings.Contains(w.Body.String(), "sql:") {
		t.Fatalf("internal detail leaked: %s", w.Body.String())
	}
}
