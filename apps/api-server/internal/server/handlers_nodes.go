package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

type createNodeBody struct {
	Name              string `json:"name" binding:"required"`
	Region            string `json:"region"`
	Host              string `json:"host" binding:"required"`
	Port              int    `json:"port" binding:"required"`
	Protocol          string `json:"protocol"`
	Transport         string `json:"transport"`
	Security          string `json:"security"`
	RuntimeType       string `json:"runtime_type"`
	WSPath            string `json:"ws_path"`
	WSHost            string `json:"ws_host"`
	GRPCServiceName   string `json:"grpc_service_name"`
	SNI               string `json:"sni"`
	Fingerprint       string `json:"fingerprint"`
	RealityPublicKey  string `json:"reality_public_key"`
	RealityShortID    string `json:"reality_short_id"`
	RealityServerName string `json:"reality_server_name"`
	Flow              string `json:"flow"`
	ALPN              string `json:"alpn"`
	MuxEnabled        int    `json:"mux_enabled"`
	SSMethod          string `json:"ss_method"`
	RealityPrivateKey string `json:"reality_private_key"`
	RealityDest       string `json:"reality_dest"`
	ObfsPassword      string `json:"obfs_password"`
	CongestionControl string `json:"congestion_control"`
	UpMbps            int    `json:"up_mbps"`
	DownMbps          int    `json:"down_mbps"`
	PortRange         string `json:"port_range"`
}

func adminCreateNode(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createNodeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		nodeID, secret, err := d.Store.CreateNode(c.Request.Context(), store.CreateNodeInput{
			Name:              body.Name,
			Region:            body.Region,
			Host:              body.Host,
			Port:              body.Port,
			Protocol:          body.Protocol,
			Transport:         body.Transport,
			Security:          body.Security,
			RuntimeType:       body.RuntimeType,
			WSPath:            body.WSPath,
			WSHost:            body.WSHost,
			GRPCServiceName:   body.GRPCServiceName,
			SNI:               body.SNI,
			Fingerprint:       body.Fingerprint,
			RealityPublicKey:  body.RealityPublicKey,
			RealityShortID:    body.RealityShortID,
			RealityServerName: body.RealityServerName,
			Flow:              body.Flow,
			ALPN:              body.ALPN,
			MuxEnabled:        body.MuxEnabled,
			SSMethod:          body.SSMethod,
			RealityPrivateKey: body.RealityPrivateKey,
			RealityDest:       body.RealityDest,
			ObfsPassword:      body.ObfsPassword,
			CongestionControl: body.CongestionControl,
			UpMbps:            body.UpMbps,
			DownMbps:          body.DownMbps,
			PortRange:         body.PortRange,
		})
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		// Backfill node_users for currently active users so new nodes light up
		// in their subscription immediately.
		ids, err := d.Store.ListUserIDsActive(c.Request.Context())
		if err == nil {
			for _, uid := range ids {
				cid, _ := newClientIDForServer()
				_ = d.Store.EnsureNodeUser(c.Request.Context(), uid, nodeID, cid, body.Protocol)
			}
		}

		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "node.create", ResourceType: "node", ResourceID: strconv.FormatInt(nodeID, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{
			"node_id":      nodeID,
			"node_secret":  secret, // returned ONCE; only the hash is persisted
		})
	}
}

func adminListNodes(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListAllNodes(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}
