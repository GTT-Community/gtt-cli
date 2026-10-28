package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/output"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

func initCmd(e *env) *cobra.Command {
	var o app.InitOptions
	var ades, sources string
	var questionnaire, noQuestionnaire bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install and initialize GTT in this project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.ADEs, o.Sources = split(ades), split(sources)
			switch {
			case questionnaire && noQuestionnaire:
				return usage("--questionnaire and --no-questionnaire are mutually exclusive")
			case questionnaire:
				o.Questionnaire = "yes"
			case noQuestionnaire:
				o.Questionnaire = "no"
			}
			res, err := e.app.Init(cmd.Context(), o)
			if err == nil && res.Outcome == "already-initialized" && !e.opts.JSON && e.app.Prompt.Interactive() {
				return existing(e, cmd, res.Installed)
			}
			if err := e.emit(res, err, func(p output.Printer) { renderInit(p, res) }); err != nil {
				return err
			}
			return afterInit(e, cmd, res, o.Launch)
		},
	}
	addResolveFlags(cmd, &o.Resolve)
	f := cmd.Flags()
	f.StringVar(&ades, "ade", "", "participating ADEs, comma separated (ids from the Bootstrap registry)")
	f.StringVar(&o.Primary, "primary", "", "Primary ADE id")
	f.StringVar(&o.Language, "language", "", "interaction language")
	f.StringVar(&o.Method, "autonomy", "", "Autonomy Profile: how much GTT does without asking (light, medium, hard, team)")
	f.StringVar(&o.Method, "method", "", "former name of --autonomy")
	_ = f.MarkHidden("method")
	f.StringVar(&o.DevStyle, "dev-method", "", "development method: Method Pack ids, comma separated, or help (see: gtt dev-method list)")
	f.StringVar(&o.DevStart, "start", "", "how to start: a Method Pack id, direct, or help")
	f.StringVar(&sources, "sources", "", "initial source documents, comma separated (path or path:type)")
	f.BoolVar(&o.NoSources, "no-sources", false, "select no initial source document")
	f.BoolVar(&questionnaire, "questionnaire", false, "materialize the Bootstrap Initial Design Questionnaire")
	f.BoolVar(&noQuestionnaire, "no-questionnaire", false, "do not create the questionnaire")
	f.BoolVar(&o.Launch, "launch", false, "launch the Primary ADE when initialization completes")
	f.BoolVar(&o.Resume, "resume", false, "resume an interrupted initialization")
	f.BoolVar(&o.Rollback, "rollback", false, "roll back an interrupted initialization")
	f.StringVar(&o.Snapshot, "snapshot", "", "restore GTT from this recovery snapshot file")
	f.BoolVar(&o.NoRestore, "no-restore", false, "ignore recovery snapshots")
	return cmd
}

func split(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func renderInit(p output.Printer, r app.InitResult) {
	switch r.Outcome {
	case "already-initialized":
		p.Line("GTT is already initialized in this project. Nothing was changed.")
		renderCheck(p, r.Installed)
		if r.DevMethodNote != "" {
			p.Line("%s", r.DevMethodNote)
		}
		p.Line("See: gtt status")
		return
	case "updated":
		if u := r.Update; u != nil {
			p.Line("\nGTT Bootstrap updated: %s -> %s", u.From, u.To)
			p.Line("  replaced %d, added %d, removed %d, preserved (changed locally) %d", len(u.Replaced), len(u.Added), len(u.Removed), len(u.Preserved))
		}
		renderCheck(p, r.Installed)
		return
	case "rolled-back":
		p.Line("The incomplete initialization was rolled back. No GTT files remain.")
		return
	case "restored":
		p.Line("\nGTT installation restored from the recovery snapshot.")
	default:
		p.Line("\nGTT initialization complete.")
	}
	for _, s := range r.Sources {
		renderSource(p, s)
	}
	switch r.DesignState {
	case "questionnaire":
		if r.Handoff != nil {
			p.Line("\nQuestionnaire created:\n%s", r.Handoff.Questionnaire)
		}
	case "none":
		p.Line("\nNo design source was selected and no questionnaire was created:\nthe project has no initial design yet.")
	}
	if h := r.Handoff; h != nil && h.NextAction != "" {
		p.Line("\nNext:\n%s", h.NextAction)
	} else if h != nil && h.PrimaryName != "" {
		p.Line("\nNext:\nLaunch %s to continue the Bootstrap-guided initialization.", h.PrimaryName)
	}
}

// renderCheck reports the Bootstrap that was already in the project.
func renderCheck(p output.Printer, c *app.BootstrapCheck) {
	if c == nil {
		return
	}
	origin := "the release this GTT CLI ships with"
	if c.Required != c.Pinned {
		origin = "chosen explicitly; this GTT CLI ships with " + c.Pinned
	}
	if c.Verified {
		p.Line("GTT Bootstrap %s is present and verified (%s).", c.Version, origin)
	}
	if len(c.Modified) > 0 {
		p.Line("Bootstrap files modified in this project (kept; see them with: gtt update):\n  %s", strings.Join(c.Modified, "\n  "))
	}
}

func renderHandoff(p output.Printer, h *app.Handoff) {
	p.Line("Bootstrap handoff\n")
	p.Line("Primary ADE:\n  %s", h.PrimaryName)
	p.Line("\nInstruction entry (owned by Bootstrap):\n  %s", h.InstructionEntry)
	if len(h.ReadFirst) > 0 {
		p.Line("\nBootstrap entry points:\n  %s", strings.Join(h.ReadFirst, "\n  "))
	}
	if h.Questionnaire != "" {
		p.Line("\nInitial Design Questionnaire:\n  %s", h.Questionnaire)
	}
}

func afterInit(e *env, cmd *cobra.Command, r app.InitResult, launch bool) error {
	h := r.Handoff
	if h == nil || e.opts.JSON {
		return nil
	}
	if launch {
		return e.app.LaunchPrimary(cmd.Context())
	}
	if !e.app.Prompt.Interactive() || e.opts.Yes {
		return nil
	}
	options := []ports.Option{{Label: "Show Bootstrap handoff"}, {Label: "Exit"}}
	if h.Launchable {
		options = append([]ports.Option{{Label: "Launch " + h.PrimaryName}}, options...)
	}
	choice, err := e.app.Prompt.Select("Primary ADE: "+h.PrimaryName, options)
	if err != nil {
		return nil
	}
	switch options[choice].Label {
	case "Show Bootstrap handoff":
		return e.out.Emit(h, func(p output.Printer) { renderHandoff(p, h) })
	case "Exit":
		return nil
	}
	return e.app.LaunchPrimary(cmd.Context())
}

// existing is the menu shown when `gtt init` finds GTT already installed.
// It never reinstalls.
func existing(e *env, cmd *cobra.Command, check *app.BootstrapCheck) error {
	v := e.app.Version(cmd.Context())
	title := "GTT is already initialized in this project.\n\nCLI:        " + v.CLI
	if v.Bootstrap != nil {
		title = "GTT is already initialized in this project.\n\nBootstrap: " + v.Bootstrap.Version + "\nCLI:       " + v.CLI
	}
	if check != nil && check.Verified {
		title += "\n\nThe Bootstrap is present and verified."
		if n := len(check.Modified); n > 0 {
			title += fmt.Sprintf(" %d of its files are modified in this project (kept; Update shows them).", n)
		}
	}
	choice, err := e.app.Prompt.Select(title, []ports.Option{{Label: "Inspect"}, {Label: "Validate"}, {Label: "Update"},
		{Label: "Reconfigure operational settings"}, {Label: "Add design documents", Detail: "gtt design add"},
		{Label: "Start the design over (before the first freeze)", Detail: "gtt design reset"}, {Label: "Abort"}})
	if err != nil {
		return err
	}
	root := cmd.Root()
	run := func(args ...string) error {
		root.SetArgs(args)
		return root.ExecuteContext(cmd.Context())
	}
	switch choice {
	case 0:
		return run("inspect")
	case 1:
		return run("validate")
	case 2:
		return run("update")
	case 3:
		sub, err := e.app.Prompt.Select("Reconfigure:", []ports.Option{{Label: "Primary ADE"}, {Label: "Autonomy Profile"}, {Label: "Participating ADEs (agents sync)"}})
		if err != nil {
			return err
		}
		return run([][]string{{"config", "primary-ade"}, {"autonomy", "set"}, {"agents", "sync"}}[sub]...)
	case 4:
		answer, err := e.app.Prompt.Ask("Design document path(s), separated by spaces:")
		if err != nil {
			return err
		}
		if paths := strings.Fields(answer); len(paths) > 0 {
			return run(append([]string{"design", "add"}, paths...)...)
		}
		return core.ErrCancelled
	case 5:
		return run("design", "reset")
	}
	return core.ErrCancelled
}
