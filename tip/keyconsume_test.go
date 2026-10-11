package main

import (
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"github.com/zzl/go-com/com"
)

// These cover the rule that decides whether TSF keeps a key. The reducer runs
// in-process now, so a reply that never arrives can only mean the reduction
// itself failed, and the key still has to go back to the application.

func TestKeyDecisionFollowsReplyConsume(t *testing.T) {
	consumed := `{"consume":true,"composition":"n","candidates":[{"text":"你","index":0}]}`
	cases := []struct {
		name    string
		reply   string
		replied bool
		want    bool
	}{
		{"consume true is eaten", consumed, true, true},
		{"consume false passes through", `{"consume":false}`, true, false},
		{"a reply that never arrived must not eat the key", "", false, false},
		{"a lost reply must not eat the key", consumed, false, false},
		{"a reduction that produced nothing must not eat the key", `not json`, true, false},
	}
	for _, c := range cases {
		if got := keyDecision(c.reply, c.replied); got != c.want {
			t.Errorf("%s: keyDecision(%q, %v) = %v, want %v", c.name, c.reply, c.replied, got, c.want)
		}
	}
}

func TestReplyConsumeReadsTheFlag(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{"true", `{"consume":true}`, true},
		{"false", `{"consume":false}`, false},
		{"whitespace tolerated", `{"consume": true}`, true},
		{"flag missing", `{"composition":"n"}`, false},
		{"flag text inside a composition is not a field", `{"consume":false,"composition":"consume:true"}`, false},
		{"first field wins over candidate text", `{"consume":true,"candidates":[{"text":"consume:false"}]}`, true},
	}
	for _, c := range cases {
		if got := replyConsume(c.line); got != c.want {
			t.Errorf("%s: replyConsume(%q) = %v, want %v", c.name, c.line, got, c.want)
		}
	}
}

func TestWinSpaceBypassOnlyClaimsSpaceWhileWinIsHeld(t *testing.T) {
	if winSpaceBypass('A') {
		t.Fatal("only VK_SPACE can be bypassed")
	}
	if got, want := winSpaceBypass(0x20), winHeld(); got != want {
		t.Fatalf("winSpaceBypass(VK_SPACE) = %v, want %v (a Windows key must be held)", got, want)
	}
}

// GetKeyState is called on every keystroke to check for Win+Space. Looking the
// export up in the wrong DLL panics there, which a guarded callback turns into a
// silent E_FAIL and no key is ever consumed.
func TestWinHeldReadsKeyboardState(t *testing.T) {
	for attempt := 0; attempt < 3; attempt++ {
		// The answer depends on what the keyboard is doing, but the call has to
		// return a real answer instead of panicking inside the lazy lookup.
		_ = winHeld()
	}
}

// The key callbacks must answer S_OK unconditionally: a failure HRESULT makes
// TSF drop the text service, which would look like an input method that cannot
// be switched to. The callbacks receive their real COM this pointer, because
// they need the service state that hangs off it. The ITfContext is null here, so
// no edit session is attempted; the write into a real context can only be
// exercised against a live TSF host.
func TestKeySinkCallbacksSurviveWithoutAContext(t *testing.T) {
	t.Setenv(strategyLogPathEnvVar, filepath.Join(t.TempDir(), "tip.log"))
	obj := com.NewComObj[tipComObj](&tipImpl{})
	defer obj.Release()
	sink := uintptr(unsafe.Pointer(&obj.keySinkComObj))
	callbacks := []struct {
		name string
		call func() uintptr
	}{
		{"OnTestKeyDown", func() uintptr { return keySinkOnTestKeyDown(sink, 0, 'N', 0, 0) }},
		{"OnKeyDown", func() uintptr { return keySinkOnKeyDown(sink, 0, 'N', 0, 0) }},
		{"OnTestKeyUp", func() uintptr { return keySinkOnTestKeyUp(sink, 0, 'N', 0, 0) }},
		{"OnKeyUp", func() uintptr { return keySinkOnKeyUp(sink, 0, 'N', 0, 0) }},
		{"OnPreservedKey", func() uintptr { return keySinkOnPreservedKey(sink, 0, 'N', 0) }},
	}
	for _, c := range callbacks {
		if got := c.call(); got != 0 {
			t.Errorf("%s returned 0x%x, want S_OK", c.name, got)
		}
	}
	// A null pfEaten must not take the host application down either.
	eat("unit", 0, true)
}

// eatenBy runs one key callback and reports what it wrote into pfEaten, which
// is the only thing TSF acts on. The BOOL has to be pinned: a local whose
// address is laundered through uintptr is stack-allocated, the callback grows
// the stack, and the write lands on the frame the goroutine has already left.
func eatenBy(t *testing.T, call func(uintptr) uintptr) bool {
	t.Helper()
	eaten := new(int32)
	*eaten = -1
	var pinner runtime.Pinner
	pinner.Pin(eaten)
	defer pinner.Unpin()
	if got := call(uintptr(unsafe.Pointer(eaten))); got != 0 {
		t.Fatalf("callback returned 0x%x, want S_OK", got)
	}
	if *eaten != 0 && *eaten != 1 {
		t.Fatalf("pfEaten = %d, which is neither false nor true", *eaten)
	}
	return *eaten == 1
}

// The whole key path end to end: OnTestKeyDown predicts consumption without
// touching state, OnKeyDown applies it, and what reaches pfEaten has to follow
// the reducer. This replaces the old named-pipe round trip, which exercised a
// transport that no longer exists.
func TestKeySinkEatDecisionFollowsTheReducer(t *testing.T) {
	t.Setenv(strategyLogPathEnvVar, filepath.Join(t.TempDir(), "tip.log"))
	service := &tipImpl{}
	obj := com.NewComObj[tipComObj](service)
	defer obj.Release()
	sink := uintptr(unsafe.Pointer(&obj.keySinkComObj))

	test := func(vk uintptr) bool {
		return eatenBy(t, func(p uintptr) uintptr {
			return keySinkOnTestKeyDown(sink, 0, vk, 0, p)
		})
	}
	press := func(vk uintptr) bool {
		return eatenBy(t, func(p uintptr) uintptr {
			return keySinkOnKeyDown(sink, 0, vk, 0, p)
		})
	}

	// Letters build a composition, so they are eaten.
	for _, letter := range []uintptr{'N', 'I', 'H', 'A', 'O'} {
		if !test(letter) {
			t.Fatalf("OnTestKeyDown must predict consumption for %q", string(rune(letter)))
		}
		if !press(letter) {
			t.Fatalf("OnKeyDown must consume %q", string(rune(letter)))
		}
	}
	if got := service.imeComposition; got != "nihao" {
		t.Fatalf("imeComposition = %q, want %q", got, "nihao")
	}

	// Escape ends the composition, and once there is none it belongs to the
	// application again.
	if !press(0x1B) {
		t.Fatal("Escape must be eaten while a composition is open")
	}
	if got := service.imeComposition; got != "" {
		t.Fatalf("imeComposition = %q after Escape, want it empty", got)
	}
	if press(0x1B) {
		t.Fatal("Escape with no composition must be left to the application")
	}

	// "ni" has several candidates, so the selection has room to move in it.
	for _, letter := range []uintptr{'N', 'I'} {
		if !press(letter) {
			t.Fatalf("OnKeyDown must consume %q", string(rune(letter)))
		}
	}
	if !press(0x28) {
		t.Fatal("ArrowDown must be consumed while a composition is open")
	}
	if got := service.imeSelected; got != 1 {
		t.Fatalf("imeSelected = %d after ArrowDown, want 1", got)
	}
	if got := service.imeComposition; got != "ni" {
		t.Fatalf("navigation changed the composition to %q", got)
	}
	if !press(0x26) {
		t.Fatal("ArrowUp must be consumed while a composition is open")
	}
	if got := service.imeSelected; got != 0 {
		t.Fatalf("imeSelected = %d after ArrowUp, want 0", got)
	}

	// A numeric pick commits the highlighted candidate and clears everything.
	if !press('2') {
		t.Fatal("a numeric pick must be consumed while a composition is open")
	}
	if service.imeComposition != "" || service.imeSelected != 0 {
		t.Fatalf("a pick must leave composition=%q selected=%d, want both cleared",
			service.imeComposition, service.imeSelected)
	}

	// A key with no mapping at all never reaches the reducer.
	if test(0x70) || press(0x70) {
		t.Fatal("F1 has no key mapping and must never be eaten")
	}
}

// Prediction must not mutate the composition: TSF calls OnTestKeyDown ahead of
// OnKeyDown for the same key, and applying it twice would double the pinyin.
func TestOnTestKeyDownDoesNotMutateComposition(t *testing.T) {
	t.Setenv(strategyLogPathEnvVar, filepath.Join(t.TempDir(), "tip.log"))
	service := &tipImpl{}
	obj := com.NewComObj[tipComObj](service)
	defer obj.Release()
	sink := uintptr(unsafe.Pointer(&obj.keySinkComObj))

	if !eatenBy(t, func(p uintptr) uintptr { return keySinkOnTestKeyDown(sink, 0, 'N', 0, p) }) {
		t.Fatal("OnTestKeyDown must predict consumption for 'N'")
	}
	if got := service.imeComposition; got != "" {
		t.Fatalf("OnTestKeyDown mutated the composition to %q, want it left empty", got)
	}
	if !eatenBy(t, func(p uintptr) uintptr { return keySinkOnKeyDown(sink, 0, 'N', 0, p) }) {
		t.Fatal("OnKeyDown must consume 'N'")
	}
	if got := service.imeComposition; got != "n" {
		t.Fatalf("imeComposition = %q after one key, want %q", got, "n")
	}
}
