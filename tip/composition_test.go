package main

import (
	"testing"
)

// These cover the reply fields the edit session depends on. The reply is
// produced in-process by runLocalStrategy with encoding/json, so the scalar
// readers only have to survive the escaping json.Marshal actually emits.

func TestReplyStringExtractsTextFields(t *testing.T) {
	line := `{"consume":true,"composition":"ni","commit":"你",` +
		`"candidates":[{"text":"你","index":0}],"selectedCandidate":0}`
	if got := replyString(line, "composition"); got != "ni" {
		t.Errorf("composition: got %q want %q", got, "ni")
	}
	if got := replyString(line, "commit"); got != "你" {
		t.Errorf("commit: got %q want %q", got, "你")
	}
	if got := replyString(line, "missing"); got != "" {
		t.Errorf("a missing field must read as empty, got %q", got)
	}
	// A field name that appears inside a value must not be mistaken for one.
	quoted := `{"consume":true,"composition":"say \"ni\" now"}`
	if got := replyString(quoted, "composition"); got != `say "ni" now` {
		t.Errorf("escaped quotes must survive, got %q", got)
	}
}

func TestReplyHasDetectsCommit(t *testing.T) {
	withCommit := `{"consume":true,"composition":"","commit":"你"}`
	withoutCommit := `{"consume":true,"composition":"n"}`
	if !replyHas(withCommit, "commit") {
		t.Error("a commit reply must be recognised as one")
	}
	if replyHas(withoutCommit, "commit") {
		t.Error("a composition-only reply must not be read as a commit")
	}
}

func TestRestoreCandidatesIgnoresAServiceWithNoComposition(t *testing.T) {
	service := &tipImpl{}
	service.candidateUI.update("ni", []imeCandidate{{Text: "你"}}, 0, [4]int32{1, 2, 3, 4}, 96)
	// No composition and nothing on screen: a remembered list alone must not be
	// restored, because no keystroke could consume a pick from it.
	service.restoreCandidates()
	if service.candidateUI.snapshot().Composition != "ni" {
		t.Error("restoreCandidates must not touch the model when there is no composition")
	}
}

func TestRestoreCandidatesIgnoresNothingRemembered(t *testing.T) {
	service := &tipImpl{}
	service.composition.Store(1)
	service.compositionContext.Store(1)
	service.restoreCandidates()
	if service.candidateUI.snapshot().Visible {
		t.Error("a service that never showed a list must stay hidden on focus")
	}
}

// Losing focus hides the list without destroying it, and regaining focus while
// the composition is still alive brings the same list back.
func TestRestoreCandidatesPutsBackASuspendedList(t *testing.T) {
	service := &tipImpl{}
	service.composition.Store(1)
	service.compositionContext.Store(1)
	service.candidateUI.update("nihao", []imeCandidate{{Text: "你好"}, {Text: "尼豪"}}, 1, [4]int32{1, 2, 3, 4}, 96)

	service.candidateUI.suspend()
	if service.candidateUI.snapshot().Visible {
		t.Fatal("suspend must hide the list")
	}
	service.restoreCandidates()
	snapshot := service.candidateUI.snapshot()
	if !snapshot.Visible {
		t.Fatal("restoreCandidates must show the list again")
	}
	if snapshot.Composition != "nihao" || snapshot.Selected != 1 || len(snapshot.Candidates) != 2 {
		t.Fatalf("the restored list lost state: %+v", snapshot)
	}
}

// The range is sized in UTF-16 code units, not bytes and not runes: SetText
// writes that many units into the document and a composition covering a
// different count would replace text it does not own.
func TestUtf16UnitsCountsUTF16CodeUnits(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "ascii pinyin", text: "ni", want: 2},
		{name: "one chinese character", text: "\u4f60", want: 1},
		{name: "mixed commit", text: "\u4f60ni", want: 3},
		{name: "astral plane needs a surrogate pair", text: "\U0001f600", want: 2},
	} {
		if got := utf16Units(tc.text); got != tc.want {
			t.Errorf("%s: utf16Units(%q) = %d, want %d", tc.name, tc.text, got, tc.want)
		}
	}
	// A text with an embedded NUL cannot be encoded at all: UTF16FromString
	// rejects it, so no range may be sized from it.
	if got := utf16Units("ni\x00"); got != -1 {
		t.Errorf("utf16Units of text with an embedded NUL = %d, want -1", got)
	}
}

func TestWrittenCompositionStartsEmptyAndFollowsEachWrite(t *testing.T) {
	service := &tipImpl{}
	if got := service.writtenComposition(); got != "" {
		t.Errorf("a fresh service has written %q, want nothing", got)
	}
	service.storeWrittenComposition("n")
	if got := service.writtenComposition(); got != "n" {
		t.Errorf("writtenComposition() = %q, want %q", got, "n")
	}
	// Ending or dropping the composition must clear it, otherwise the next
	// composition would replace text that no longer belongs to it.
	service.dropComposition()
	if got := service.writtenComposition(); got != "" {
		t.Errorf("after dropComposition the service has written %q, want nothing", got)
	}
}
