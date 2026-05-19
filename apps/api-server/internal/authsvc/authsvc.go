package authsvc

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

const (
	UserSessionTTL  = 7 * 24 * time.Hour
	AdminSessionTTL = 12 * time.Hour
)

type Service struct {
	Store      *store.Store
	SetupToken string
}

func New(s *store.Store, setupToken string) *Service {
	return &Service{Store: s, SetupToken: setupToken}
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
