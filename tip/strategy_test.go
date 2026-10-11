package main

import "testing"

func TestReduceIMEKeyLifecycle(t *testing.T) {
	s := imeState{}
	for _, key := range []string{"n", "i"} {
		r := reduceIME(s, key)
		if !r.consume {
			t.Fatalf("key %q should be consumed", key)
		}
		s = r.state
	}
	if s.composition != "ni" {
		t.Fatalf("composition = %q, want ni", s.composition)
	}
	if got := candidatesFor(s.composition); len(got) == 0 || got[0].Text == "" {
		t.Fatalf("expected embedded dictionary candidates for ni, got %#v", got)
	}
	if got := candidatesFor("nihao"); len(got) == 0 {
		t.Fatal("expected segmented phrase candidates for nihao")
	}
	r := reduceIME(s, " ")
	if !r.consume || r.commit == nil || *r.commit == "" || r.state.composition != "" {
		t.Fatalf("unexpected commit reduction: %#v", r)
	}
}

func TestTestKeyPredictionDoesNotMutateState(t *testing.T) {
	service := &tipImpl{imeComposition: "ni", imeSelected: 0}
	beforeText, beforeSelected := service.imeComposition, service.imeSelected
	_, consume := service.runLocalStrategy("testKeyDown", uintptr('1'))
	if !consume {
		t.Fatal("first candidate digit should be predicted as consumed")
	}
	if service.imeComposition != beforeText || service.imeSelected != beforeSelected {
		t.Fatal("test-key phase mutated IME state")
	}
}

func TestReduceIMEEscapeAndEmptyEditing(t *testing.T) {
	if r := reduceIME(imeState{}, "Backspace"); r.consume {
		t.Fatal("Backspace with no composition must pass through")
	}
	if r := reduceIME(imeState{composition: "ni"}, "Escape"); !r.consume || r.state.composition != "" {
		t.Fatalf("Escape should cancel composition: %#v", r)
	}
}
