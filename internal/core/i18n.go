package core

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// LangSource ranks where a language choice came from: a lower one wins.
type LangSource int

const (
	// LangExplicit is --language, or the human's answer in init.
	LangExplicit LangSource = iota
	// LangProject is the language the project recorded.
	LangProject
	// LangEnv is GTT_LANG. The locale (LANG, LC_*) is never read: the
	// output must not depend on the machine it runs on.
	LangEnv
)

// Language is the interaction language of one invocation. English strings
// in the code are the keys; a catalog translates them at the output
// boundary. A string without a translation is shown in English. The zero
// value and a nil *Language speak English.
type Language struct{ by [3]string }

// Set records a language choice from one source; "" clears it.
func (l *Language) Set(from LangSource, code string) {
	if l == nil {
		return
	}
	code = strings.ToLower(strings.TrimSpace(code))
	if i := strings.IndexAny(code, "_-."); i > 0 {
		code = code[:i] // es_CL.UTF-8 -> es
	}
	l.by[from] = code
}

// Code is the language in force: the first choice by rank, else English.
func (l *Language) Code() string {
	if l != nil {
		for _, c := range l.by {
			if c != "" {
				return c
			}
		}
	}
	return "en"
}

// catalogs holds the translations by language; English has none.
var catalogs = map[string]map[string]string{"es": spanish}

func (l *Language) catalog() map[string]string { return catalogs[l.Code()] }

// Sprintf translates format before formatting, so its arguments survive.
// A string argument that is itself a known message is translated too (none,
// yes, a reason...). A format without a translation is formatted in English and
// then matched as a whole message (see T).
func (l *Language) Sprintf(format string, args ...any) string {
	cat := l.catalog()
	if cat == nil {
		return fmt.Sprintf(format, args...)
	}
	if t, ok := cat[format]; ok {
		return fmt.Sprintf(t, l.args(args)...)
	}
	return l.T(fmt.Sprintf(format, args...))
}

// T translates one complete message: an exact catalog entry, else an entry
// whose %s/%d slots match the message (messages built by concatenation).
// Surrounding blank space is kept; anything unknown stays English.
func (l *Language) T(s string) string {
	cat := l.catalog()
	if cat == nil || strings.TrimSpace(s) == "" {
		return s
	}
	if t, ok := cat[s]; ok {
		return t
	}
	body := strings.TrimSpace(s)
	lead, trail := s[:strings.Index(s, body)], s[strings.Index(s, body)+len(body):]
	exact, patterns := compiled(l.Code(), cat)
	if t, ok := exact[body]; ok {
		return lead + t + trail
	}
	for _, p := range patterns {
		if m := p.re.FindStringSubmatch(body); m != nil {
			args := make([]any, len(m)-1)
			for i, v := range m[1:] {
				args[i] = v
			}
			return lead + fmt.Sprintf(p.out, l.args(args)...) + trail
		}
	}
	return s
}

func (l *Language) args(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = a
		if s, ok := a.(string); ok {
			out[i] = l.T(s)
		}
	}
	return out
}

// pattern is a catalog entry with slots, matched against whole messages.
type pattern struct {
	re  *regexp.Regexp
	out string // the translation with every slot as %s (captures are text)
	lit int    // literal length: the most specific entry is tried first
}

// matcher is a catalog prepared for whole messages: its entries without
// surrounding blank space, and its entries with slots as patterns.
type matcher struct {
	exact    map[string]string
	patterns []pattern
}

var (
	verb     = regexp.MustCompile(`%(\[\d+\])?[sdvq]`)
	matchers = map[string]matcher{}
	compile  sync.Mutex
)

func compiled(code string, cat map[string]string) (map[string]string, []pattern) {
	compile.Lock()
	defer compile.Unlock()
	if m, ok := matchers[code]; ok {
		return m.exact, m.patterns
	}
	exact := map[string]string{}
	var ps []pattern
	for key, val := range cat {
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		exact[key] = val
		if !verb.MatchString(key) || strings.Contains(key, "%%") {
			continue
		}
		parts := verb.Split(key, -1)
		verbs := verb.FindAllString(key, -1)
		lit := len(strings.Join(parts, ""))
		if lit < 4 {
			continue // too little text to recognize a message by
		}
		var re strings.Builder
		re.WriteString(`(?s)^`)
		for i, part := range parts {
			re.WriteString(regexp.QuoteMeta(part))
			if i < len(verbs) {
				if strings.HasSuffix(verbs[i], "d") {
					re.WriteString(`(-?\d+)`)
				} else {
					re.WriteString(`(.*?)`)
				}
			}
		}
		re.WriteString(`$`)
		ps = append(ps, pattern{re: regexp.MustCompile(re.String()), out: verb.ReplaceAllString(val, "%${1}s"), lit: lit})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].lit != ps[j].lit {
			return ps[i].lit > ps[j].lit
		}
		return ps[i].re.String() < ps[j].re.String()
	})
	matchers[code] = matcher{exact: exact, patterns: ps}
	return exact, ps
}
