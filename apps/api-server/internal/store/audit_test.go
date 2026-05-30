package store

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateAuditDetailShortPassesThrough(t *testing.T) {
	in := "ok"
	if got := truncateAuditDetail(in); got != in {
		t.Fatalf("short detail mutated: got %q", got)
	}
}

func TestTruncateAuditDetailStaysWithinBudget(t *testing.T) {
	in := strings.Repeat("a", AuditDetailMaxBytes+500)
	got := truncateAuditDetail(in)
	if len(got) > AuditDetailMaxBytes {
		t.Fatalf("truncated detail exceeds cap: len=%d cap=%d", len(got), AuditDetailMaxBytes)
	}
	if !strings.HasSuffix(got, auditTruncMarker) {
		t.Fatalf("missing truncation marker: %q", got[len(got)-20:])
	}
}

// The regression M15 guards against: a byte-level slice can cut through a
// multi-byte rune and leave invalid UTF-8, which MySQL's utf8mb4 columns reject
// in strict mode (dropping the whole audit row). The fix must always emit valid
// UTF-8 regardless of where the cap lands relative to a codepoint boundary.
func TestTruncateAuditDetailNeverProducesInvalidUTF8(t *testing.T) {
	// "世" is 3 bytes in UTF-8. Filling the budget with these guarantees that a
	// naive cut at AuditDetailMaxBytes lands mid-rune for at least one offset.
	for pad := 0; pad < 6; pad++ {
		in := strings.Repeat("x", pad) + strings.Repeat("世", AuditDetailMaxBytes)
		got := truncateAuditDetail(in)
		if !utf8.ValidString(got) {
			t.Fatalf("pad=%d produced invalid UTF-8", pad)
		}
		if len(got) > AuditDetailMaxBytes {
			t.Fatalf("pad=%d exceeds cap: len=%d", pad, len(got))
		}
		if !strings.HasSuffix(got, auditTruncMarker) {
			t.Fatalf("pad=%d missing marker", pad)
		}
	}
}
