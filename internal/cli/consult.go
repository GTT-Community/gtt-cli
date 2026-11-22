package cli

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// consultCmd is `gtt consult`: the thread in Claude web where the questions that need thinking are taken. Sonar's top bar is where a
// person works it (it copies the text and opens the thread); these are the same pieces for the terminal and for scripts.
func consultCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consult",
		Short: "The Claude web thread for the questions that need thinking (first message, open questions, answers)",
		Long: "A question that needs thinking goes to ONE conversation per project in Claude web, where Claude holds the project's context.\n" +
			"Sonar's top bar copies the text and opens the thread; these commands are the same pieces for the terminal.\n\n" +
			"gtt consult                  where things stand: the thread, the open questions, what changed in the design\n" +
			"gtt consult seed             the first message of the thread (summaries of the design, each with its anchor)\n" +
			"gtt consult batch [REF]      the open questions (or one) as a message for the thread\n" +
			"gtt consult set <URL>        register the thread (https://claude.ai/chat/<id> or /project/<id>)\n" +
			"gtt consult clear | seen     forget the thread | take it as up to date with today's design\n" +
			"gtt consult answers          read the «Respuesta para GTT:» block (--text, or on stdin) and give you the `gtt answer` command of each\n\n" +
			"Nothing is sent: you paste, and the answer comes back as a block you paste into your agent.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runConsult(e, cmd, "consult.status", nil, false) },
	}
	cmd.AddCommand(
		&cobra.Command{Use: "status", Short: "Where things stand", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return runConsult(e, cmd, "consult.status", nil, false) }},
		&cobra.Command{Use: "seed", Short: "The first message of the thread", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return runConsult(e, cmd, "consult.seed", nil, false) }},
		&cobra.Command{Use: "batch [REF]", Short: "The open questions (or one) as a message for the thread", Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				a := map[string]string{}
				if len(args) == 1 {
					a["ref"] = args[0]
				}
				return runConsult(e, cmd, "consult.batch", a, false)
			}},
		&cobra.Command{Use: "set <URL>", Short: "Register the thread (a Claude web conversation or project)", Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runConsult(e, cmd, "consult.set", map[string]string{"url": args[0]}, true)
			}},
		&cobra.Command{Use: "clear", Short: "Forget the thread", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return runConsult(e, cmd, "consult.clear", nil, true) }},
		&cobra.Command{Use: "seen", Short: "Take the thread as up to date with today's design", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error { return runConsult(e, cmd, "consult.seen", nil, true) }},
		consultAnswersCmd(e),
	)
	return cmd
}

func consultAnswersCmd(e *env) *cobra.Command {
	var text string
	cmd := &cobra.Command{
		Use:   "answers",
		Short: "Read the «Respuesta para GTT:» block the thread ended with and give you the commands",
		Long: "Reads the pasted block (--text, or on stdin) and, for each `13: 1` line, checks the answer against that question and gives you the exact\n" +
			"command to record it in your terminal: gtt answer 13 1. It records nothing itself: an answer is yours, and the agent that you paste the\n" +
			"block into records it in your words. An answer to a question that is not open is refused, never invented.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(text) == "" {
				if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice == 0 {
					raw, _ := io.ReadAll(io.LimitReader(os.Stdin, 64*1024))
					text = string(raw)
				}
			}
			if strings.TrimSpace(text) == "" {
				return &core.Error{Code: core.ExitUsage, Unmodified: true, What: "There is no block to read.",
					Next: "Pass it with --text '<block>' or on stdin: gtt consult answers < block.txt"}
			}
			// An operation argument cannot carry a line break (the contract refuses control characters): the block goes as one line with
			// the breaks written `\n`, which the Bootstrap reads back.
			flat := strings.NewReplacer("\r\n", `\n`, "\n", `\n`, "\r", `\n`, "\t", " ").Replace(text)
			return runConsult(e, cmd, "consult.answers", map[string]string{"text": flat}, false)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "the pasted block (or give it on stdin)")
	return cmd
}

func runConsult(e *env, cmd *cobra.Command, op string, args map[string]string, apply bool) error {
	res, err := e.app.Consult(cmd.Context(), op, args, apply)
	return e.emit(res, err, func(p output.Printer) { p.Line("%s", res.Output) })
}
