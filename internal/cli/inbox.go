package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// inboxCmd is `gtt inbox`: every change in one pipeline. Listing and showing only read; park, reject, reopen,
// priority and drop are acts of the human: each shows what it will do and asks (--yes pre-authorizes).
func inboxCmd(e *env) *cobra.Command {
	var q app.InboxQuery
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Everything that is on its way to becoming work, by stage",
		Long: "The inbox holds every idea you wrote and every proposal the agent made, from new to closed.\n" +
			"You write (gtt idea add, gtt adr new), decide (gtt answer N k) and ratify (gtt promote, gtt approve, gtt run).\n" +
			"park, reject, reopen, priority and drop are maintenance acts, yours alone.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Inbox(cmd.Context(), q)
			return e.emit(res, err, func(p output.Printer) { renderInbox(p, res) })
		},
	}
	cmd.Flags().BoolVar(&q.All, "all", false, "also list what is closed, parked or rejected")
	cmd.Flags().StringVar(&q.Origin, "origin", "", "human (ideas) or agent (proposals)")
	act := func(use, short string, run func(cmd *cobra.Command, ref string, reason string) (app.InboxResult, error), reason bool) *cobra.Command {
		var why string
		c := &cobra.Command{Use: use + " <REF>", Short: short, Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				res, err := run(cmd, args[0], why)
				return e.emit(res, err, func(p output.Printer) { renderInboxAct(p, res) })
			}}
		if reason {
			c.Flags().StringVar(&why, "reason", "", "why, in one line (it goes in the trail)")
		}
		return c
	}
	show := &cobra.Command{Use: "show <REF>", Short: "One entry with its trail", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.InboxShow(cmd.Context(), args[0])
			return e.emit(res, err, func(p output.Printer) {
				if res.Message != "" {
					p.Line("%s", res.Message)
					return
				}
				p.Line("%s", strings.TrimRight(res.Entry.Text, "\n"))
			})
		}}
	priority := &cobra.Command{Use: "priority <REF> urgent|normal|low", Short: "Change the priority of an accepted entry (a human act)", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.InboxPriority(cmd.Context(), args[0], args[1])
			return e.emit(res, err, func(p output.Printer) { renderInboxAct(p, res) })
		}}
	migrate := &cobra.Command{Use: "migrate", Short: "Bring a 1.22 project to the inbox (gtt update does it)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.InboxMigrate(cmd.Context())
			return e.emit(res, err, func(p output.Printer) {
				switch {
				case !res.Offered:
					p.Line("this Bootstrap does not offer the inbox: gtt update")
				case res.Applied || res.Report != "":
					p.Line("%s", res.Report)
				}
			})
		}}
	cmd.AddCommand(show,
		act("park", "Set an entry aside without a review (a human act)", func(c *cobra.Command, r, w string) (app.InboxResult, error) {
			return e.app.InboxPark(c.Context(), r, w)
		}, true),
		act("reject", "Discard a proposal without a review (a human act)", func(c *cobra.Command, r, w string) (app.InboxResult, error) {
			return e.app.InboxReject(c.Context(), r, w)
		}, true),
		act("reopen", "Bring a parked entry back (a human act)", func(c *cobra.Command, r, _ string) (app.InboxResult, error) {
			return e.app.InboxReopen(c.Context(), r)
		}, false),
		act("drop", "Discard an idea (a human act)", func(c *cobra.Command, r, w string) (app.InboxResult, error) {
			return e.app.InboxDrop(c.Context(), r, w)
		}, true),
		priority, migrate)
	return cmd
}

func renderInboxAct(p output.Printer, r app.InboxResult) {
	if r.Message != "" {
		p.Line("%s", r.Message)
		return
	}
	p.Line("%s", r.Output)
}

// renderInbox lists the entries by stage: the ref, the title, where it came from, how long it has been there
// and, when something waits for you, the command that answers it.
func renderInbox(p output.Printer, r app.InboxList) {
	if r.Message != "" {
		p.Line("%s", r.Message)
		return
	}
	if len(r.Entries) == 0 {
		p.Line("The inbox is empty.")
		return
	}
	for _, stage := range app.StageOrder() {
		var rows []app.InboxEntry
		for _, x := range r.Entries {
			if x.Stage == stage {
				rows = append(rows, x)
			}
		}
		if len(rows) == 0 {
			continue
		}
		p.Line(app.StageHeading(stage), len(rows))
		for _, x := range rows {
			line := "  " + x.Ref + "  " + x.Title
			if x.From != "" {
				line += "  <- " + x.From
			}
			if x.Priority != "" {
				line += "  [" + x.Priority + "]"
			}
			p.Line("%s", line)
			if x.Command != "" {
				p.Line("      -> %s", x.Command)
			}
		}
	}
}

// adrCmd is `gtt adr`: the architect's door to the inbox.
func adrCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adr",
		Short: "The architect's ADR drafts: new, adopt",
		Long: "gtt adr new \"<title>\"       start a draft in gtt-domain/adr/drafts/ (your text, or --file)\n" +
			"gtt adr adopt <file>         take an ADR you wrote by hand in gtt-domain/adr/ into the inbox",
	}
	var file, by string
	nw := &cobra.Command{Use: "new <title>", Short: "Start an ADR draft (a human act)", Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.AdrNew(cmd.Context(), strings.Join(args, " "), file, by)
			return e.emit(res, err, func(p output.Printer) { renderInboxAct(p, res) })
		}}
	nw.Flags().StringVar(&file, "file", "", "start from this text instead of the template")
	nw.Flags().StringVar(&by, "by", "", "who writes it")
	var by2 string
	adopt := &cobra.Command{Use: "adopt <file>", Short: "Take an ADR written by hand into the inbox (a human act)", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.AdrAdopt(cmd.Context(), args[0], by2)
			return e.emit(res, err, func(p output.Printer) { renderInboxAct(p, res) })
		}}
	adopt.Flags().StringVar(&by2, "by", "", "who adopts it")
	cmd.AddCommand(nw, adopt)
	return cmd
}
