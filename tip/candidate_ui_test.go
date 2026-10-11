package main

import "testing"

func TestCandidateUIShowUpdateAndSelectedIndexClamping(t *testing.T) {
	var model candidateUIModel
	candidates := []imeCandidate{{Text: "你好", Index: 0}, {Text: "拟好", Index: 1}}
	model.update("nihao", candidates, 99, [4]int32{1, 2, 3, 4}, 144)
	s := model.snapshot()
	if !s.Visible || s.Composition != "nihao" || s.Selected != 0 || s.DPI != 144 {
		t.Fatalf("unexpected shown state: %+v", s)
	}
	if len(s.Candidates) != 2 || s.Candidates[0].Text != "你好" || s.Caret != [4]int32{1, 2, 3, 4} {
		t.Fatalf("candidate data not retained: %+v", s)
	}
	// A snapshot must not expose the model's backing slice.
	s.Candidates[0].Text = "mutated"
	if got := model.snapshot().Candidates[0].Text; got != "你好" {
		t.Fatalf("snapshot mutation leaked into model: %q", got)
	}
	model.update("nihao", candidates, 1, [4]int32{}, 0)
	if got := model.snapshot(); got.Selected != 1 || got.DPI != 96 {
		t.Fatalf("selection update/default DPI failed: %+v", got)
	}
}

func TestCandidateUIHidesForEmptyCompositionOrCandidates(t *testing.T) {
	var model candidateUIModel
	model.update("ni", []imeCandidate{{Text: "你", Index: 0}}, 0, [4]int32{}, 96)
	model.update("", []imeCandidate{{Text: "你", Index: 0}}, 0, [4]int32{}, 96)
	if s := model.snapshot(); s.Visible || len(s.Candidates) != 0 || s.Selected != 0 {
		t.Fatalf("empty composition should hide and clear candidates: %+v", s)
	}
	model.update("ni", nil, 5, [4]int32{}, 96)
	if s := model.snapshot(); s.Visible || len(s.Candidates) != 0 {
		t.Fatalf("empty candidate list should hide: %+v", s)
	}
}

func TestCandidateUIHideResetsLifecycleState(t *testing.T) {
	var model candidateUIModel
	model.update("nihao", []imeCandidate{{Text: "你好", Index: 0}}, 0, [4]int32{1, 2, 3, 4}, 120)
	model.hide()
	if s := model.snapshot(); s.Visible || s.Composition != "" || len(s.Candidates) != 0 || s.DPI != 96 {
		t.Fatalf("hide did not reset state: %+v", s)
	}
}

func TestCandidateUISuspendHidesWithoutLosingTheList(t *testing.T) {
	var model candidateUIModel
	model.update("nihao", []imeCandidate{{Text: "你好", Index: 0}}, 0, [4]int32{}, 120)
	model.suspend()
	if s := model.snapshot(); s.Visible {
		t.Fatalf("suspend must hide the list: %+v", s)
	}
	if s := model.snapshot(); s.Composition != "nihao" || len(s.Candidates) != 1 {
		t.Fatalf("suspend must keep the list so focus can restore it: %+v", s)
	}
	model.resume()
	if s := model.snapshot(); !s.Visible {
		t.Fatalf("resume must show the list again: %+v", s)
	}
	// Resuming after a hide must not resurrect a list that is gone.
	model.hide()
	model.resume()
	if s := model.snapshot(); s.Visible {
		t.Fatalf("resume must not resurrect a hidden list: %+v", s)
	}
}

// The wiring between the reducer and the model is the only thing that makes the
// model reflect what the user typed: showCandidates is driven by exactly the
// reply applyKeyLocked got.
func TestNotifyCandidatesFollowsTheReducerReply(t *testing.T) {
	service := &tipImpl{}
	caret := [4]int32{10, 20, 30, 40}

	for _, letter := range []uintptr{'N', 'I', 'H', 'A', 'O'} {
		reply, replied := service.runLocalStrategy("keyDown", letter)
		if !replied {
			t.Fatalf("runLocalStrategy(%q) produced no reply", string(rune(letter)))
		}
		service.notifyCandidates(reply, caret)
	}

	s := service.candidateUI.snapshot()
	if !s.Visible {
		t.Fatalf("the candidate list must be visible after typing nihao: %+v", s)
	}
	if s.Composition != "nihao" {
		t.Fatalf("Composition = %q, want %q", s.Composition, "nihao")
	}
	if len(s.Candidates) == 0 {
		t.Fatal("nihao must produce candidates")
	}
	if s.Caret != caret {
		t.Fatalf("Caret = %v, want %v", s.Caret, caret)
	}
	if s.DPI == 0 {
		t.Fatal("the caret DPI must never be reported as 0")
	}

	// Backspace inside the composition regenerates the list rather than hiding
	// it, as long as candidates remain.
	reply, replied := service.runLocalStrategy("keyDown", 0x08)
	if !replied {
		t.Fatal("runLocalStrategy(Backspace) produced no reply")
	}
	service.notifyCandidates(reply, caret)
	if s := service.candidateUI.snapshot(); !s.Visible || s.Composition != "niha" {
		t.Fatalf("Backspace must keep the list open on the shorter composition: %+v", s)
	}
}

// Arrow navigation changes the selection without changing the composition, and
// the model has to follow it.
func TestNotifyCandidatesTracksArrowSelection(t *testing.T) {
	service := &tipImpl{}
	for _, letter := range []uintptr{'N', 'I'} {
		reply, _ := service.runLocalStrategy("keyDown", letter)
		service.notifyCandidates(reply, [4]int32{})
	}
	if got := service.candidateUI.snapshot().Selected; got != 0 {
		t.Fatalf("Selected = %d before any navigation, want 0", got)
	}
	for _, arrow := range []uintptr{0x28, 0x28} {
		reply, _ := service.runLocalStrategy("keyDown", arrow)
		service.notifyCandidates(reply, [4]int32{})
	}
	s := service.candidateUI.snapshot()
	if s.Selected != 2 {
		t.Fatalf("Selected = %d after two ArrowDown, want 2", s.Selected)
	}
	if s.Composition != "ni" {
		t.Fatalf("navigation must not change the composition, got %q", s.Composition)
	}
	// Selection is clamped to the list, so it can never point past the end.
	for i := 0; i < 20; i++ {
		reply, _ := service.runLocalStrategy("keyDown", 0x28)
		service.notifyCandidates(reply, [4]int32{})
	}
	if s := service.candidateUI.snapshot(); s.Selected >= len(s.Candidates) {
		t.Fatalf("Selected = %d with %d candidates, want it clamped", s.Selected, len(s.Candidates))
	}
}

// Escape, Space and a numeric pick all commit, and every one of them must take
// the list off the screen.
func TestNotifyCandidatesHidesOnCommitAndCancel(t *testing.T) {
	for _, tc := range []struct {
		name string
		vks  []uintptr
	}{
		{"escape", []uintptr{'N', 'I', 'H', 'A', 'O', 0x1B}},
		{"space", []uintptr{'N', 'I', 'H', 'A', 'O', 0x20}},
		{"numeric pick", []uintptr{'N', 'I', 'H', 'A', 'O', '1'}},
		{"enter", []uintptr{'N', 'I', 'H', 'A', 'O', 0x0D}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &tipImpl{}
			for _, vk := range tc.vks {
				reply, _ := service.runLocalStrategy("keyDown", vk)
				service.notifyCandidates(reply, [4]int32{1, 2, 3, 4})
			}
			s := service.candidateUI.snapshot()
			if s.Visible || s.Composition != "" || len(s.Candidates) != 0 {
				t.Fatalf("%s must leave no list behind: %+v", tc.name, s)
			}
		})
	}
}

// A reduction that failed to encode must not leave whatever was on screen from
// the previous key.
func TestNotifyCandidatesHidesOnMalformedReply(t *testing.T) {
	service := &tipImpl{}
	reply, _ := service.runLocalStrategy("keyDown", 'N')
	service.notifyCandidates(reply, [4]int32{})
	if !service.candidateUI.snapshot().Visible {
		t.Fatal("the first key must open the list")
	}
	service.notifyCandidates("not json", [4]int32{})
	if s := service.candidateUI.snapshot(); s.Visible {
		t.Fatalf("a malformed reply must hide the list: %+v", s)
	}
}

// Rapid consecutive keys must leave the model showing exactly the last
// reduction, not a blend of several.
func TestNotifyCandidatesConsecutiveKeysLeaveTheLastState(t *testing.T) {
	service := &tipImpl{}
	for _, letter := range []uintptr{'N', 'I', 'H', 'A', 'O'} {
		reply, _ := service.runLocalStrategy("keyDown", letter)
		service.notifyCandidates(reply, [4]int32{})
	}
	s := service.candidateUI.snapshot()
	if s.Composition != service.imeComposition {
		t.Fatalf("model shows %q but the reducer holds %q", s.Composition, service.imeComposition)
	}
	// The list has to be exactly the one the reducer would regenerate from the
	// state it holds now, not an accumulation of the intermediate ones.
	want := candidatesFor(service.imeComposition)
	if len(s.Candidates) != len(want) {
		t.Fatalf("the model holds %d candidates, the reducer produces %d", len(s.Candidates), len(want))
	}
	for i := range want {
		if s.Candidates[i].Text != want[i].Text {
			t.Errorf("candidate %d: model has %q, reducer has %q", i, s.Candidates[i].Text, want[i].Text)
		}
	}
}

// Unicode has to survive the whole path: pinyin in, Chinese out, through the
// same encoding the model hands to a presenter.
func TestCandidateUIKeepsUnicodeCandidatesIntact(t *testing.T) {
	service := &tipImpl{}
	for _, letter := range []uintptr{'N', 'I', 'H', 'A', 'O'} {
		reply, _ := service.runLocalStrategy("keyDown", letter)
		service.notifyCandidates(reply, [4]int32{})
	}
	s := service.candidateUI.snapshot()
	if len(s.Candidates) == 0 {
		t.Fatal("nihao must produce candidates")
	}
	for i, candidate := range s.Candidates {
		if candidate.Text == "" {
			t.Errorf("candidate %d has no text", i)
		}
		for _, r := range candidate.Text {
			if r == 0xFFFD {
				t.Errorf("candidate %d (%q) was mangled into a replacement rune", i, candidate.Text)
			}
		}
		if candidate.Annotation == "" {
			t.Errorf("candidate %d (%q) lost its pinyin annotation", i, candidate.Text)
		}
	}
	if s.Candidates[0].Text != "你好" {
		t.Fatalf("the top candidate for nihao is %q, want %q", s.Candidates[0].Text, "你好")
	}
}
