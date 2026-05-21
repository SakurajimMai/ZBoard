package authsvc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
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

func TestRegisterAppliesTrialSettings(t *testing.T) {
	s := testsupport.NewStore(t)
	svc := authsvc.New(s, "setup", nil)
	ctx := context.Background()

	nodeID, _, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name: "US-01", Host: "us.example.com", Port: 443, Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := s.SetSettings(ctx, map[string]string{
		"trial_traffic_gb":          "5",
		"trial_days":                "7",
		"user_default_device_limit": "2",
	}); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	id, err := svc.RegisterUser(ctx, "trial@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	u, err := s.FindUserByID(ctx, id)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.TrafficLimit != int64(5*1073741824) || u.ExpiredAt == nil {
		t.Fatalf("trial not applied: %+v", u)
	}
	nu, err := s.FindNodeUser(ctx, id, nodeID)
	if err != nil {
		t.Fatalf("find node user: %v", err)
	}
	if nu.DeviceLimit != 2 {
		t.Fatalf("device limit = %d, want 2", nu.DeviceLimit)
	}
}

func TestRegisterWithoutTrialDoesNotProvisionNodeAccess(t *testing.T) {
	s := testsupport.NewStore(t)
	svc := authsvc.New(s, "setup", nil)
	ctx := context.Background()

	nodeID, _, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name: "US-01", Host: "us.example.com", Port: 443, Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	id, err := svc.RegisterUser(ctx, "no-trial@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := s.FindNodeUser(ctx, id, nodeID); !store.IsNoRows(err) {
		t.Fatalf("default registration should not provision node user, err=%v", err)
	}
	u, err := s.FindUserByID(ctx, id)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.TrafficLimit != 0 || u.ExpiredAt != nil {
		t.Fatalf("default registration should not apply trial quota: %+v", u)
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
