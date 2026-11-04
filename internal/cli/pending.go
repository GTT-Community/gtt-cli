package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// pendingCmd is `gtt pending`: what waits for the human in gtt-domain/pending/.
// It only reads; answering is `gtt answer`.
func pendingCmd(e *env) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "pending",
		Short: "List what waits for you: questions to answer and decisions to see",
		Long: "Anything that needs you is one small file in gtt-domain/pending/. This lists the open questions and the\n" +
			"decisions the agent took, with the number to answer: gtt answer 14 1. The folder is empty when nothing needs you.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.PendingEvents(cmd.Context(), all)
			return e.emit(res, err, func(p output.Printer) { renderPending(p, res) })
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "also list the events already marked as attended")
	return cmd
}

// answerCmd is `gtt answer`: the human's answer to a pending event. There is no
// --yes: the command is the confirmation, and an agent is denied it.
func answerCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "answer <n> <answer> | ok",
		Short: "Answer a pending question or decision (a human act)",
		Long: "gtt answer 14 1        choose option 1 of question 14\n" +
			"gtt answer 15 ok       approve decision 15\n" +
			"gtt answer ok          approve every pending decision\n\n" +
			"It shows what it writes, then writes it. A key never goes in an answer: use .env.local. An agent is denied this command.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.PendingRespond(cmd.Context(), args[0], strings.Join(args[1:], " "))
			return e.emit(res, err, func(p output.Printer) {
				if len(res.Answered) > 0 {
					p.Line("Written. The agent applies it on its next message.")
				}
			})
		},
	}
}

func renderPending(p output.Printer, r app.PendingList) {
	if len(r.Events) == 0 {
		p.Line("Nothing pending.")
		return
	}
	for _, it := range r.Events {
		tail := ""
		switch {
		case it.State == "attended":
			tail = "  [attended]"
		case it.State == "answered":
			tail = "  [answered, not applied yet]"
		case it.Kind == "ASK":
			tail = "  → recommended " + strconv.Itoa(it.Recommended)
			if it.Blocks != "" {
				tail += " · blocks " + it.Blocks
			}
		}
		p.Line("%-4d %s  %s%s", it.Number, it.Kind, it.Title, tail)
		if it.Kind == "ASK" && it.State == "open" {
			renderForm(p, it)
		}
	}
}

// renderForm shows an open question in the form its file has: the topic, each option with what happens, and the two closing lines.
func renderForm(p output.Printer, it app.PendingItem) {
	if it.Topic != "" {
		p.Line("       %s", it.Topic)
	}
	for i, option := range it.Options {
		mark := ""
		if i+1 == it.Recommended {
			mark = " *"
		}
		p.Line("       %d. %s%s", i+1, option, mark)
		if note := it.Notes[strconv.Itoa(i+1)]; note != "" {
			p.Line("          %s", note)
		}
	}
	for i, line := range it.Closing {
		p.Line("       %d. %s", len(it.Options)+1+i, line)
	}
	p.Line("       answer: gtt answer %d <number>", it.Number)
}
