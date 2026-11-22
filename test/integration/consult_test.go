package integration

import (
	"strings"
	"testing"
)

// `gtt consult` is the terminal side of the thread in Claude web: where things stand, the first message, the open questions, the registration
// of the thread and the reading of the answer block. Sonar's top bar is where a person works it; these run the same Bootstrap operations.
func TestConsultThreadFromTheTerminal(t *testing.T) {
	root := installed(t, nil)
	const thread = "https://claude.ai/chat/0b6e1f2a-4c3d-4e5f-8a9b-1c2d3e4f5a6b"

	if out := gtt(t, root, "consult").must(t, 0).out; !strings.Contains(out, "No thread registered") || !strings.Contains(out, "open question") {
		t.Errorf("with no thread it says so and how many questions are open:\n%s", out)
	}
	seed := gtt(t, root, "consult", "seed").must(t, 0).out
	if !strings.Contains(seed, "Respuesta para GTT:") || !strings.Contains(seed, "You cannot see the repository") {
		t.Errorf("the first message tells Claude how to work and how to answer:\n%s", seed)
	}
	if out := gtt(t, root, "consult", "batch").must(t, 0).out; !strings.Contains(out, "no open questions") {
		t.Errorf("with nothing open there is no batch:\n%s", out)
	}

	// Only a Claude web conversation or project is a thread: anything else is refused and nothing is saved.
	gtt(t, root, "consult", "set", "https://evil.example/chat/0b6e1f2a-4c3d-4e5f-8a9b-1c2d3e4f5a6b").must(t, 1)
	gtt(t, root, "consult", "set", "https://claude.ai/new").must(t, 1)
	if out := gtt(t, root, "consult", "status").must(t, 0).out; strings.Contains(out, thread) || !strings.Contains(out, "No thread registered") {
		t.Errorf("a refused URL leaves no thread:\n%s", out)
	}
	gtt(t, root, "consult", "set", thread+"?utm=x").must(t, 0)
	if out := gtt(t, root, "consult").must(t, 0).out; !strings.Contains(out, thread) {
		t.Errorf("the registered thread (cleaned) is shown:\n%s", out)
	}
	if exists(root, ".gtt/local/consult.json") == false {
		t.Error("the thread lives in the person's own state, under .gtt/local")
	}

	// The answer block: no block is refused with the format; an answer to a question that is not open is refused, never invented. The CLI
	// records nothing (an answer is the person's, and the agent they paste it into records it in their words): it checks and gives the command.
	gtt(t, root, "consult", "answers", "--text", "no block here").must(t, 1)
	r := gtt(t, root, "consult", "answers", "--text", "Respuesta para GTT:\n99: 1").must(t, 1)
	if !strings.Contains(r.err+r.out, "no open question") {
		t.Errorf("the refusal says why:\n%s%s", r.out, r.err)
	}
	gtt(t, root, "consult", "answers", "--apply", "--text", "x").must(t, 3) // there is no --apply: it records nothing

	doc := gtt(t, root, "consult", "--json").must(t, 0).json(t)
	if doc["supported"] != true || doc["operation"] != "consult.status" {
		t.Errorf("machine readable: %v", doc)
	}
	gtt(t, root, "consult", "clear").must(t, 0)
	if out := gtt(t, root, "consult").must(t, 0).out; strings.Contains(out, thread) {
		t.Errorf("clearing forgets the thread:\n%s", out)
	}
}
