package stats

import (
	"sort"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestQueryStatsRequestMarshal(t *testing.T) {
	r := &queryStatsRequest{Pattern: "user>>>", Reset_: true}
	got, err := r.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Hand-decode to confirm the wire format.
	var pattern string
	var reset bool
	data := got
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			t.Fatalf("ConsumeTag: %v", protowire.ParseError(n))
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			data = data[n:]
			pattern = v
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			data = data[n:]
			reset = v != 0
		default:
			data = data[protowire.ConsumeFieldValue(num, typ, data):]
		}
	}
	if pattern != "user>>>" || !reset {
		t.Fatalf("decoded request: pattern=%q reset=%v", pattern, reset)
	}
}

// encodeStat builds a single Stat sub-message body the way Xray would.
func encodeStat(name string, value int64) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, name)
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(value))
	return b
}

// encodeResponse wraps stats into the QueryStatsResponse envelope.
func encodeResponse(stats []Stat) []byte {
	var b []byte
	for _, s := range stats {
		body := encodeStat(s.Name, s.Value)
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendBytes(b, body)
	}
	return b
}

func TestQueryStatsResponseUnmarshal(t *testing.T) {
	in := []Stat{
		{Name: "user>>>u1@zboard>>>traffic>>>uplink", Value: 100},
		{Name: "user>>>u1@zboard>>>traffic>>>downlink", Value: 200},
		{Name: "inbound>>>api>>>traffic>>>uplink", Value: 9999},
	}
	wire := encodeResponse(in)
	var resp queryStatsResponse
	if err := resp.Unmarshal(wire); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(resp.Stats) != len(in) {
		t.Fatalf("got %d stats, want %d", len(resp.Stats), len(in))
	}
	for i, s := range resp.Stats {
		if s.Name != in[i].Name || s.Value != in[i].Value {
			t.Errorf("stat[%d] = {%q,%d}, want {%q,%d}", i, s.Name, s.Value, in[i].Name, in[i].Value)
		}
	}
}

func TestAggregateXrayAndSingBoxNames(t *testing.T) {
	stats := []Stat{
		{Name: "user>>>u1@zboard>>>traffic>>>uplink", Value: 1024},
		{Name: "user>>>u1@zboard>>>traffic>>>downlink", Value: 4096},
		// sing-box style — no @zboard suffix
		{Name: "user>>>u2>>>traffic>>>uplink", Value: 500},
		{Name: "user>>>u2>>>traffic>>>downlink", Value: 0},
		// inbound stat must be ignored
		{Name: "inbound>>>api>>>traffic>>>uplink", Value: 9999},
		// malformed
		{Name: "user>>>not-a-user>>>traffic>>>uplink", Value: 7},
		// zero-only user must be dropped from output
		{Name: "user>>>u3@zboard>>>traffic>>>uplink", Value: 0},
		{Name: "user>>>u3@zboard>>>traffic>>>downlink", Value: 0},
	}
	got := Aggregate(stats)
	sort.Slice(got, func(i, j int) bool { return got[i].UserID < got[j].UserID })

	want := []UserDelta{
		{UserID: 1, Upload: 1024, Download: 4096},
		{UserID: 2, Upload: 500, Download: 0},
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("delta[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestAggregateSumsMultipleSamples(t *testing.T) {
	stats := []Stat{
		{Name: "user>>>u9>>>traffic>>>uplink", Value: 10},
		{Name: "user>>>u9>>>traffic>>>uplink", Value: 30},
		{Name: "user>>>u9>>>traffic>>>downlink", Value: 7},
	}
	got := Aggregate(stats)
	if len(got) != 1 {
		t.Fatalf("want 1 entry, got %d", len(got))
	}
	if got[0] != (UserDelta{UserID: 9, Upload: 40, Download: 7}) {
		t.Fatalf("aggregate = %#v", got[0])
	}
}

func TestProtoCodecDispatch(t *testing.T) {
	codec := protoCodec{}
	if codec.Name() != "proto" {
		t.Fatalf("codec name = %q, want proto", codec.Name())
	}
	req := &queryStatsRequest{Pattern: "x", Reset_: false}
	body, err := codec.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(body) == 0 {
		t.Fatalf("empty marshal output")
	}
	if _, err := codec.Marshal("not a proto message"); err == nil {
		t.Fatal("expected error marshaling unsupported type")
	}
}
