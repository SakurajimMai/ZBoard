package server

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
)

type generateRealityBody struct {
	ServerName string `json:"server_name"`
}

func adminGenerateRealityConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body generateRealityBody
		_ = c.ShouldBindJSON(&body)
		serverName := strings.TrimSpace(body.ServerName)
		if serverName == "" {
			serverName = "www.cloudflare.com"
		}

		privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		shortIDBytes := make([]byte, 8)
		if _, err := rand.Read(shortIDBytes); err != nil {
			httpx.Fail(c, err)
			return
		}

		httpx.OK(c, gin.H{
			"reality_server_name": serverName,
			"reality_dest":        serverName + ":443",
			"reality_public_key":  base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes()),
			"reality_private_key": base64.RawURLEncoding.EncodeToString(privateKey.Bytes()),
			"reality_short_id":    hex.EncodeToString(shortIDBytes),
		})
	}
}
