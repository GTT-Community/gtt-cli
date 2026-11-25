package app

// Doors: which declared operations each door into GTT may reach. A door is something other than the human
// at the terminal: the Sonar server today, the MCP of 1.26 tomorrow. This table is the ONE place that says
// so, and a test refuses a ratification verb or a human-authority operation outside what a door was given.
//
// The terminal CLI is not a door: there the human types the command, which is the confirmation.

// Door names.
const (
	DoorSonar = "sonar" // the local server of `gtt sonar`: the page of the human
	DoorMCP   = "mcp"   // `gtt mcp`: Claude, the agent, reading the portfolio over stdio. Read only
)

// Operations the Sonar server accepts, in the order of the order's table 2.1.
const (
	OpBacklogSet    = "backlog.set"
	OpInboxAdd      = "inbox.add"
	OpPendingAnswer = "pending.answer"
	OpSonarPrefs    = "sonar.prefs"
	OpConsultSet    = "consult.set" // the person's own thread in Claude web: registered once from the page (no human authority: it is their state)
)

// Operations the MCP door reads: three declared operations that change nothing.
const (
	OpSonarSnapshot = "sonar.snapshot"
	OpFindingsList  = "findings.list"
	OpCodeMap       = "code.map"
)

// The seven tools of `gtt mcp`, in the order the order's table 12.1 lists them.
const (
	ToolProjects = "gtt_projects"
	ToolFindings = "gtt_findings"
	ToolReady    = "gtt_ready"
	ToolNeedsYou = "gtt_needs_you"
	ToolEpic     = "gtt_epic"
	ToolActivity = "gtt_activity"
	ToolCode     = "gtt_code"
)

// MCPToolOrder lists the tools as they are offered.
var MCPToolOrder = []string{ToolProjects, ToolFindings, ToolReady, ToolNeedsYou, ToolEpic, ToolActivity, ToolCode}

// mcpToolOps is THE table of the MCP: which declared operations each tool may reach. It is the only place that
// says so: the server runs an operation for a tool only if it is listed here (mcpRun), and a test walks this table
// and fails if a tool reaches an operation that mutates, one with human_authority, or a ratification verb.
// Tomorrow's actions for the agent go here too, and only operations without human_authority may.
var mcpToolOps = map[string][]string{
	ToolProjects: {OpSonarSnapshot},
	ToolFindings: {OpFindingsList},
	ToolReady:    {OpFindingsList},
	ToolNeedsYou: {OpSonarSnapshot},
	ToolEpic:     {OpSonarSnapshot},
	ToolActivity: {OpSonarSnapshot},
	ToolCode:     {OpCodeMap},
}

// MCPToolOperations lists what a tool may reach (a copy).
func MCPToolOperations(tool string) []string { return append([]string(nil), mcpToolOps[tool]...) }

func mcpToolAllows(tool, op string) bool {
	for _, o := range mcpToolOps[tool] {
		if o == op {
			return true
		}
	}
	return false
}

// doorOps is the closed list per door. The Sonar server writes only what the human could write by hand,
// through the operation the CLI would run; it never ratifies.
var doorOps = map[string][]string{
	DoorSonar: {OpBacklogSet, OpInboxAdd, OpPendingAnswer, OpSonarPrefs, OpConsultSet},
	DoorMCP:   {OpSonarSnapshot, OpFindingsList, OpCodeMap}, // the union of mcpToolOps; a test keeps them equal
}

// DoorOperations lists what a door may reach (a copy).
func DoorOperations(door string) []string {
	return append([]string(nil), doorOps[door]...)
}

// DoorAllows reports whether a door may reach an operation.
func DoorAllows(door, op string) bool {
	for _, o := range doorOps[door] {
		if o == op {
			return true
		}
	}
	return false
}

// humanAuthorityAllowed are the operations with human authority that a door may still reach: the human is
// the one at the page, and each of these is something the human could write by hand.
var humanAuthorityAllowed = map[string]bool{OpBacklogSet: true, OpInboxAdd: true, OpPendingAnswer: true}

// HumanAuthorityAllowed reports whether a human-authority operation is one a door may carry at all.
func HumanAuthorityAllowed(op string) bool { return humanAuthorityAllowed[op] }

// RatificationVerb reports whether an operation is the ratification of something: approving, promoting,
// running, undoing, freezing, adopting an ADR. No door may reach one; the page shows the command to copy.
func RatificationVerb(op string) bool {
	switch op {
	case "approve", "promote", "undo", "freeze", "run.execute", "adr.adopt", "unfreeze", "inbox.promote":
		return true
	}
	return false
}
