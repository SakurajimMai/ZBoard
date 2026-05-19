// Package stats reads per-user uplink/downlink counters from the local
// Xray/sing-box runtime via the StatsService gRPC API. Both engines expose the
// same wire format on 127.0.0.1:10085 (Xray's xray.app.stats.command,
// sing-box's experimental.v2ray_api), so one client handles both.
//
// We hand-roll the protobuf encoding for QueryStatsRequest/Response with
// google.golang.org/protobuf/encoding/protowire to avoid pulling in xray-core
// or running protoc as part of the build.
package stats

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protowire"
)

// QueryStatsMethod is the fully-qualified gRPC method path. Both Xray and
// sing-box's v2ray_api advertise it under the same name.
const QueryStatsMethod = "/xray.app.stats.command.StatsService/QueryStats"

// UserDelta is the per-user traffic delta returned by QueryAndReset.
type UserDelta struct {
	UserID   int64
	Upload   int64
	Download int64
}

// Client is a thin grpc wrapper around the StatsService.
type Client struct {
	addr string
	conn *grpc.ClientConn
}

// Dial creates a Client. The grpc connection is established lazily on the
// first Invoke, and grpc-go transparently reconnects when the runtime restarts
// (which happens every config swap), so callers don't need retry loops.
func Dial(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("stats dial %s: %w", addr, err)
	}
	return &Client{addr: addr, conn: conn}, nil
}

// Close releases the underlying grpc connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// QueryAndReset asks the runtime for every `user>>>...>>>traffic>>>...` stat
// and atomically resets the counter to zero, so each call returns a delta.
//
// Stat names look like `user>>>u7@zboard>>>traffic>>>uplink` (Xray) or
// `user>>>u7>>>traffic>>>downlink` (sing-box). The numeric user_id is
// extracted from the leading `u<id>` token.
func (c *Client) QueryAndReset(ctx context.Context) ([]UserDelta, error) {
	req := &queryStatsRequest{Pattern: "user>>>", Reset_: true}
	resp := &queryStatsResponse{}
	if err := c.conn.Invoke(ctx, QueryStatsMethod, req, resp, grpc.ForceCodec(protoCodec{})); err != nil {
		return nil, fmt.Errorf("stats query: %w", err)
	}
	return Aggregate(resp.Stats), nil
}

// Stat is the public form of a single stat row, exported for tests.
type Stat struct {
	Name  string
	Value int64
}

var (
	// userStatRE matches "user>>>{key}>>>traffic>>>{uplink|downlink}".
	userStatRE = regexp.MustCompile(`^user>>>([^>]+)>>>traffic>>>(uplink|downlink)$`)
	// userIDRE extracts the integer user_id from a leading `u<id>` token.
	userIDRE = regexp.MustCompile(`^u(\d+)`)
)

// Aggregate groups stats by user_id and direction. Stats whose names don't
// match the user pattern are silently dropped; rows with zero totals are also
// dropped so the agent doesn't ship empty reports.
func Aggregate(stats []Stat) []UserDelta {
	by := map[int64]*UserDelta{}
	for _, s := range stats {
		m := userStatRE.FindStringSubmatch(s.Name)
		if len(m) != 3 {
			continue
		}
		idMatch := userIDRE.FindStringSubmatch(m[1])
		if len(idMatch) != 2 {
			continue
		}
		id, err := strconv.ParseInt(idMatch[1], 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		d, ok := by[id]
		if !ok {
			d = &UserDelta{UserID: id}
			by[id] = d
		}
		if m[2] == "uplink" {
			d.Upload += s.Value
		} else {
			d.Download += s.Value
		}
	}
	out := make([]UserDelta, 0, len(by))
	for _, d := range by {
		if d.Upload == 0 && d.Download == 0 {
			continue
		}
		out = append(out, *d)
	}
	return out
}

// ===== Protobuf wire encoding =====

type queryStatsRequest struct {
	Pattern string
	Reset_  bool
}

// Marshal serializes QueryStatsRequest{ string pattern = 1; bool reset = 2; }.
func (r *queryStatsRequest) Marshal() ([]byte, error) {
	var b []byte
	if r.Pattern != "" {
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, r.Pattern)
	}
	if r.Reset_ {
		b = protowire.AppendTag(b, 2, protowire.VarintType)
		b = protowire.AppendVarint(b, 1)
	}
	return b, nil
}

type queryStatsResponse struct {
	Stats []Stat
}

// Unmarshal decodes QueryStatsResponse{ repeated Stat stat = 1; } where
// Stat is { string name = 1; int64 value = 2; }.
func (r *queryStatsResponse) Unmarshal(data []byte) error {
	for len(data) > 0 {
		num, typ, hdr := protowire.ConsumeTag(data)
		if hdr < 0 {
			return protowire.ParseError(hdr)
		}
		data = data[hdr:]
		if num == 1 && typ == protowire.BytesType {
			v, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
			var s Stat
			if err := unmarshalStat(v, &s); err != nil {
				return err
			}
			r.Stats = append(r.Stats, s)
			continue
		}
		skip := protowire.ConsumeFieldValue(num, typ, data)
		if skip < 0 {
			return protowire.ParseError(skip)
		}
		data = data[skip:]
	}
	return nil
}

func unmarshalStat(data []byte, s *Stat) error {
	for len(data) > 0 {
		num, typ, hdr := protowire.ConsumeTag(data)
		if hdr < 0 {
			return protowire.ParseError(hdr)
		}
		data = data[hdr:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
			s.Name = v
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
			s.Value = int64(v)
		default:
			skip := protowire.ConsumeFieldValue(num, typ, data)
			if skip < 0 {
				return protowire.ParseError(skip)
			}
			data = data[skip:]
		}
	}
	return nil
}

// protoCodec is the minimal grpc codec we install per-call via grpc.ForceCodec.
// Name() returns "proto" so the wire `content-type` header matches what
// Xray/sing-box expect; we don't pollute grpc-go's global codec registry.
type protoCodec struct{}

func (protoCodec) Name() string { return "proto" }

func (protoCodec) Marshal(v any) ([]byte, error) {
	if m, ok := v.(interface{ Marshal() ([]byte, error) }); ok {
		return m.Marshal()
	}
	return nil, fmt.Errorf("stats codec: unsupported marshal type %T", v)
}

func (protoCodec) Unmarshal(data []byte, v any) error {
	if m, ok := v.(interface{ Unmarshal([]byte) error }); ok {
		return m.Unmarshal(data)
	}
	return fmt.Errorf("stats codec: unsupported unmarshal type %T", v)
}
