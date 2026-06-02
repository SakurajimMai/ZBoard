package store_test

import (
	"context"
	"testing"

	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
)

// ReorderNodes must persist new sort values and the list endpoints must return
// nodes ordered by sort (then id), so the admin drag-to-reorder UI controls the
// subscription display order.
func TestReorderNodesControlsListOrder(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	mk := func(name string) int64 {
		id, _, err := s.CreateNode(ctx, store.CreateNodeInput{
			Name: name, Host: name + ".example.com", Port: 443,
			Protocol: "vless", Transport: "tcp", Security: "tls",
			RuntimeType: "xray",
		})
		if err != nil {
			t.Fatalf("CreateNode %s: %v", name, err)
		}
		return id
	}
	a := mk("A")
	b := mk("B")
	c := mk("C")

	// New nodes append in creation order (sort = max+1), so the initial order
	// is A, B, C by both sort and id.
	nodes, err := s.ListActiveNodes(ctx)
	if err != nil {
		t.Fatalf("ListActiveNodes: %v", err)
	}
	if len(nodes) != 3 || nodes[0].ID != a || nodes[2].ID != c {
		t.Fatalf("unexpected initial order: %+v", nodeIDs(nodes))
	}

	// Reorder to C, A, B.
	if err := s.ReorderNodes(ctx, []store.NodeSort{
		{ID: c, Sort: 0}, {ID: a, Sort: 1}, {ID: b, Sort: 2},
	}); err != nil {
		t.Fatalf("ReorderNodes: %v", err)
	}

	nodes, err = s.ListActiveNodes(ctx)
	if err != nil {
		t.Fatalf("ListActiveNodes after reorder: %v", err)
	}
	got := nodeIDs(nodes)
	want := []int64{c, a, b}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order after reorder = %v, want %v", got, want)
		}
	}
}

func nodeIDs(nodes []store.Node) []int64 {
	out := make([]int64, len(nodes))
	for i, n := range nodes {
		out[i] = n.ID
	}
	return out
}
