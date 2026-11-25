package app

import "testing"

// The Sonar door reaches five operations (the four of table 2.1 and the registering of the person's own thread) and no ratification verb: approving, promoting, running, undoing,
// freezing and adopting an ADR are never available to a page (criterion 15 of the order).
func TestSonarDoorIsTheClosedListAndRatifiesNothing(t *testing.T) {
	want := map[string]bool{"backlog.set": true, "inbox.add": true, "pending.answer": true, "sonar.prefs": true, "consult.set": true}
	got := DoorOperations(DoorSonar)
	if len(got) != len(want) {
		t.Fatalf("the Sonar door lists %v, want exactly %v", got, want)
	}
	for _, op := range got {
		if !want[op] {
			t.Errorf("%s is not one of the operations the Sonar door was given", op)
		}
		if RatificationVerb(op) {
			t.Errorf("%s is a ratification verb: no door may reach it", op)
		}
		if DoorAllows(DoorSonar, op) != true {
			t.Errorf("%s is listed but not allowed", op)
		}
	}
	for _, verb := range []string{"approve", "promote", "run.execute", "undo", "freeze", "adr.adopt"} {
		if !RatificationVerb(verb) {
			t.Errorf("%s must be known as a ratification verb", verb)
		}
		if DoorAllows(DoorSonar, verb) {
			t.Errorf("the Sonar door must not reach %s", verb)
		}
	}
	for _, op := range got {
		if op != "sonar.prefs" && op != "consult.set" && !HumanAuthorityAllowed(op) {
			t.Errorf("%s is an act of the human and must be on the human-authority allow list", op)
		}
	}
	if HumanAuthorityAllowed("consult.set") {
		t.Error("consult.set has no human authority: it is the person's own state, and must not be on the human-authority allow list")
	}
	if DoorAllows("unknown", "backlog.set") {
		t.Error("an unknown door reaches nothing")
	}
}
