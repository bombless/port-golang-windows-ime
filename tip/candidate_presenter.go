package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

// candidatePresenter owns a private Walk UI thread. TSF callbacks only enqueue
// immutable snapshots; they never create windows, pump messages, or wait for UI.
type candidatePresenter struct {
	mu      sync.Mutex
	enabled bool
	started bool
	updates chan candidateUISnapshot
}

func (p *candidatePresenter) setEnabled(enabled bool) {
	p.mu.Lock()
	p.enabled = enabled
	updates := p.updates
	started := p.started
	p.mu.Unlock()
	candidateDebugLog("presenter.enabled enabled=%v started=%v", enabled, started)
	if started && !enabled {
		select {
		case updates <- candidateUISnapshot{DPI: 96}:
		default:
			select {
			case <-updates:
			default:
			}
			select {
			case updates <- candidateUISnapshot{DPI: 96}:
			default:
			}
		}
	}
}

func (p *candidatePresenter) publish(snapshot candidateUISnapshot) {
	candidateDebugLog("presenter.publish visible=%v candidate_count=%d selected_index=%d", snapshot.Visible, len(snapshot.Candidates), snapshot.Selected)
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.enabled {
		candidateDebugLog("presenter.publish.skipped reason=disabled")
		return
	}
	if !p.started {
		p.started = true
		p.updates = make(chan candidateUISnapshot, 1)
		go p.run()
		candidateDebugLog("presenter.thread.start")
	}
	updates := p.updates

	// Coalesce bursts of keystrokes and keep only the newest immutable snapshot.
	select {
	case updates <- snapshot:
	default:
		select {
		case <-updates:
		default:
		}
		select {
		case updates <- snapshot:
		default:
		}
	}
}

func (p *candidatePresenter) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var window *walk.MainWindow
	var contents *walk.Label
	err := (MainWindow{
		AssignTo: &window,
		Title:    "PortGo IME Candidates",
		Visible:  false,
		Size:     Size{Width: 320, Height: 40},
		Layout:   VBox{Margins: Margins{Left: 8, Top: 5, Right: 8, Bottom: 5}},
		Children: []Widget{Label{AssignTo: &contents, Text: ""}},
	}).Create()
	if err != nil {
		candidateDebugLog("presenter.window.create_failed error=%q", err.Error())
		return
	}

	// Convert the hidden Walk form into a borderless, non-activating tool popup.
	hwnd := window.Handle()
	style := uint32(win.WS_POPUP | win.WS_CLIPCHILDREN | win.WS_CLIPSIBLINGS)
	win.SetWindowLong(hwnd, win.GWL_STYLE, int32(style))
	exStyle := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	popupExStyle := uint32(exStyle) | uint32(win.WS_EX_TOOLWINDOW|win.WS_EX_NOACTIVATE|win.WS_EX_TOPMOST)
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, int32(popupExStyle))
	if !win.SetWindowPos(hwnd, win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED|win.SWP_HIDEWINDOW) {
		candidateDebugLog("presenter.window.style_failed")
	}
	candidateDebugLog("presenter.window.created")

	go func() {
		for snapshot := range p.updates {
			copySnapshot := snapshot
			window.Synchronize(func() { p.apply(window, contents, copySnapshot) })
		}
	}()
	window.Run()
	candidateDebugLog("presenter.window.closed")
}

func (p *candidatePresenter) apply(window *walk.MainWindow, contents *walk.Label, snapshot candidateUISnapshot) {
	if window == nil || contents == nil || window.IsDisposed() {
		candidateDebugLog("presenter.update.skipped reason=window_unavailable")
		return
	}
	if !snapshot.Visible || len(snapshot.Candidates) == 0 {
		win.ShowWindow(window.Handle(), win.SW_HIDE)
		candidateDebugLog("presenter.window.hidden")
		return
	}

	var lines []string
	for i, candidate := range snapshot.Candidates {
		marker := "  "
		if i == snapshot.Selected {
			marker = "▶ "
		}
		lines = append(lines, fmt.Sprintf("%s%d. %s", marker, i+1, candidate.Text))
	}
	if err := contents.SetText(strings.Join(lines, "\r\n")); err != nil {
		candidateDebugLog("presenter.content.update_failed")
		return
	}
	width := 24
	for _, line := range lines {
		if w := len([]rune(line))*14 + 24; w > width {
			width = w
		}
	}
	if width > 720 {
		width = 720
	}
	height := len(lines)*26 + 12
	if height > 246 {
		height = 246
	}
	left, top := int(snapshot.Caret[0]), int(snapshot.Caret[3])
	if snapshot.Caret[2] <= snapshot.Caret[0] || snapshot.Caret[3] <= snapshot.Caret[1] {
		var cursor win.POINT
		if win.GetCursorPos(&cursor) {
			left, top = int(cursor.X), int(cursor.Y)
		} else {
			left, top = 16, 16
		}
	}
	anchorTop := top
	virtualLeft := int(win.GetSystemMetrics(win.SM_XVIRTUALSCREEN))
	virtualTop := int(win.GetSystemMetrics(win.SM_YVIRTUALSCREEN))
	virtualWidth := int(win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN))
	virtualHeight := int(win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN))
	if virtualWidth <= 0 || virtualHeight <= 0 {
		virtualLeft, virtualTop = 0, 0
		virtualWidth, virtualHeight = int(win.GetSystemMetrics(win.SM_CXSCREEN)), int(win.GetSystemMetrics(win.SM_CYSCREEN))
	}
	if left+width > virtualLeft+virtualWidth {
		left = virtualLeft + virtualWidth - width
	}
	if left < virtualLeft {
		left = virtualLeft
	}
	if top+height > virtualTop+virtualHeight {
		top = anchorTop - height
	}
	if top < virtualTop {
		top = virtualTop
	}
	if !win.SetWindowPos(window.Handle(), win.HWND_TOPMOST, int32(left), int32(top), int32(width), int32(height), win.SWP_NOACTIVATE|win.SWP_SHOWWINDOW) {
		candidateDebugLog("presenter.window.show_failed")
		return
	}
	candidateDebugLog("presenter.window.shown candidate_count=%d selected_index=%d dpi=%d", len(snapshot.Candidates), snapshot.Selected, snapshot.DPI)
}
