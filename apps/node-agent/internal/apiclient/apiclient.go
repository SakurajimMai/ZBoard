// Package apiclient is the HMAC-signed HTTP client the Agent uses to talk to
// the Zboard control plane. Signing matches the server's `internal/agentauth`:
// HMAC-SHA256(`${node_id}|${ts}|${nonce}|${body_sha256}|${method}|${path}`,
// key=hex(sha256(node_secret))).
package apiclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	HeaderNodeID    = "X-Zboard-Node-Id"
	HeaderTimestamp = "X-Zboard-Timestamp"
	HeaderNonce     = "X-Zboard-Nonce"
	HeaderBodyHash  = "X-Zboard-Body-SHA256"
	HeaderSignature = "X-Zboard-Signature"
)

type Client struct {
	BaseURL    string
	NodeID     int64
	NodeSecret string
	HTTP       *http.Client
}

func New(baseURL string, nodeID int64, nodeSecret string) *Client {
	return &Client{
		BaseURL:    baseURL,
		NodeID:     nodeID,
		NodeSecret: nodeSecret,
		HTTP:       &http.Client{Timeout: 30 * time.Second},
	}
}

// Do POSTs `body` (which is JSON-encoded) to `path` with HMAC headers. The
// signed `path` is the URL path only, no query string — matching the server.
func (c *Client) Do(ctx context.Context, path string, body any, out any) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	nonce, err := newNonce()
	if err != nil {
		return err
	}
	ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	bodyHash := sha256Hex(raw)

	keyHash := sha256Hex([]byte(c.NodeSecret))
	msg := strconv.FormatInt(c.NodeID, 10) + "|" + ts + "|" + nonce + "|" + bodyHash + "|" + http.MethodPost + "|" + u.Path
	mac := hmac.New(sha256.New, []byte(keyHash))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderNodeID, strconv.FormatInt(c.NodeID, 10))
	req.Header.Set(HeaderTimestamp, ts)
	req.Header.Set(HeaderNonce, nonce)
	req.Header.Set(HeaderBodyHash, bodyHash)
	req.Header.Set(HeaderSignature, sig)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("control plane %s -> %d: %s", path, resp.StatusCode, string(respBody))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode %s: %w (body=%s)", path, err, respBody)
		}
	}
	return nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func newNonce() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
