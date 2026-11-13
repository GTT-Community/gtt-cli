package cli

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// sonarCmd is `gtt sonar`: the read-only page of everything GTT knows about the project (or the portfolio).
// It generates a derived file and opens it; it never runs, writes or decides anything in a project.
func sonarCmd(e *env) *cobra.Command {
	var all, log, noOpen bool
	var ro app.ResolveOptions
	cmd := &cobra.Command{
		Use:   "sonar [folder]",
		Short: "See what is being done and what comes next, in one page (read only)",
		Long: "gtt sonar generates the page of this repo from its own state and opens it. The page changes nothing: every act\n" +
			"that needs you is a command to copy and run here.\n\n" +
			"Inside a project it shows that project. In a folder that holds projects (gtt sonar ~/dev) it finds every GTT project\n" +
			"under it, to two levels, and shows them all: go to the folder, run it, and move between them in the page.\n\n" +
			"gtt sonar --all    the repos of `gtt sonar list` when you keep one, else the projects found under this folder\n" +
			"gtt sonar --log    the page opened on the latest activity",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				abs, aerr := filepath.Abs(args[0])
				if aerr != nil {
					return aerr
				}
				e.app.Cwd = abs
			}
			var res app.SonarResult
			var err error
			if all {
				res, err = e.app.SonarAll(cmd.Context(), ro)
			} else {
				res, err = e.app.Sonar(cmd.Context(), ro)
			}
			if err == nil && res.Page != "" && !noOpen && !e.opts.JSON {
				view := "view=portfolio"
				if !all && res.Page != "" && !strings.HasSuffix(res.Page, "index.html") {
					view = "view=project"
					if log {
						view = "view=last"
					}
				}
				target := (&url.URL{Scheme: "file", Path: res.Page}).String() + "#/" + view
				if oerr := e.app.OpenPage(cmd.Context(), target); oerr != nil {
					res.Message = "The page is ready but the browser could not be opened (" + oerr.Error() + "): open the file above."
				}
			}
			return e.emit(res, err, func(p output.Printer) {
				if res.Message != "" {
					p.Line("%s", res.Message)
				}
				if res.Page != "" {
					p.Line("Sonar page: %s", res.Page)
				}
				if len(res.Repos) > 0 {
					p.Line("%d project(s) found.", len(res.Repos))
				}
				for _, r := range res.Repos {
					if r.State != "ok" && r.State != "read" {
						p.Line("  %-14s %s - %s", r.State, r.Path, r.Reason)
					}
				}
			})
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "the portfolio of every repo in `gtt sonar list`")
	cmd.Flags().BoolVar(&log, "log", false, "open the page on the latest activity")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "generate the page and print its path without opening a browser")
	addResolveFlags(cmd, &ro)
	cmd.AddCommand(sonarAddCmd(e), sonarListCmd(e), sonarRemoveCmd(e))
	return cmd
}

func renderSonarList(p output.Printer, res app.SonarList) {
	if len(res.Repos) == 0 {
		p.Line("The portfolio is empty. Add a repo with: gtt sonar add <path>")
		return
	}
	for _, r := range res.Repos {
		note := strings.TrimSpace(r.Version + " " + r.Reason)
		p.Line("%-12s %s  %s", r.State, r.Path, note)
	}
}

func sonarAddCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a repo to the Sonar portfolio (a human act; the repo is not touched)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.SonarAdd(cmd.Context(), args[0])
			return e.emit(res, err, func(p output.Printer) { renderSonarList(p, res) })
		},
	}
}

func sonarListCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the repos of the portfolio and whether each one offers Sonar",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.SonarListRepos(cmd.Context())
			return e.emit(res, err, func(p output.Printer) { renderSonarList(p, res) })
		},
	}
}

func sonarRemoveCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <path>",
		Short: "Take a repo out of the Sonar portfolio (a human act; the repo is not touched)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.SonarRemove(cmd.Context(), args[0])
			return e.emit(res, err, func(p output.Printer) { renderSonarList(p, res) })
		},
	}
}

// ideaCmd is `gtt idea`: the human's loose ideas, one file each. add, park and drop are acts of the human (the
// command is the confirmation, and an agent is denied them).
func ideaCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "idea",
		Short: "Your ideas for the project: add, list, park, drop",
		Long: "An idea is your own words in docs/ideas/IDEA-NNN.md. GTT keeps the text exactly as you wrote it; the agent may only add\n" +
			"one line saying what became of it (covered by an Epic, a proposal, a non-goal, a question).",
	}
	var o app.IdeaAddOptions
	add := &cobra.Command{
		Use:   "add [text]",
		Short: "Write down an idea, or a change request with --type solicitud (a human act)",
		Long: "gtt idea add \"text\"                       an idea, exactly as you wrote it (or --file F)\n" +
			"gtt idea add --type solicitud --change \"what\" --reason \"why\"   a change request (it goes in now)\n\n" +
			"It shows what it will create, then asks. --now asks the agent to look at it at once.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Text = strings.Join(args, " ")
			res, err := e.app.IdeaAdd(cmd.Context(), o)
			return e.emit(res, err, func(p output.Printer) { renderIdea(p, res) })
		},
	}
	f := add.Flags()
	f.StringVar(&o.File, "file", "", "take the text from this file")
	f.BoolVar(&o.Now, "now", false, "ask the agent to triage it in this session")
	f.StringVar(&o.Type, "type", "", "idea (default) or solicitud")
	f.StringVar(&o.Change, "change", "", "solicitud: what needs to change")
	f.StringVar(&o.Reason, "reason", "", "solicitud: why")
	f.StringVar(&o.Trigger, "trigger", "", "solicitud: what caused it")
	f.StringVar(&o.Scope, "scope", "", "solicitud: what is in and out")
	f.StringVar(&o.Impact, "impact", "", "solicitud: what it touches")
	f.StringVar(&o.Risk, "risk", "", "solicitud: the risk")
	f.StringVar(&o.Priority, "priority", "", "solicitud: priority")
	f.StringVar(&o.By, "by", "", "who writes it")
	list := &cobra.Command{
		Use:   "list",
		Short: "List the ideas and where each stands",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			all, _ := cmd.Flags().GetBool("all")
			res, err := e.app.Ideas(cmd.Context(), all)
			return e.emit(res, err, func(p output.Printer) {
				if len(res.Ideas) == 0 {
					p.Line("No ideas.")
				}
				for _, i := range res.Ideas {
					to := ""
					if i.To != "" {
						to = "  -> " + i.To
					}
					p.Line("%-9s %-10s %s%s", i.Ref, i.State, i.Title, to)
				}
			})
		},
	}
	list.Flags().Bool("all", false, "also list the dropped ideas")
	park := &cobra.Command{
		Use: "park <IDEA-NNN>", Short: "Set an idea aside (a human act)", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.IdeaPark(cmd.Context(), args[0])
			return e.emit(res, err, func(p output.Printer) { renderIdea(p, res) })
		},
	}
	drop := &cobra.Command{
		Use: "drop <IDEA-NNN>", Short: "Discard an idea (a human act)", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.IdeaDrop(cmd.Context(), args[0])
			return e.emit(res, err, func(p output.Printer) { renderIdea(p, res) })
		},
	}
	cmd.AddCommand(add, list, park, drop)
	return cmd
}

func renderIdea(p output.Printer, r app.IdeaResult) {
	if r.Message != "" {
		p.Line("%s", r.Message)
		return
	}
	p.Line("%s", r.Output)
}
