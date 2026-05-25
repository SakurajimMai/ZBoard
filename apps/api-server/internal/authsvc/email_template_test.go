package authsvc

import (
	"context"
	"strings"
	"testing"

	"github.com/zboard/api-server/internal/testsupport"
)

func TestEmailCodeTemplateUsesAdminSettings(t *testing.T) {
	st := testsupport.NewStore(t)
	ctx := context.Background()
	if err := st.SetSettings(ctx, map[string]string{
		"site_name":                       "九云",
		"email_template_register_subject": "[{{site_name}}] 验证码 {{code}}",
		"email_template_register_body":    "用户 {{username}}，邮箱 {{email}}，验证码 {{code}}，站点 {{site_name}}",
	}); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	svc := New(st, "setup", nil)
	subject, body, isHTML, err := svc.emailCodeTemplate(ctx, "user@example.com", "register", "123456")
	if err != nil {
		t.Fatalf("emailCodeTemplate: %v", err)
	}
	if subject != "[九云] 验证码 123456" {
		t.Fatalf("subject=%q", subject)
	}
	for _, want := range []string{"用户 user", "邮箱 user@example.com", "验证码 123456", "站点 九云"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
	if isHTML {
		t.Fatalf("plain template should not be marked html")
	}
}

func TestEmailCodeTemplateDetectsHTML(t *testing.T) {
	st := testsupport.NewStore(t)
	ctx := context.Background()
	if err := st.SetSettings(ctx, map[string]string{
		"email_template_reset_subject": "重置 {{code}}",
		"email_template_reset_body":    "<p>{{code}}</p>",
	}); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	svc := New(st, "setup", nil)
	_, _, isHTML, err := svc.emailCodeTemplate(ctx, "user@example.com", "reset_password", "654321")
	if err != nil {
		t.Fatalf("emailCodeTemplate: %v", err)
	}
	if !isHTML {
		t.Fatalf("html template should be marked html")
	}
}
