// Smoke test for the Agent HMAC flow. Not part of the production binary.
//go:build ignore

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

func main() {
	base := flag.String("base", "http://127.0.0.1:3216", "API base URL")
	nodeID := flag.Int64("node", 1, "Node ID")
	secret := flag.String("secret", "", "Plaintext node_secret returned at create time")
	flag.Parse()

	if *secret == "" {
		fmt.Println("missing -secret")
		return
	}
	// Server stores sha256(node_secret) as the HMAC key
	sum := sha256.Sum256([]byte(*secret))
	hmacKey := hex.EncodeToString(sum[:])

	send := func(method, path string, body any) {
		raw, _ := json.Marshal(body)
		nonceB := make([]byte, 8)
		_, _ = io.ReadFull(rng(), nonceB)
		nonce := hex.EncodeToString(nonceB)
		ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
		bodyHash := sha256.Sum256(raw)
		bh := hex.EncodeToString(bodyHash[:])
		message := strconv.FormatInt(*nodeID, 10) + "|" + ts + "|" + nonce + "|" + bh + "|" + method + "|" + path
		mac := hmac.New(sha256.New, []byte(hmacKey))
		mac.Write([]byte(message))
		sig := hex.EncodeToString(mac.Sum(nil))

		req, _ := http.NewRequest(method, *base+path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Zboard-Node-Id", strconv.FormatInt(*nodeID, 10))
		req.Header.Set("X-Zboard-Timestamp", ts)
		req.Header.Set("X-Zboard-Nonce", nonce)
		req.Header.Set("X-Zboard-Body-SHA256", bh)
		req.Header.Set("X-Zboard-Signature", sig)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("ERR", method, path, err)
			return
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		fmt.Printf(">>> %s %s -> %d %s\n", method, path, resp.StatusCode, string(out))
	}

	send("POST", "/api/agent/v1/register", map[string]any{
		"agent_version": "0.1.0", "os_info": "linux/amd64", "runtime_info": "xray-1.8",
	})
	send("POST", "/api/agent/v1/heartbeat", map[string]any{
		"agent_version": "0.1.0", "runtime_status": "running", "system_load": "0.5",
	})
	send("POST", "/api/agent/v1/tasks/pull", map[string]any{})
}

// rng returns a non-crypto rand source that's adequate for nonce uniqueness in
// a single-process smoke test.
func rng() io.Reader {
	return &timeRand{}
}

type timeRand struct{ n int64 }

func (t *timeRand) Read(p []byte) (int, error) {
	t.n++
	v := time.Now().UnixNano() ^ t.n*2862933555777941757
	for i := range p {
		v ^= v << 13
		v ^= int64(uint64(v) >> 7)
		v ^= v << 17
		p[i] = byte(v)
	}
	return len(p), nil
}
