// Package ports declares the interfaces through which use cases reach every
// external system, and the operational data they exchange. Nothing here
// carries GTT methodology: these are operational abstractions only.
package ports

import (
	"context"
	"encoding/json"
)

// ---------------------------------------------------------------- process

// Command is one structured process invocation: a trusted executable, its
// arguments, environment additions and working directory.
type Command struct {
	Path        string
	Args        []string
	Dir         string
	Env         []string
	Interactive bool // attach the terminal instead of capturing output
}

// Result is the outcome of a Command.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner executes processes.
type Runner interface {
	Run(ctx context.Context, c Command) (Result, error)
	LookPath(name string) (string, error)
}

// -------------------------------------------------------------- bootstrap

// Release is the Bootstrap release descriptor.
type Release struct {
	Bootstrap struct {
		ID            string `json:"id"`
		Version       string `json:"version"`
		SchemaVersion int    `json:"schema_version"`
		Channel       string `json:"channel"`
	} `json:"bootstrap"`
	Scaffold struct {
		Version  int    `json:"version"`
		Manifest string `json:"manifest"`
	} `json:"scaffold"`
	Canon         string `json:"canon"`
	Compatibility struct {
		CLI struct {
			MinVersion string  `json:"min_version"`
			MaxVersion *string `json:"max_version"`
		} `json:"cli"`
		Schema struct {
			Version int `json:"version"`
		} `json:"schema"`
	} `json:"compatibility"`
	RequiresCLICapabilities []string       `json:"requires_cli_capabilities"`
	SupportedLanguages      []string       `json:"supported_languages"`
	Contracts               map[string]int `json:"contracts"`
}

// Negotiation is the compatibility decision. Compatible false means the
// project must not be touched.
type Negotiation struct {
	Compatible bool     `json:"compatible"`
	Reasons    []string `json:"reasons"`
}

// Capability is one entry of the Bootstrap capability registry.
type Capability struct {
	ID         string   `json:"id"`
	Version    int      `json:"version"`
	Operations []string `json:"operations"`
	Summary    string   `json:"summary"`
}

// OperationArg is one declared argument of an operation.
type OperationArg struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Values   []string `json:"values"`
}

// Operation is one entry of the Bootstrap operation registry.
type Operation struct {
	Version        int            `json:"version"`
	Args           []OperationArg `json:"args"`
	Mutates        bool           `json:"mutates"`
	HumanAuthority bool           `json:"human_authority"`
	Outputs        struct {
		Format   string `json:"format"`
		Contract string `json:"contract"`
	} `json:"outputs"`
}

// HasArg reports whether the operation declares an argument.
func (o Operation) HasArg(name string) bool {
	for _, a := range o.Args {
		if a.Name == name {
			return true
		}
	}
	return false
}

// OperationRequest asks the Bootstrap to run one declared operation.
type OperationRequest struct {
	Operation string
	Args      map[string]string
	// Apply performs the change; without it a mutating operation is a dry run.
	Apply bool
	// ConfirmedByHuman must be set only after the human actually confirmed.
	ConfirmedByHuman bool
}

// OperationResult is the envelope the Bootstrap returns.
type OperationResult struct {
	Operation string
	ExitCode  int
	Stdout    string
	Stderr    string
	Format    string
	Applied   bool
}

// OK reports a zero exit of the implementation.
func (r OperationResult) OK() bool { return r.ExitCode == 0 }

// JSON decodes a JSON-format result into v.
func (r OperationResult) JSON(v any) error { return json.Unmarshal([]byte(r.Stdout), v) }

// Bootstrap is the runtime boundary to one Bootstrap tree (an installed
// project or a catalog). Implementations hide every Bootstrap detail.
type Bootstrap interface {
	Root() string
	Release(ctx context.Context) (Release, error)
	// Negotiate fails closed: any doubt is an incompatibility.
	Negotiate(ctx context.Context) (Negotiation, error)
	Capabilities(ctx context.Context) ([]Capability, error)
	Operations(ctx context.Context) (map[string]Operation, error)
	Show(ctx context.Context, contract string) (json.RawMessage, error)
	CheckContracts(ctx context.Context) ([]string, error)
	Execute(ctx context.Context, req OperationRequest) (OperationResult, error)
}

// BootstrapFactory binds a runtime to a directory.
type BootstrapFactory interface {
	At(root string) Bootstrap
	// Installed reports whether root holds a Bootstrap contract surface.
	Installed(root string) bool
}

// ResolveRequest states how the human asked for a Bootstrap.
type ResolveRequest struct {
	Path     string // explicit --bootstrap
	Version  string // --bootstrap-version
	Offline  bool
	Recorded string // source recorded by the project, if any
	Refresh  bool   // look for a newer release instead of reusing the cache
	Checksum string // expected checksum, if the caller has one
}

// Package is a resolved Bootstrap catalog on disk.
type Package struct {
	Root     string
	Source   string // local | recorded | cache | remote
	Origin   string // path or URL it came from
	Checksum string
	Release  Release
	// Integrity and Signature are the verification results, recorded as-is.
	Integrity string
	Signature string
}

// KeyPair names the files of a release signing key.
type KeyPair struct {
	ID      string `json:"key_id"`
	Private string `json:"private_key"`
	Public  string `json:"public_key"`
}

// SignResult describes a signed Bootstrap package.
type SignResult struct {
	Path     string `json:"signature"`
	KeyID    string `json:"key_id"`
	Version  string `json:"bootstrap_version"`
	Checksum string `json:"checksum"`
}

// ReleaseSigner creates release keys and signs Bootstrap packages. It is for
// whoever publishes a Bootstrap release; verification never uses it.
type ReleaseSigner interface {
	Keygen(dir, id string) (KeyPair, error)
	Sign(ctx context.Context, root, keyFile string) (SignResult, error)
}

// Resolver finds and verifies a Bootstrap catalog.
type Resolver interface {
	Resolve(ctx context.Context, req ResolveRequest) (Package, error)
	Verify(ctx context.Context, pkg *Package) error
}

// CoreLayout is what the scaffold manifest says the portable core is.
type CoreLayout struct {
	Paths []string // every top-level file and directory, slash separated
	// Package are the paths that hold Bootstrap package content (the engine
	// and the entry points); the governed domain is never among them.
	Package []string
	// Tracking is where the Bootstrap declares the initial-source tracking
	// record (manifest section `tracking:`); empty when it declares none.
	Tracking string
}

// Installer stages and commits the portable core into a project.
type Installer interface {
	Layout(pkg Package) (CoreLayout, error)
	Conflicts(pkg Package, projectRoot string) ([]string, error)
	// Install copies the core and returns the ledger path -> sha256.
	Install(pkg Package, projectRoot string) (map[string]string, error)
	Remove(projectRoot string, layout CoreLayout) error
}

// --------------------------------------------------------------------- ADE

// ADE is one entry of the Bootstrap ADE registry, reduced to what the CLI
// operates on.
type ADE struct {
	ID          string
	Name        string
	DetectPaths []string
	Binary      string
	Entry       string
	Enforcement string
	OwnedPaths  []string // project paths the Bootstrap declares as this ADE's
	Version     int      // version of the registry entry
}

// Candidate is a detected ADE: an observation, never a choice.
type Candidate struct {
	ADE
	Signals []string
	Status  string // Bootstrap-reported status when installed, else ""
}

// HandoffContract is what the CLI delivers to an ADE. Its content comes from
// the Bootstrap; the CLI never authors instructions.
type HandoffContract struct {
	ProjectRoot      string
	InstructionEntry string
	ReadFirst        []string
}

// AgentAdapter is the integration of one ADE/agent as the CLI operates it:
// how it is detected and which project locations hold its context. Every
// name and path comes from the Bootstrap ADE registry, so a new ADE needs a
// registry entry and no CLI change. Validating and synchronising the
// context are Bootstrap operations, not adapter methods.
type AgentAdapter interface {
	ID() string
	Name() string
	// Version is the version of the adapter's registry entry.
	Version() int
	// Detect returns the signals observed in the project or on the machine.
	// A signal makes the agent a candidate and nothing more.
	Detect(projectRoot string, runner Runner) []string
	// ContextLocations are the project paths the adapter manages.
	ContextLocations() []string
	// InstructionEntry is the file the agent reads first.
	InstructionEntry() string
}

// AgentRegistry is the set of agents GTT has an integration for.
type AgentRegistry interface {
	Adapters() []AgentAdapter
	Adapter(id string) (AgentAdapter, bool)
}

// ADEExecutor knows how to detect and start one ADE. It knows no methodology.
type ADEExecutor interface {
	ID() string
	Available() bool
	BuildCommand(contract HandoffContract) Command
	Execute(ctx context.Context, contract HandoffContract) error
}

// ---------------------------------------------------------------- terminal

// Option is one selectable entry.
type Option struct {
	Label  string
	Detail string
}

// Prompter asks the human. Every method returns core.ErrCancelled-compatible
// errors when input ends, and must not be used when Interactive is false.
type Prompter interface {
	Interactive() bool
	Confirm(question string, def bool) (bool, error)
	Select(title string, options []Option) (int, error)
	MultiSelect(title string, options []Option) ([]int, error)
}

// --------------------------------------------------------------------- git

// Git answers read-only questions about the repository.
type Git interface {
	Available() bool
	Root(ctx context.Context, dir string) (string, bool)
}

// ---------------------------------------------------------------- progress

// Reporter receives progress lines meant for a human. Implementations stay
// silent when machine-readable output was requested.
type Reporter interface {
	Info(format string, args ...any)
	// Detail is shown only when the human asked for detail.
	Detail(text string)
}

// Logger receives internal log lines (error, warn, info, debug). They go to
// stderr, never to the result, and must never carry tokens, credentials or
// environment values.
type Logger interface {
	Log(level, format string, args ...any)
}
