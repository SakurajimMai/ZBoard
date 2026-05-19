package stats_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/zboard/node-agent/internal/stats"
)

// fakeStatsServer answers the same wire protocol Xray/sing-box use, so the
// agent's gRPC client gets exercised end-to-end without pulling in xray-core.
type fakeStatsServer struct {
	stats   []stats.Stat
	queries int
	resets  int
}

// QueryStatsMethod must match what the client dials.
const queryMethod = "/xray.app.stats.command.StatsService/QueryStats"

// streamHandler implements the grpc StreamHandler signature; we register it
// generically so we don't need to import or generate pb stubs server-side.
func (s *fakeStatsServer) streamHandler(srv any, stream grpc.ServerStream) error {
	// Decode the incoming QueryStatsRequest by hand.
	var raw rawBytes
	if err := stream.RecvMsg(&raw); err != nil {
		return err
	}
	var pattern string
	var reset bool
	d := []byte(raw)
	for len(d) > 0 {
		num, typ, n := protowire.ConsumeTag(d)
		if n < 0 {
			return protowire.ParseError(n)
		}
		d = d[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(d)
			if n < 0 {
				return protowire.ParseError(n)
			}
			d = d[n:]
			pattern = v
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(d)
			if n < 0 {
				return protowire.ParseError(n)
			}
			d = d[n:]
			reset = v != 0
		default:
			d = d[protowire.ConsumeFieldValue(num, typ, d):]
		}
	}
	s.queries++
	if reset {
		s.resets++
	}
	_ = pattern // we honor the call regardless of pattern in this fake

	// Build the response, then if reset==true clear our local stats.
	body := encodeResponseBytes(s.stats)
	if reset {
		s.stats = nil
	}
	out := rawBytes(body)
	return stream.SendMsg(&out)
}

// rawBytes is a passthrough proto type whose Marshal/Unmarshal copy bytes
// verbatim. Implements the same interface our client codec expects.
type rawBytes []byte

func (r *rawBytes) Marshal() ([]byte, error) { return []byte(*r), nil }
func (r *rawBytes) Unmarshal(b []byte) error { *r = append((*r)[:0], b...); return nil }

// passthroughCodec is a grpc encoding.Codec that defers to types implementing
// Marshal()/Unmarshal(). Required so the test server can route bytes to/from
// our rawBytes type without going through proto.Marshal.
type passthroughCodec struct{}

func (passthroughCodec) Name() string { return "proto" }

func (passthroughCodec) Marshal(v any) ([]byte, error) {
	if m, ok := v.(interface{ Marshal() ([]byte, error) }); ok {
		return m.Marshal()
	}
	return nil, errUnsupported
}

func (passthroughCodec) Unmarshal(data []byte, v any) error {
	if m, ok := v.(interface{ Unmarshal([]byte) error }); ok {
		return m.Unmarshal(data)
	}
	return errUnsupported
}

var errUnsupported = errorString("passthroughCodec: unsupported type")

type errorString string

func (e errorString) Error() string { return string(e) }

func encodeStatBytes(name string, value int64) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, name)
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(value))
	return b
}

func encodeResponseBytes(in []stats.Stat) []byte {
	var b []byte
	for _, s := range in {
		body := encodeStatBytes(s.Name, s.Value)
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendBytes(b, body)
	}
	return b
}

func TestQueryAndResetIntegration(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	fake := &fakeStatsServer{
		stats: []stats.Stat{
			{Name: "user>>>u1@zboard>>>traffic>>>uplink", Value: 1024},
			{Name: "user>>>u1@zboard>>>traffic>>>downlink", Value: 4096},
			{Name: "user>>>u2>>>traffic>>>uplink", Value: 500},
			{Name: "inbound>>>api>>>traffic>>>uplink", Value: 9999},
		},
	}
	svc := &grpc.ServiceDesc{
		ServiceName: "xray.app.stats.command.StatsService",
		HandlerType: (*any)(nil),
		Methods:     nil,
		Streams: []grpc.StreamDesc{{
			StreamName:    "QueryStats",
			Handler:       fake.streamHandler,
			ClientStreams: false,
			ServerStreams: false,
		}},
	}
	srv := grpc.NewServer(grpc.ForceServerCodec(passthroughCodec{}))
	srv.RegisterService(svc, fake)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	client, err := stats.Dial(lis.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Force ForceCodec to use our codec on this dial. The Client already does
	// this internally via grpc.ForceCodec(protoCodec{}) per call.
	_ = grpc.WithTransportCredentials(insecure.NewCredentials())

	deltas, err := client.QueryAndReset(ctx)
	if err != nil {
		t.Fatalf("QueryAndReset: %v", err)
	}
	wantByID := map[int64]stats.UserDelta{
		1: {UserID: 1, Upload: 1024, Download: 4096},
		2: {UserID: 2, Upload: 500, Download: 0},
	}
	if len(deltas) != len(wantByID) {
		t.Fatalf("got %d deltas, want %d: %#v", len(deltas), len(wantByID), deltas)
	}
	for _, d := range deltas {
		w, ok := wantByID[d.UserID]
		if !ok {
			t.Errorf("unexpected user_id %d", d.UserID)
			continue
		}
		if d != w {
			t.Errorf("delta for u%d = %#v, want %#v", d.UserID, d, w)
		}
	}

	if fake.queries != 1 || fake.resets != 1 {
		t.Errorf("server saw queries=%d resets=%d, want 1/1", fake.queries, fake.resets)
	}

	// Second call must return zero deltas because reset cleared the counters.
	deltas2, err := client.QueryAndReset(ctx)
	if err != nil {
		t.Fatalf("QueryAndReset #2: %v", err)
	}
	if len(deltas2) != 0 {
		t.Errorf("expected empty after reset, got %#v", deltas2)
	}
}
