package main

import "sync"

// candidateUISnapshot is the immutable view state consumed by an in-process
// candidate presenter. It deliberately contains no COM pointers: presentation
// code must not extend the lifetime of a TSF context or composition.
type candidateUISnapshot struct {
	Composition string
	Candidates  []imeCandidate
	Selected    int
	Caret       [4]int32
	DPI         uint32
	Visible     bool
}

// candidateUIModel is owned by one tipImpl. Its lock permits future UI-thread
// marshaling without making TSF callbacks share mutable slices with a painter.
type candidateUIModel struct {
	mu        sync.RWMutex
	state     candidateUISnapshot
	presenter candidatePresenter
}

func (m *candidateUIModel) update(composition string, candidates []imeCandidate, selected int, caret [4]int32, dpi uint32) {
	candidateDebugLog("candidate.update.begin component=tip stage=model.update candidate_count=%d selected_index=%d composition_empty=%v", len(candidates), selected, composition == "")
	copyCandidates := append([]imeCandidate(nil), candidates...)
	if selected < 0 || selected >= len(copyCandidates) {
		candidateDebugLog("candidate.update.normalize component=tip stage=model.update reason=selected_index_out_of_range old_index=%d candidate_count=%d", selected, len(copyCandidates))
		selected = 0
	}
	visible := composition != "" && len(copyCandidates) != 0
	if !visible {
		copyCandidates = nil
		selected = 0
	}
	if dpi == 0 {
		dpi = 96
	}
	m.mu.Lock()
	m.state = candidateUISnapshot{Composition: composition, Candidates: copyCandidates, Selected: selected, Caret: caret, DPI: dpi, Visible: visible}
	snapshot := m.state
	m.mu.Unlock()
	m.presenter.publish(snapshot)
}

func (m *candidateUIModel) hide() {
	m.mu.Lock()
	m.state = candidateUISnapshot{DPI: 96}
	snapshot := m.state
	m.mu.Unlock()
	candidateDebugLog("candidate.hide component=tip stage=model.hide")
	m.presenter.publish(snapshot)
}

func (m *candidateUIModel) suspend() {
	m.mu.Lock()
	m.state.Visible = false
	snapshot := m.state
	m.mu.Unlock()
	m.presenter.publish(snapshot)
}

func (m *candidateUIModel) resume() {
	m.mu.Lock()
	m.state.Visible = m.state.Composition != "" && len(m.state.Candidates) != 0
	snapshot := m.state
	m.mu.Unlock()
	m.presenter.publish(snapshot)
}

func (m *candidateUIModel) snapshot() candidateUISnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.state
	s.Candidates = append([]imeCandidate(nil), s.Candidates...)
	return s
}
