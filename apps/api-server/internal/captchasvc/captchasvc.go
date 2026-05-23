package captchasvc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

const (
	ProviderNone       = "none"
	ProviderTurnstile  = "turnstile"
	ProviderRecaptcha  = "recaptcha"
	ProviderHCaptcha   = "hcaptcha"
	verifyHTTPTimeout  = 8 * time.Second
)

// Scene controls per-page enable flags. The zero value disables verification.
type Scene string

const (
	SceneRegister Scene = "register"
	SceneLogin    Scene = "login"
	SceneForgot   Scene = "forgot"
	SceneTicket   Scene = "ticket"
)

type Service struct {
	Store      *store.Store
	HTTPClient *http.Client
}

func New(s *store.Store) *Service {
	return &Service{
		Store:      s,
		HTTPClient: &http.Client{Timeout: verifyHTTPTimeout},
	}
}

// Verify runs server-side siteverify against the configured provider when the
// scene is enabled. When the captcha is disabled (provider == none or scene
// flag off) the call is a no-op and returns nil.
func (s *Service) Verify(ctx context.Context, scene Scene, token, remoteIP string) error {
	provider, err := s.Store.GetSetting(ctx, "captcha_provider", ProviderNone)
	if err != nil {
		return err
	}
	provider = strings.TrimSpace(provider)
	if provider == "" || provider == ProviderNone {
		return nil
	}
	enabled, err := s.Store.BoolSetting(ctx, sceneSettingKey(scene), false)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	secret, err := s.Store.GetSetting(ctx, "captcha_secret_key", "")
	if err != nil {
		return err
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return httpx.NewError(http.StatusServiceUnavailable, "captcha_misconfigured", "人机验证未配置完整")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return httpx.NewError(http.StatusBadRequest, "captcha_required", "请先完成人机验证")
	}
	switch provider {
	case ProviderTurnstile:
		return s.verifyEndpoint(ctx, "https://challenges.cloudflare.com/turnstile/v0/siteverify", secret, token, remoteIP)
	case ProviderRecaptcha:
		return s.verifyEndpoint(ctx, "https://www.google.com/recaptcha/api/siteverify", secret, token, remoteIP)
	case ProviderHCaptcha:
		return s.verifyEndpoint(ctx, "https://api.hcaptcha.com/siteverify", secret, token, remoteIP)
	default:
		return httpx.NewError(http.StatusServiceUnavailable, "captcha_misconfigured", "未知的人机验证服务")
	}
}

func sceneSettingKey(scene Scene) string {
	switch scene {
	case SceneRegister:
		return "captcha_enabled_register"
	case SceneLogin:
		return "captcha_enabled_login"
	case SceneForgot:
		return "captcha_enabled_forgot"
	case SceneTicket:
		return "captcha_enabled_ticket"
	}
	return ""
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func (s *Service) verifyEndpoint(ctx context.Context, endpoint, secret, token, remoteIP string) error {
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return httpx.NewError(http.StatusBadGateway, "captcha_upstream", "人机验证服务不可达")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return httpx.NewError(http.StatusBadGateway, "captcha_upstream", "人机验证响应异常")
	}
	var body siteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return httpx.NewError(http.StatusBadGateway, "captcha_upstream", "人机验证响应解析失败")
	}
	if !body.Success {
		return httpx.NewError(http.StatusBadRequest, "captcha_failed", "人机验证未通过")
	}
	return nil
}

var ErrCaptchaRequired = errors.New("captcha required")
