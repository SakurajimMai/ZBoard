package agentauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

const (
	HeaderNodeID    = "X-Zboard-Node-Id"
	HeaderTimestamp = "X-Zboard-Timestamp"
	HeaderNonce     = "X-Zboard-Nonce"
	HeaderBodyHash  = "X-Zboard-Body-SHA256"
	HeaderSignature = "X-Zboard-Signature"

	WindowSeconds   int64 = 300
	NonceCtxKey           = "zb_agent_nonce"
	NodeIDCtxKey          = "zb_agent_node_id"
	BodyCtxKey            = "zb_agent_body"
)

// HMAC verifies all five headers and stores the captured body for handlers.
// The signing material is `${node_id}|${ts}|${nonce}|${body_sha256}|${method}|${path}`.
func HMAC(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeIDStr := c.GetHeader(HeaderNodeID)
		ts := c.GetHeader(HeaderTimestamp)
		nonce := c.GetHeader(HeaderNonce)
		bodyHash := c.GetHeader(HeaderBodyHash)
		signature := c.GetHeader(HeaderSignature)

		if nodeIDStr == "" || ts == "" || nonce == "" || bodyHash == "" || signature == "" {
			httpx.Fail(c, httpx.NewError(http.StatusUnauthorized, "agent_signature_missing", "Agent 签名头缺失"))
			return
		}
		nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusUnauthorized, "agent_node_invalid", "节点 ID 不合法"))
			return
		}
		tsInt, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusUnauthorized, "agent_timestamp_invalid", "时间戳不合法"))
			return
		}
		now := time.Now().UTC().Unix()
		if tsInt < now-WindowSeconds || tsInt > now+WindowSeconds {
			httpx.Fail(c, httpx.NewError(http.StatusUnauthorized, "agent_timestamp_window", "时间戳超出窗口"))
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_body", "无法读取请求体"))
			return
		}
		_ = c.Request.Body.Close()

		gotHash := sha256Hex(body)
		if !hmac.Equal([]byte(gotHash), []byte(bodyHash)) {
			httpx.Fail(c, httpx.NewError(http.StatusUnauthorized, "agent_body_hash_mismatch", "请求体哈希不一致"))
			return
		}

		// Look up the agent secret hash; that is our HMAC key.
		secretHash, err := s.GetAgentSecretHash(c.Request.Context(), nodeID)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusUnauthorized, "agent_unknown", "节点未注册"))
			return
		}

		message := nodeIDStr + "|" + ts + "|" + nonce + "|" + bodyHash + "|" + c.Request.Method + "|" + c.Request.URL.Path
		mac := hmac.New(sha256.New, []byte(secretHash))
		mac.Write([]byte(message))
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(signature)) {
			httpx.Fail(c, httpx.NewError(http.StatusUnauthorized, "agent_signature_invalid", "签名校验失败"))
			return
		}

		// Replay protection: insert (node_id, nonce). Duplicates are rejected.
		if err := s.InsertAgentNonce(c.Request.Context(), nodeID, nonce, tsInt); err != nil {
			if store.IsUniqueViolation(err) {
				httpx.Fail(c, httpx.NewError(http.StatusUnauthorized, "agent_replay", "重放保护：nonce 已使用"))
				return
			}
			httpx.Fail(c, err)
			return
		}

		c.Set(NodeIDCtxKey, nodeID)
		c.Set(NonceCtxKey, nonce)
		c.Set(BodyCtxKey, body)
		c.Next()
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ErrUnknownNode signals the node has no agent secret persisted.
var ErrUnknownNode = errors.New("unknown agent node")
