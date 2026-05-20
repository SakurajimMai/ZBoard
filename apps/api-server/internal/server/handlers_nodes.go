package server

import (
	"net/http"
	"strconv"
	"strings"

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

type updateNodeBody struct {
	createNodeBody
	Status string `json:"status"`
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
			"node_id":     nodeID,
			"node_secret": secret, // returned ONCE; only the hash is persisted
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

func adminUpdateNode(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		var body updateNodeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		status := strings.TrimSpace(body.Status)
		if status == "" {
			status = "active"
		}
		if status != "active" && status != "inactive" {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "节点状态不合法"))
			return
		}
		in := normalizeNodeUpdate(body)
		in.Status = status
		if err := d.Store.UpdateNode(c.Request.Context(), id, in); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "node.update", ResourceType: "node", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func normalizeNodeUpdate(body updateNodeBody) store.UpdateNodeInput {
	in := store.UpdateNodeInput{
		Name:              body.Name,
		Region:            body.Region,
		Host:              body.Host,
		Port:              body.Port,
		Protocol:          defaultNodeString(body.Protocol, "vless"),
		Transport:         body.Transport,
		Security:          body.Security,
		RuntimeType:       body.RuntimeType,
		WSPath:            defaultNodeString(body.WSPath, "/"),
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
	}
	if in.Transport == "" {
		in.Transport = "tcp"
	}
	if in.Security == "" {
		in.Security = "tls"
	}
	if in.RuntimeType == "" {
		in.RuntimeType = "xray"
	}
	if in.SNI == "" {
		in.SNI = in.Host
	}
	if in.Protocol == "ss" || in.Protocol == "shadowsocks" {
		if in.SSMethod == "" {
			in.SSMethod = "2022-blake3-aes-128-gcm"
		}
	}
	if in.Protocol == "vless" && in.Security == "reality" && in.Flow == "" {
		in.Flow = "xtls-rprx-vision"
	}
	if in.Protocol == "hysteria2" || in.Protocol == "tuic" {
		in.RuntimeType = "sing-box"
		in.Transport = "udp"
		if in.CongestionControl == "" {
			in.CongestionControl = "bbr"
		}
	}
	if in.Protocol == "hysteria2" {
		if in.UpMbps == 0 {
			in.UpMbps = 100
		}
		if in.DownMbps == 0 {
			in.DownMbps = 200
		}
	}
	return in
}

func defaultNodeString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
