package agentauth_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/agentauth"
	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
)

func setupAgentRouter(t *testing.T) (*gin.Engine, *store.Store, int64, string) {
	t.Helper()
	s := testsupport.NewStore(t)
	ctx := context.Background()

	nodeID, secret, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name: "T", Host: "h", Port: 443, Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/agent/v1")
	g.Use(agentauth.HMAC(s))
	g.POST("/heartbeat", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r, s, nodeID, secret
}

func sign(t *testing.T, method, path, secret string, nodeID int64, body []byte, ts int64, nonce string) http.Header {
	t.Helper()
	bodyHash := sha256.Sum256(body)
	bh := hex.EncodeToString(bodyHash[:])
	keySum := sha256.Sum256([]byte(secret))
	key := hex.EncodeToString(keySum[:])
	msg := strconv.FormatInt(nodeID, 10) + "|" + strconv.FormatInt(ts, 10) + "|" + nonce + "|" + bh + "|" + method + "|" + path
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set(agentauth.HeaderNodeID, strconv.FormatInt(nodeID, 10))
	h.Set(agentauth.HeaderTimestamp, strconv.FormatInt(ts, 10))
	h.Set(agentauth.HeaderNonce, nonce)
	h.Set(agentauth.HeaderBodyHash, bh)
	h.Set(agentauth.HeaderSignature, sig)
	return h
}

func TestHMACMissingHeadersIs401(t *testing.T) {
	r, _, _, _ := setupAgentRouter(t)
	req := httptest.NewRequest("POST", "/api/agent/v1/heartbeat", bytes.NewReader([]byte("{}")))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHMACValidRequestIs200(t *testing.T) {
	r, _, nodeID, secret := setupAgentRouter(t)
	body, _ := json.Marshal(map[string]any{"agent_version": "0.1.0"})
	headers := sign(t, "POST", "/api/agent/v1/heartbeat", secret, nodeID, body, time.Now().UTC().Unix(), "n1")
	req := httptest.NewRequest("POST", "/api/agent/v1/heartbeat", bytes.NewReader(body))
	for k, vs := range headers {
		req.Header[k] = vs
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHMACReplayIsRejected(t *testing.T) {
	r, _, nodeID, secret := setupAgentRouter(t)
	body, _ := json.Marshal(map[string]any{})
	ts := time.Now().UTC().Unix()
	nonce := "replay-1"
	doReq := func() int {
		headers := sign(t, "POST", "/api/agent/v1/heartbeat", secret, nodeID, body, ts, nonce)
		req := httptest.NewRequest("POST", "/api/agent/v1/heartbeat", bytes.NewReader(body))
		for k, vs := range headers {
			req.Header[k] = vs
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}
	if c := doReq(); c != http.StatusOK {
		t.Fatalf("first request: want 200, got %d", c)
	}
	if c := doReq(); c != http.StatusUnauthorized {
		t.Fatalf("replay: want 401, got %d", c)
	}
}

func TestHMACOutOfWindowTimestamp(t *testing.T) {
	r, _, nodeID, secret := setupAgentRouter(t)
	body := []byte("{}")
	old := time.Now().UTC().Add(-time.Hour).Unix()
	headers := sign(t, "POST", "/api/agent/v1/heartbeat", secret, nodeID, body, old, "old-1")
	req := httptest.NewRequest("POST", "/api/agent/v1/heartbeat", bytes.NewReader(body))
	for k, vs := range headers {
		req.Header[k] = vs
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for stale ts, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHMACTamperedBodyHash(t *testing.T) {
	r, _, nodeID, secret := setupAgentRouter(t)
	body := []byte(`{"a":1}`)
	headers := sign(t, "POST", "/api/agent/v1/heartbeat", secret, nodeID, body, time.Now().UTC().Unix(), "tamper-1")
	headers.Set(agentauth.HeaderBodyHash, hex.EncodeToString(sha256Sum([]byte("different"))))
	req := httptest.NewRequest("POST", "/api/agent/v1/heartbeat", bytes.NewReader(body))
	for k, vs := range headers {
		req.Header[k] = vs
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		b, _ := io.ReadAll(rr.Body)
		t.Fatalf("want 401 for body hash mismatch, got %d body=%s", rr.Code, b)
	}
}

func sha256Sum(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

// authx.HashToken sanity check — used as the HMAC key derivation.
func TestAuthxHashTokenStable(t *testing.T) {
	a := authx.HashToken("hello")
	b := authx.HashToken("hello")
	if a != b {
		t.Fatal("HashToken not stable")
	}
}
