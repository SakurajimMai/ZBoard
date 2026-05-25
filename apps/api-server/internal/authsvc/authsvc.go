package authsvc

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/mailer"
	"github.com/zboard/api-server/internal/store"
)

const (
	UserSessionTTL  = 180 * 24 * time.Hour
	AdminSessionTTL = 180 * 24 * time.Hour
	EmailCodeTTL    = 10 * time.Minute
	EmailResendCool = 120 * time.Second
)

type Service struct {
	Store      *store.Store
	SetupToken string
	Mailer     *mailer.Mailer
}

func New(s *store.Store, setupToken string, m *mailer.Mailer) *Service {
	return &Service{Store: s, SetupToken: setupToken, Mailer: m}
}

// SendEmailCode generates a 6-digit code, persists it, and emails it. Enforces
// 120s resend cooldown per (email, purpose). When SMTP is not configured, the
// code is logged only; useful for dev environments.
func (s *Service) SendEmailCode(ctx context.Context, email, purpose string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return httpx.NewError(http.StatusBadRequest, "bad_request", "邮箱为空")
	}

	// Cooldown check
	last, err := s.Store.FindLatestEmailCode(ctx, email, purpose)
	if err != nil && !store.IsNoRows(err) {
		return err
	}
	if last != nil {
		gap := time.Since(last.LastSentAt)
		if gap < EmailResendCool {
			remain := int((EmailResendCool - gap).Seconds())
			return httpx.NewError(http.StatusTooManyRequests, "rate_limited",
				fmt.Sprintf("请 %d 秒后再试", remain))
		}
	}

	// Purpose-specific guard
	switch purpose {
	case "register":
		if u, err := s.Store.FindUserByEmail(ctx, email); err == nil && u != nil {
			return httpx.NewError(http.StatusConflict, "email_taken", "邮箱已注册")
		}
	case "reset_password":
		if _, err := s.Store.FindUserByEmail(ctx, email); err != nil {
			if store.IsNoRows(err) {
				return httpx.NewError(http.StatusNotFound, "email_not_found", "该邮箱未注册")
			}
			return err
		}
	default:
		return httpx.NewError(http.StatusBadRequest, "bad_request", "无效的 purpose")
	}

	code := genCode6()
	if err := s.Store.CreateEmailCode(ctx, email, code, purpose, EmailCodeTTL); err != nil {
		return err
	}
	if m, err := s.mailerForRequest(ctx); err == nil && m != nil {
		subject, body, html, err := s.emailCodeTemplate(ctx, email, purpose, code)
		if err != nil {
			return err
		}
		if html {
			_ = m.SendHTML(email, subject, body)
		} else {
			_ = m.SendText(email, subject, body)
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (s *Service) mailerForRequest(ctx context.Context) (*mailer.Mailer, error) {
	host, err := s.Store.GetSetting(ctx, "smtp_host", "")
	if err != nil {
		return nil, err
	}
	from, err := s.Store.GetSetting(ctx, "smtp_from_email", "")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(host) == "" || strings.TrimSpace(from) == "" {
		return s.Mailer, nil
	}
	portRaw, err := s.Store.GetSetting(ctx, "smtp_port", "587")
	if err != nil {
		return nil, err
	}
	user, err := s.Store.GetSetting(ctx, "smtp_user", "")
	if err != nil {
		return nil, err
	}
	pass, err := s.Store.GetSetting(ctx, "smtp_pass", "")
	if err != nil {
		return nil, err
	}
	fromName, err := s.Store.GetSetting(ctx, "smtp_from_name", "")
	if err != nil {
		return nil, err
	}
	encryption, err := s.Store.GetSetting(ctx, "smtp_encryption", "starttls")
	if err != nil {
		return nil, err
	}
	authEnabled, err := s.Store.BoolSetting(ctx, "smtp_auth_enabled", true)
	if err != nil {
		return nil, err
	}
	verifySSL, err := s.Store.BoolSetting(ctx, "smtp_ssl_verify_enabled", true)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(portRaw))
	if err != nil || port <= 0 {
		port = 587
	}
	return mailer.New(mailer.Config{
		Host:        strings.TrimSpace(host),
		Port:        port,
		User:        strings.TrimSpace(user),
		Pass:        pass,
		From:        strings.TrimSpace(from),
		FromName:    strings.TrimSpace(fromName),
		Encryption:  strings.TrimSpace(strings.ToLower(encryption)),
		AuthEnabled: authEnabled,
		SkipVerify:  !verifySSL,
	}), nil
}

func (s *Service) SendAdminEmail(ctx context.Context, to, subject, body string) error {
	m, err := s.mailerForRequest(ctx)
	if err != nil {
		return err
	}
	if m == nil || !m.Enabled() {
		return httpx.NewError(http.StatusBadRequest, "mailer_not_configured", "邮件服务未配置")
	}
	return m.SendText(to, subject, body)
}

func (s *Service) emailCodeTemplate(ctx context.Context, email, purpose, code string) (string, string, bool, error) {
	subjectKey := "email_template_register_subject"
	bodyKey := "email_template_register_body"
	defaultSubject := "[{{site_name}}] 注册验证码 {{code}}"
	defaultBody := "您好，您的 {{site_name}} 验证码是：{{code}}\n\n验证码 10 分钟内有效。如果不是本人操作，请忽略此邮件。"
	if purpose == "reset_password" {
		subjectKey = "email_template_reset_subject"
		bodyKey = "email_template_reset_body"
		defaultSubject = "[{{site_name}}] 密码重置验证码 {{code}}"
		defaultBody = "您好，您的 {{site_name}} 密码重置验证码是：{{code}}\n\n验证码 10 分钟内有效。如果不是本人操作，请立即检查账号安全。"
	}
	subject, err := s.Store.GetSetting(ctx, subjectKey, defaultSubject)
	if err != nil {
		return "", "", false, err
	}
	body, err := s.Store.GetSetting(ctx, bodyKey, defaultBody)
	if err != nil {
		return "", "", false, err
	}
	siteName, err := s.Store.GetSetting(ctx, "site_name", "Zboard")
	if err != nil {
		return "", "", false, err
	}
	username := strings.Split(strings.TrimSpace(email), "@")[0]
	replacer := strings.NewReplacer(
		"{{code}}", code,
		"{{email}}", email,
		"{{username}}", username,
		"{{site_name}}", siteName,
	)
	subject = replacer.Replace(subject)
	body = replacer.Replace(body)
	lower := strings.ToLower(body)
	isHTML := strings.Contains(lower, "<html") || strings.Contains(lower, "<body") || strings.Contains(lower, "<p") || strings.Contains(lower, "<div")
	return subject, body, isHTML, nil
}

// RegisterUserWithCode validates the verification code, then creates the user.
func (s *Service) RegisterUserWithCode(ctx context.Context, email, password, code string) (int64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" || code == "" {
		return 0, httpx.NewError(http.StatusBadRequest, "bad_request", "邮箱、密码或验证码为空")
	}
	if len(password) < 6 {
		return 0, httpx.NewError(http.StatusBadRequest, "bad_request", "密码至少 6 位")
	}
	ok, err := s.Store.VerifyEmailCode(ctx, email, code, "register")
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, httpx.NewError(http.StatusBadRequest, "invalid_code", "验证码错误或已过期")
	}
	hash, err := authx.HashPassword(password)
	if err != nil {
		return 0, err
	}
	id, err := s.Store.CreateUser(ctx, email, hash)
	if err != nil {
		if store.IsUniqueViolation(err) {
			return 0, httpx.NewError(http.StatusConflict, "email_taken", "邮箱已注册")
		}
		return 0, err
	}
	if err := s.applyTrialSettings(ctx, id); err != nil {
		return 0, err
	}
	return id, nil
}

// ResetPasswordWithCode verifies the code then updates the user's password hash.
func (s *Service) ResetPasswordWithCode(ctx context.Context, email, newPassword, code string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || newPassword == "" || code == "" {
		return httpx.NewError(http.StatusBadRequest, "bad_request", "邮箱、新密码或验证码为空")
	}
	if len(newPassword) < 6 {
		return httpx.NewError(http.StatusBadRequest, "bad_request", "密码至少 6 位")
	}
	u, err := s.Store.FindUserByEmail(ctx, email)
	if err != nil {
		if store.IsNoRows(err) {
			return httpx.NewError(http.StatusNotFound, "email_not_found", "该邮箱未注册")
		}
		return err
	}
	ok, err := s.Store.VerifyEmailCode(ctx, email, code, "reset_password")
	if err != nil {
		return err
	}
	if !ok {
		return httpx.NewError(http.StatusBadRequest, "invalid_code", "验证码错误或已过期")
	}
	hash, err := authx.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.Store.UpdateUserPasswordHash(ctx, u.ID, hash)
}

// genCode6 returns a 6-digit numeric verification code.
func genCode6() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	n := binary.BigEndian.Uint32(b[:]) % 1000000
	return fmt.Sprintf("%06d", n)
}

// ===== User =====

func (s *Service) RegisterUser(ctx context.Context, email, password string) (int64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return 0, httpx.NewError(http.StatusBadRequest, "bad_request", "邮箱或密码为空")
	}
	if existing, err := s.Store.FindUserByEmail(ctx, email); err == nil && existing != nil {
		return 0, httpx.NewError(http.StatusConflict, "email_taken", "邮箱已注册")
	} else if err != nil && !store.IsNoRows(err) {
		return 0, err
	}
	hash, err := authx.HashPassword(password)
	if err != nil {
		return 0, err
	}
	id, err := s.Store.CreateUser(ctx, email, hash)
	if err != nil {
		if store.IsUniqueViolation(err) {
			return 0, httpx.NewError(http.StatusConflict, "email_taken", "邮箱已注册")
		}
		return 0, err
	}
	if err := s.applyTrialSettings(ctx, id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) applyTrialSettings(ctx context.Context, userID int64) error {
	trafficGB, err := s.intSetting(ctx, "trial_traffic_gb", 0)
	if err != nil {
		return err
	}
	trialDays, err := s.intSetting(ctx, "trial_days", 0)
	if err != nil {
		return err
	}
	deviceLimit, err := s.intSetting(ctx, "user_default_device_limit", 3)
	if err != nil {
		return err
	}
	if trafficGB <= 0 && trialDays <= 0 {
		return nil
	}
	var expiredAt *time.Time
	if trialDays > 0 {
		t := time.Now().UTC().AddDate(0, 0, trialDays)
		expiredAt = &t
	}
	trafficLimit := int64(trafficGB) * 1073741824
	u, err := s.Store.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.Store.AdminUpdateUser(ctx, userID, store.AdminUpdateUserInput{
		Email:        u.Email,
		Balance:      u.Balance,
		PlanID:       u.PlanID,
		ExpiredAt:    expiredAt,
		TrafficLimit: trafficLimit,
		TrafficUsed:  u.TrafficUsed,
		Status:       u.Status,
	}); err != nil {
		return err
	}
	if deviceLimit <= 0 {
		deviceLimit = 3
	}
	nodes, err := s.Store.ListActiveNodes(ctx)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		clientID, err := newClientID()
		if err != nil {
			return err
		}
		if err := s.Store.EnsureNodeUserWithLimits(ctx, userID, n.ID, clientID, n.Protocol, 0, deviceLimit); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) intSetting(ctx context.Context, key string, fallback int) (int, error) {
	raw, err := s.Store.GetSetting(ctx, key, strconv.Itoa(fallback))
	if err != nil {
		return fallback, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback, nil
	}
	return n, nil
}

func newClientID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	), nil
}

func (s *Service) LoginUser(ctx context.Context, email, password string) (string, *store.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	u, err := s.Store.FindUserByEmail(ctx, email)
	if err != nil {
		if store.IsNoRows(err) {
			return "", nil, httpx.NewError(http.StatusUnauthorized, "invalid_credentials", "账号或密码错误")
		}
		return "", nil, err
	}
	if u.Status != "active" {
		return "", nil, httpx.NewError(http.StatusForbidden, "user_disabled", "账号已禁用")
	}
	if err := authx.VerifyPassword(u.PasswordHash, password); err != nil {
		return "", nil, httpx.NewError(http.StatusUnauthorized, "invalid_credentials", "账号或密码错误")
	}
	token, err := authx.NewToken(32)
	if err != nil {
		return "", nil, err
	}
	if err := s.Store.CreateUserSession(ctx, u.ID, authx.HashToken(token), time.Now().UTC().Add(UserSessionTTL)); err != nil {
		return "", nil, err
	}
	return token, u, nil
}

func (s *Service) ResolveUserToken(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, httpx.ErrUnauthorized
	}
	tokenHash := authx.HashToken(token)
	id, err := s.Store.FindUserSession(ctx, tokenHash)
	if err != nil {
		if store.IsNoRows(err) {
			return 0, httpx.ErrUnauthorized
		}
		return 0, err
	}
	if err := s.Store.RefreshUserSession(ctx, tokenHash, time.Now().UTC().Add(UserSessionTTL)); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) LogoutUser(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.Store.DeleteUserSession(ctx, authx.HashToken(token))
}

// ===== Admin =====

func (s *Service) BootstrapAdmin(ctx context.Context, setupToken, email, password string) (int64, error) {
	if s.SetupToken == "" || setupToken != s.SetupToken {
		return 0, httpx.NewError(http.StatusForbidden, "setup_token_invalid", "Setup token 无效")
	}
	count, err := s.Store.CountAdmins(ctx)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, httpx.NewError(http.StatusConflict, "already_initialized", "管理员已初始化")
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return 0, httpx.NewError(http.StatusBadRequest, "bad_request", "邮箱或密码为空")
	}
	hash, err := authx.HashPassword(password)
	if err != nil {
		return 0, err
	}
	return s.Store.CreateAdmin(ctx, email, hash, "owner")
}

func (s *Service) LoginAdmin(ctx context.Context, email, password string) (string, *store.AdminUser, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	a, err := s.Store.FindAdminByEmail(ctx, email)
	if err != nil {
		if store.IsNoRows(err) {
			return "", nil, httpx.NewError(http.StatusUnauthorized, "invalid_credentials", "账号或密码错误")
		}
		return "", nil, err
	}
	if a.Status != "active" {
		return "", nil, httpx.NewError(http.StatusForbidden, "admin_disabled", "管理员已禁用")
	}
	if err := authx.VerifyPassword(a.PasswordHash, password); err != nil {
		return "", nil, httpx.NewError(http.StatusUnauthorized, "invalid_credentials", "账号或密码错误")
	}
	token, err := authx.NewToken(32)
	if err != nil {
		return "", nil, err
	}
	if err := s.Store.CreateAdminSession(ctx, a.ID, authx.HashToken(token), time.Now().UTC().Add(AdminSessionTTL)); err != nil {
		return "", nil, err
	}
	_ = s.Store.TouchAdminLogin(ctx, a.ID)
	return token, a, nil
}

func (s *Service) ResolveAdminToken(ctx context.Context, token string) (*store.AdminUser, error) {
	if token == "" {
		return nil, httpx.ErrUnauthorized
	}
	tokenHash := authx.HashToken(token)
	id, err := s.Store.FindAdminSession(ctx, tokenHash)
	if err != nil {
		if store.IsNoRows(err) {
			return nil, httpx.ErrUnauthorized
		}
		return nil, err
	}
	if err := s.Store.RefreshAdminSession(ctx, tokenHash, time.Now().UTC().Add(AdminSessionTTL)); err != nil {
		return nil, err
	}
	a, err := s.Store.FindAdminByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.Status != "active" {
		return nil, httpx.ErrForbidden
	}
	return a, nil
}

func (s *Service) LogoutAdmin(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.Store.DeleteAdminSession(ctx, authx.HashToken(token))
}

// ExtractBearer reads the Bearer token from an Authorization header value.
func ExtractBearer(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

// ErrInvalidToken signals a malformed/missing Bearer header to handlers.
var ErrInvalidToken = errors.New("invalid token")
