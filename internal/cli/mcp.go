package cli

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/core"
)

// mcpCmd is `gtt mcp`: the portfolio read by Claude over the Model Context Protocol, on stdin and stdout. It
// opens no port and no network, and it only reads: the seven tools answer from declared read-only operations
// (see MCPToolOrder and the table in internal/app/doors.go).
func mcpCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Let Claude ask about your projects (MCP server over stdio, read only)",
		Long: "gtt mcp is an MCP server over stdio for Claude Desktop, the claude command-line client or any MCP client. It has seven tools\n" +
			"(gtt_projects, gtt_findings, gtt_ready, gtt_needs_you, gtt_epic, gtt_activity, gtt_code), all read only: they answer\n" +
			"with the same data as Sonar and change nothing. It opens no port and uses no network.\n" +
			"It reads the project it is started in plus the repos of `gtt sonar list`.\n\n" +
			"claude CLI:  claude mcp add gtt -- gtt mcp",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return NewMCPServer(e.app).Run(cmd.Context(), &mcp.StdioTransport{})
		},
	}
}

// Inputs of the tools. The paging fields are the same everywhere: at most 50 items per page.
type paging struct {
	Offset int `json:"offset,omitempty" jsonschema:"items to skip (paging)"`
	Limit  int `json:"limit,omitempty" jsonschema:"items per page, at most 50"`
}

type repoOnly struct {
	Repo string `json:"repo,omitempty" jsonschema:"repo of the portfolio: its folder name or path; omit for the project this server runs in"`
	paging
}

type projectsIn struct{ paging }

type findingsIn struct {
	Repo  string `json:"repo,omitempty" jsonschema:"repo of the portfolio (folder name or path); omit for the project this server runs in"`
	Plane string `json:"plane,omitempty" jsonschema:"definition, governance, delivery or code"`
	Owner string `json:"owner,omitempty" jsonschema:"human or agent: whom the finding is up to"`
	Level string `json:"level,omitempty" jsonschema:"notice, warning or blocking"`
	Ref   string `json:"ref,omitempty" jsonschema:"only findings of this Epic, Story or other reference"`
	paging
}

type readyIn struct {
	Repo string `json:"repo,omitempty" jsonschema:"repo of the portfolio (folder name or path); omit for the project this server runs in"`
	Ref  string `json:"ref" jsonschema:"the Epic or Story, for example EPIC-001 or STORY-003"`
	paging
}

type needsIn struct {
	Repo string `json:"repo,omitempty" jsonschema:"one repo; omit to join what needs you in ALL repos of the portfolio"`
	paging
}

type epicIn struct {
	Repo string `json:"repo,omitempty" jsonschema:"repo of the portfolio (folder name or path); omit for the project this server runs in"`
	Epic string `json:"epic,omitempty" jsonschema:"one Epic with its Stories, for example EPIC-001; omit to list every Epic"`
	paging
}

type codeIn struct {
	Repo  string `json:"repo,omitempty" jsonschema:"repo of the portfolio (folder name or path); omit for the project this server runs in"`
	Epic  string `json:"epic,omitempty" jsonschema:"limit the design-to-code view to one Epic"`
	Story string `json:"story,omitempty" jsonschema:"limit the code map to one Story"`
	paging
}

func (p paging) args() app.MCPArgs { return app.MCPArgs{Offset: p.Offset, Limit: p.Limit} }

// NewMCPServer builds the server with its seven read-only tools. A tool never fails for what a repo cannot
// give (an older Bootstrap): it answers, and says which Bootstrap release is needed.
func NewMCPServer(a *app.App) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "gtt", Title: "GTT", Version: core.Version}, nil)
	readOnly := func(title string) *mcp.ToolAnnotations {
		no := false
		return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: &no}
	}
	reg := func(name, title, desc string) *mcp.Tool {
		return &mcp.Tool{Name: name, Title: title, Description: desc, Annotations: readOnly(title)}
	}
	handle := func(tool string, f func(ctx context.Context) (app.MCPAnswer, error)) (*mcp.CallToolResult, any, error) {
		ans, err := f(context.Background())
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, nil, nil
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: ans.Summary}, &mcp.TextContent{Text: ans.JSON()}}}, nil, nil
	}
	mcp.AddTool(s, reg(app.ToolProjects, "GTT projects", "The repos of the portfolio with Bootstrap version, phase and counts."),
		func(ctx context.Context, _ *mcp.CallToolRequest, in projectsIn) (*mcp.CallToolResult, any, error) {
			return handle(app.ToolProjects, func(context.Context) (app.MCPAnswer, error) {
				return a.MCPCall(ctx, app.ToolProjects, in.args())
			})
		})
	mcp.AddTool(s, reg(app.ToolFindings, "GTT findings", "The findings of a repo (what an Epic or Story lacks, gaps, drift), filtered by plane, owner, level or ref."),
		func(ctx context.Context, _ *mcp.CallToolRequest, in findingsIn) (*mcp.CallToolResult, any, error) {
			return handle(app.ToolFindings, func(context.Context) (app.MCPAnswer, error) {
				x := in.args()
				x.Repo, x.Plane, x.Owner, x.Level, x.Ref = in.Repo, in.Plane, in.Owner, in.Level, in.Ref
				return a.MCPCall(ctx, app.ToolFindings, x)
			})
		})
	mcp.AddTool(s, reg(app.ToolReady, "GTT ready", "What an Epic or Story still lacks to be ready, and whom each thing is up to."),
		func(ctx context.Context, _ *mcp.CallToolRequest, in readyIn) (*mcp.CallToolResult, any, error) {
			return handle(app.ToolReady, func(context.Context) (app.MCPAnswer, error) {
				x := in.args()
				x.Repo, x.Ref = in.Repo, in.Ref
				return a.MCPCall(ctx, app.ToolReady, x)
			})
		})
	mcp.AddTool(s, reg(app.ToolNeedsYou, "GTT needs you", "What needs the human (questions, decisions, definitions), for one repo or for all repos of the portfolio."),
		func(ctx context.Context, _ *mcp.CallToolRequest, in needsIn) (*mcp.CallToolResult, any, error) {
			return handle(app.ToolNeedsYou, func(context.Context) (app.MCPAnswer, error) {
				x := in.args()
				x.Repo = in.Repo
				return a.MCPCall(ctx, app.ToolNeedsYou, x)
			})
		})
	mcp.AddTool(s, reg(app.ToolEpic, "GTT Epic", "Progress, estimated vs active hours, forecast date or range of an Epic and its Stories."),
		func(ctx context.Context, _ *mcp.CallToolRequest, in epicIn) (*mcp.CallToolResult, any, error) {
			return handle(app.ToolEpic, func(context.Context) (app.MCPAnswer, error) {
				x := in.args()
				x.Repo, x.Epic = in.Repo, in.Epic
				return a.MCPCall(ctx, app.ToolEpic, x)
			})
		})
	mcp.AddTool(s, reg(app.ToolActivity, "GTT activity", "What the agent is doing now and the latest events."),
		func(ctx context.Context, _ *mcp.CallToolRequest, in repoOnly) (*mcp.CallToolResult, any, error) {
			return handle(app.ToolActivity, func(context.Context) (app.MCPAnswer, error) {
				x := in.args()
				x.Repo = in.Repo
				return a.MCPCall(ctx, app.ToolActivity, x)
			})
		})
	mcp.AddTool(s, reg(app.ToolCode, "GTT code", "The module map, the dependencies the architecture does not allow, and design to code."),
		func(ctx context.Context, _ *mcp.CallToolRequest, in codeIn) (*mcp.CallToolResult, any, error) {
			return handle(app.ToolCode, func(context.Context) (app.MCPAnswer, error) {
				x := in.args()
				x.Repo, x.Epic, x.Story = in.Repo, in.Epic, in.Story
				return a.MCPCall(ctx, app.ToolCode, x)
			})
		})
	return s
}
