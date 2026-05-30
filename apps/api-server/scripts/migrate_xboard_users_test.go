package main

import (
	"os"
	"strings"
	"testing"
)

func TestXboardMigrationDoesNotPersistPlainSubscriptionToken(t *testing.T) {
	body, err := os.ReadFile("migrate_xboard_users.go")
	if err != nil {
		t.Fatalf("read migration script: %v", err)
	}
	src := string(body)
	if strings.Contains(src, "userID, token, hex.EncodeToString(hash[:])") {
		t.Fatalf("migration script inserts the source subscription token as plaintext")
	}
	if !strings.Contains(src, "storedSubToken") {
		t.Fatalf("migration script should persist only a stored subscription token representation")
	}
}
