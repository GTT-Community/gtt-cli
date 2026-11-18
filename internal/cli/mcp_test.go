package cli

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GTT-Community/gtt-cli/internal/app"
)

// Criterion 41: the server answers the protocol handshake and lists the seven tools, each read only.
func TestMCPHandshakeListsSevenReadOnlyTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ct, st := mcp.NewInMemoryTransports()
	srv := NewMCPServer(&app.App{})
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if info := cs.InitializeResult().ServerInfo; info == nil || info.Name != "gtt" {
		t.Errorf("server info: %+v", info)
	}
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s must declare readOnlyHint true", tool.Name)
		}
	}
	want := append([]string(nil), app.MCPToolOrder...)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != 7 {
		t.Fatalf("got %d tools: %v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tools %v, want %v", got, want)
			break
		}
	}
}
