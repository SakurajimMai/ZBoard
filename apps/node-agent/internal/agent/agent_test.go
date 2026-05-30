package agent

import "testing"

func TestValidTaskID(t *testing.T) {
	valid := []string{
		"task-20240131120000-ab12",
		"task-disable-expired-ab12cd",
		"task-rollback-20240101010101-rand-9f8e",
		"a",
		"A1._-b2",
	}
	for _, id := range valid {
		if !validTaskID(id) {
			t.Errorf("expected %q to be valid", id)
		}
	}

	invalid := []string{
		"",                          // empty
		"task/../../etc/passwd",     // path traversal
		"task-1/result",            // injects a path segment
		"task 1",                   // space
		"task?x=1",                 // query string
		"task#frag",                // fragment
		"задача",                   // non-ASCII
		"task\n1",                  // newline
	}
	for _, id := range invalid {
		if validTaskID(id) {
			t.Errorf("expected %q to be rejected", id)
		}
	}

	// Over the 128-char cap.
	long := make([]byte, 129)
	for i := range long {
		long[i] = 'a'
	}
	if validTaskID(string(long)) {
		t.Errorf("expected over-length id to be rejected")
	}
}
