package authsvc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/testsupport"
)

func TestUserRegisterLoginResolve(t *testing.T) {
	s := testsupport.NewStore(t)
	svc := authsvc.New(s, "setup", nil)
	ctx := context.Background()

	id, err := svc.RegisterUser(ctx, "X@Example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if id == 0 {
		t.Fatalf("zero id")
	}

	// Email is normalized to lowercase
	tok, u, err := svc.LoginUser(ctx, "x@example.com", "secret123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if tok == "" || u.ID != id {
		t.Fatalf("login result: tok=%q u=%+v", tok, u)
	}

	resolved, err := svc.ResolveUserToken(ctx, tok)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved != id {
		t.Fatalf("resolve mismatch: %d vs %d", resolved, id)
	}
}

func TestAdminBootstrapSingleton(t *testing.T) {
	s := testsupport.NewStore(t)
	svc := authsvc.New(s, "setup-token", nil)
	ctx := context.Background()

	id, err := svc.BootstrapAdmin(ctx, "setup-token", "owner@zboard.local", "pw")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if id == 0 {
		t.Fatalf("zero id")
	}

	// Wrong setup token -> 403
	_, err = svc.BootstrapAdmin(ctx, "wrong", "x@example.com", "pw")
	var ae *httpx.AppError
	if !errors.As(err, &ae) || ae.Code != "setup_token_invalid" {
		t.Fatalf("expected setup_token_invalid, got %v", err)
	}

	// Already initialized -> 409
	_, err = svc.BootstrapAdmin(ctx, "setup-token", "x@example.com", "pw")
	if !errors.As(err, &ae) || ae.Code != "already_initialized" {
		t.Fatalf("expected already_initialized, got %v", err)
	}
}

func TestExtractBearer(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"", ""},
		{"Bearer abc", "abc"},
		{"Bearer  spaced ", "spaced"},
		{"Token nope", ""},
	}
	for _, c := range cases {
		if got := authsvc.ExtractBearer(c.header); got != c.want {
			t.Errorf("ExtractBearer(%q)=%q want %q", c.header, got, c.want)
		}
	}
}
