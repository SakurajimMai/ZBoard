package authsvc

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/mailer"
	"github.com/zboard/api-server/internal/store"
)

const (
	UserSessionTTL  = 7 * 24 * time.Hour
	AdminSessionTTL = 12 * time.Hour
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
	if s.Mailer != nil {
		_ = s.Mailer.SendCode(email, code, purpose)
	}
	return nil
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
	return id, nil
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
	id, err := s.Store.FindUserSession(ctx, authx.HashToken(token))
	if err != nil {
		if store.IsNoRows(err) {
			return 0, httpx.ErrUnauthorized
		}
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
	id, err := s.Store.FindAdminSession(ctx, authx.HashToken(token))
	if err != nil {
		if store.IsNoRows(err) {
			return nil, httpx.ErrUnauthorized
		}
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
