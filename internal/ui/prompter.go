// Package ui is the terminal prompter.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Prompter asks questions on a reader/writer pair.
type Prompter struct {
	in          *bufio.Reader
	out         io.Writer
	interactive bool
}

// New builds a prompter. It is interactive only when stdin is a terminal
// and prompting was not disabled.
func New(in io.Reader, out io.Writer, disabled bool) *Prompter {
	interactive := false
	if f, ok := in.(*os.File); ok && !disabled {
		info, err := f.Stat()
		interactive = err == nil && info.Mode()&os.ModeCharDevice != 0
	}
	return &Prompter{in: bufio.NewReader(in), out: out, interactive: interactive}
}

// Scripted builds an always-interactive prompter for tests.
func Scripted(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), out: out, interactive: true}
}

// Interactive reports whether the human can be asked.
func (p *Prompter) Interactive() bool { return p.interactive }

func (p *Prompter) line() (string, error) {
	s, err := p.in.ReadString('\n')
	if err != nil && s == "" {
		return "", core.ErrCancelled
	}
	return strings.TrimSpace(s), nil
}

// Confirm asks a yes/no question.
func (p *Prompter) Confirm(question string, def bool) (bool, error) {
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	for {
		fmt.Fprintf(p.out, "%s %s ", question, hint)
		s, err := p.line()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(s) {
		case "":
			return def, nil
		case "y", "yes", "s", "si", "sí":
			return true, nil
		case "n", "no":
			return false, nil
		}
	}
}

func (p *Prompter) list(title string, options []ports.Option) {
	fmt.Fprintf(p.out, "\n%s\n\n", title)
	for i, o := range options {
		fmt.Fprintf(p.out, "  [%d] %s\n", i+1, o.Label)
		if o.Detail != "" {
			fmt.Fprintf(p.out, "      %s\n", o.Detail)
		}
	}
	fmt.Fprintln(p.out)
}

// Select asks for exactly one option. There is no default: silence is not
// an answer.
func (p *Prompter) Select(title string, options []ports.Option) (int, error) {
	p.list(title, options)
	for {
		fmt.Fprintf(p.out, "Select [1-%d]: ", len(options))
		s, err := p.line()
		if err != nil {
			return 0, err
		}
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
	}
}

// MultiSelect asks for zero or more options as a comma-separated list.
func (p *Prompter) MultiSelect(title string, options []ports.Option) ([]int, error) {
	p.list(title, options)
	for {
		fmt.Fprint(p.out, "Select one or more (comma separated, empty for none): ")
		s, err := p.line()
		if err != nil {
			return nil, err
		}
		if s == "" {
			return nil, nil
		}
		var picked []int
		seen := map[int]bool{}
		ok := true
		for _, part := range strings.Split(s, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || n < 1 || n > len(options) {
				ok = false
				break
			}
			if !seen[n-1] {
				seen[n-1] = true
				picked = append(picked, n-1)
			}
		}
		if ok {
			return picked, nil
		}
	}
}
