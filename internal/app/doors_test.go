package app

import "testing"

// The Sonar door reaches four operations and no ratification verb: approving, promoting, running, undoing,
// freezing and adopting an ADR are never available to a page (criterion 15 of the order).
func TestSonarDoorIsTheClosedListAndRatifiesNothing(t *testing.T) {
	want := map[string]bool{"backlog.set": true, "inbox.add": true, "pending.answer": true, "sonar.prefs": true}
	got := DoorOperations(DoorSonar)
	if len(got) != len(want) {
		t.Fatalf("the Sonar door lists %v, want exactly %v", got, want)
	}
	for _, op := range got {
		if !want[op] {
			t.Errorf("%s is not one of the four operations of table 2.1", op)
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
		if op != "sonar.prefs" && !HumanAuthorityAllowed(op) {
			t.Errorf("%s is an act of the human and must be on the human-authority allow list", op)
		}
	}
	if DoorAllows("unknown", "backlog.set") {
		t.Error("an unknown door reaches nothing")
	}
}
