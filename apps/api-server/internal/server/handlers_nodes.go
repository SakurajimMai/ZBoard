package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/runtime"
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
	TLSInsecure       *int   `json:"tls_insecure"`
}

type updateNodeBody struct {
	createNodeBody
	Status string `json:"status"`
	Sort   int    `json:"sort"`
}

func adminCreateNode(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createNodeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		protocol, transport, security, runtimeType := normalizeNodeCreateRuntimeFields(body)
		if err := validateNodeRuntimeFields(
			body.Port,
			protocol,
			transport,
			security,
			runtimeType,
			body.RealityServerName,
			body.RealityPublicKey,
			body.RealityPrivateKey,
			body.PortRange,
		); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		nodeID, secret, err := d.Store.CreateNode(c.Request.Context(), store.CreateNodeInput{
			Name:              body.Name,
			Region:            body.Region,
			Host:              body.Host,
			Port:              body.Port,
			Protocol:          protocol,
			Transport:         transport,
			Security:          security,
			RuntimeType:       runtimeType,
			WSPath:            body.WSPath,
			WSHost:            body.WSHost,
			GRPCServiceName:   runtime.NormalizeGRPCServiceName(transport, body.GRPCServiceName),
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
			TLSInsecure:       normalizeNodeTLSInsecure(protocol, body.TLSInsecure),
			TLSInsecureSet:    body.TLSInsecure != nil,
		})
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		// Backfill node_users for currently active users so new nodes light up
		// in their subscription immediately.
		if err := backfillNodeUsersForSubscription(c.Request.Context(), d.Store, nodeID, protocol); err != nil {
			httpx.Fail(c, err)
			return
		}

		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "node.create", ResourceType: "node", ResourceID: strconv.FormatInt(nodeID, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		resp := gin.H{
			"node_id":     nodeID,
			"node_secret": secret, // returned ONCE; only the hash is persisted
		}
		if d.Nodes != nil {
			if taskID, version, err := d.Nodes.GenerateSyncTask(c.Request.Context(), nodeID); err == nil {
				resp["sync_task_id"] = taskID
				resp["sync_version"] = version
			} else {
				resp["sync_error"] = err.Error()
			}
		}
		httpx.Created(c, resp)
	}
}

func adminListNodes(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := paginationFromQuery(c)
		threshold, err := d.Store.IntSetting(c.Request.Context(), "node_offline_threshold", 120)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		rows, total, err := d.Store.ListAllNodeViewsPage(c.Request.Context(), params, threshold)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		// Mask server-side secrets before they leave the process. The admin edit
		// form reads these back and re-submits them, so we can't drop the fields
		// entirely — adminUpdateNode restores the stored value when the masked
		// placeholder comes back unchanged.
		for i := range rows {
			rows[i].RealityPrivateKey = maskNodeSecret(rows[i].RealityPrivateKey)
			rows[i].ObfsPassword = maskNodeSecret(rows[i].ObfsPassword)
		}
		httpx.OK(c, gin.H{"items": rows, "page": params.Page, "page_size": params.PageSize, "total": total})
	}
}

// maskNodeSecret hides a node's stored secret (Reality private key, hysteria2
// obfs password) in admin list responses. Mirrors the payment-provider config
// masking: keep the first/last 2 chars so an operator can recognize the value
// without the response carrying the full secret (which a compromised admin
// session or an XSS in the panel would otherwise exfiltrate wholesale).
func maskNodeSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// isMaskedNodeSecret reports whether a submitted value is a masked placeholder
// echoed back by the edit form rather than a real secret the admin typed.
func isMaskedNodeSecret(s string) bool {
	return strings.Contains(s, "****")
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
		// The edit form echoes back the masked secrets it received from the list
		// endpoint. When the submitted value is still the mask, restore the
		// stored secret so a save doesn't overwrite the real key with "xx****yy".
		if isMaskedNodeSecret(in.RealityPrivateKey) || isMaskedNodeSecret(in.ObfsPassword) {
			existing, ferr := d.Store.FindNodeByID(c.Request.Context(), id)
			if ferr != nil {
				if store.IsNoRows(ferr) {
					httpx.Fail(c, httpx.NewError(http.StatusNotFound, "node_not_found", "节点不存在"))
					return
				}
				httpx.Fail(c, ferr)
				return
			}
			if isMaskedNodeSecret(in.RealityPrivateKey) {
				in.RealityPrivateKey = existing.RealityPrivateKey
			}
			if isMaskedNodeSecret(in.ObfsPassword) {
				in.ObfsPassword = existing.ObfsPassword
			}
		}
		if err := validateNodeRuntimeFields(
			in.Port,
			in.Protocol,
			in.Transport,
			in.Security,
			in.RuntimeType,
			in.RealityServerName,
			in.RealityPublicKey,
			in.RealityPrivateKey,
			in.PortRange,
		); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if err := d.Store.UpdateNode(c.Request.Context(), id, in); err != nil {
			httpx.Fail(c, err)
			return
		}
		if status == "active" {
			if err := backfillNodeUsersForSubscription(c.Request.Context(), d.Store, id, in.Protocol); err != nil {
				httpx.Fail(c, err)
				return
			}
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "node.update", ResourceType: "node", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		resp := gin.H{"ok": true}
		if status == "active" && d.Nodes != nil {
			if taskID, version, err := d.Nodes.GenerateSyncTask(c.Request.Context(), id); err == nil {
				resp["sync_task_id"] = taskID
				resp["sync_version"] = version
			} else {
				resp["sync_error"] = err.Error()
			}
		}
		httpx.OK(c, resp)
	}
}

func validateNodeRuntimeFields(port int, protocol, transport, security, runtimeType, realityServerName, realityPublicKey, realityPrivateKey, portRange string) error {
	return runtime.ValidateNode(&store.Node{
		Port:              port,
		Protocol:          protocol,
		Transport:         transport,
		Security:          security,
		RuntimeType:       runtimeType,
		RealityServerName: realityServerName,
		RealityPublicKey:  realityPublicKey,
		RealityPrivateKey: realityPrivateKey,
		PortRange:         portRange,
	})
}

func backfillNodeUsersForSubscription(ctx context.Context, st *store.Store, nodeID int64, protocol string) error {
	ids, err := st.ListUserIDsProvisionable(ctx)
	if err != nil {
		return err
	}
	for _, uid := range ids {
		cid, err := newClientIDForServer()
		if err != nil {
			return err
		}
		deviceLimit := 0
		if u, err := st.FindUserByID(ctx, uid); err == nil && u.PlanID != nil {
			if plan, err := st.FindPlanByID(ctx, *u.PlanID); err == nil {
				deviceLimit = plan.DeviceLimit
			}
		}
		if err := st.EnsureNodeUserWithLimits(ctx, uid, nodeID, cid, protocol, 0, deviceLimit); err != nil {
			return err
		}
	}
	return nil
}

func normalizeNodeCreateRuntimeFields(body createNodeBody) (protocol, transport, security, runtimeType string) {
	protocol = defaultNodeString(body.Protocol, "vless")
	transport = defaultNodeString(body.Transport, "tcp")
	security = defaultNodeString(body.Security, "tls")
	runtimeType = defaultNodeString(body.RuntimeType, "xray")
	if protocol == "hysteria2" || protocol == "tuic" {
		transport = "udp"
		security = "tls"
		runtimeType = "sing-box"
	}
	return protocol, transport, security, runtimeType
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
		GRPCServiceName:   runtime.NormalizeGRPCServiceName(body.Transport, body.GRPCServiceName),
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
		TLSInsecure:       normalizeNodeTLSInsecure(defaultNodeString(body.Protocol, "vless"), body.TLSInsecure),
		Sort:              body.Sort,
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
	if in.Protocol == "hysteria2" || in.Protocol == "tuic" {
		in.RuntimeType = "sing-box"
		in.Transport = "udp"
		if in.CongestionControl == "" {
			in.CongestionControl = "bbr"
		}
		if body.TLSInsecure == nil {
			in.TLSInsecure = 1
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

func normalizeNodeTLSInsecure(protocol string, raw *int) int {
	if raw != nil {
		if *raw != 0 {
			return 1
		}
		return 0
	}
	if protocol == "hysteria2" || protocol == "tuic" {
		return 1
	}
	return 0
}

type reorderNodesBody struct {
	Items []struct {
		ID   int64 `json:"id" binding:"required"`
		Sort int   `json:"sort"`
	} `json:"items" binding:"required"`
}

// adminReorderNodes persists a new display order for nodes. Ordering only
// affects subscription rendering (subrender sorts by Sort, then NodeID), so
// this never touches runtime config or triggers a config sync — clients pick
// up the new order on their next subscription pull.
func adminReorderNodes(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body reorderNodesBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if len(body.Items) == 0 {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "items 不能为空"))
			return
		}
		items := make([]store.NodeSort, 0, len(body.Items))
		for _, it := range body.Items {
			items = append(items, store.NodeSort{ID: it.ID, Sort: it.Sort})
		}
		if err := d.Store.ReorderNodes(c.Request.Context(), items); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "node.reorder", ResourceType: "node", ResourceID: "",
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}
