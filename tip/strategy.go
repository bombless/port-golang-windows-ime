package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The dictionary is shipped with the TSF DLL. No Node process or external host
// is needed to generate pinyin candidates.
//
//go:embed pinyin_simp.dict.yaml
var dictionaryFS embed.FS

type imeState struct {
	composition string
	selected    int
}

type imeCandidate struct {
	Text       string `json:"text"`
	Index      int    `json:"index"`
	Annotation string `json:"annotation,omitempty"`
}

type imeReduction struct {
	state   imeState
	consume bool
	commit  *string
}

type dictionaryEntry struct {
	text   string
	code   string
	weight float64
}

var pinyinDictionary = loadPinyinDictionary()
var pinyinCodes = func() []string {
	codes := make([]string, 0, len(pinyinDictionary))
	for code := range pinyinDictionary {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}()

func normalizePinyin(s string) string {
	s = strings.ToLower(s)
	return strings.NewReplacer(" ", "", "\t", "", "'", "", "’", "", "`", "", "-", "").Replace(s)
}

func loadPinyinDictionary() map[string][]dictionaryEntry {
	result := make(map[string][]dictionaryEntry)
	file, err := dictionaryFS.Open("pinyin_simp.dict.yaml")
	if err != nil {
		return result
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	inTable := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "..." {
			inTable = true
			continue
		}
		if !inTable || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		columns := strings.Split(line, "\t")
		if len(columns) < 2 {
			continue
		}
		text, code := strings.TrimSpace(columns[0]), normalizePinyin(strings.TrimSpace(columns[1]))
		valid := text != "" && code != ""
		for _, r := range code {
			if r < 'a' || r > 'z' {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		var weight float64
		if len(columns) > 2 && !strings.HasSuffix(strings.TrimSpace(columns[2]), "%") {
			weight, _ = strconv.ParseFloat(strings.TrimSpace(columns[2]), 64)
		}
		result[code] = append(result[code], dictionaryEntry{text: text, code: code, weight: weight})
	}
	for code := range result {
		sort.SliceStable(result[code], func(i, j int) bool { return result[code][i].weight > result[code][j].weight })
	}
	return result
}

func candidatesFor(composition string) []imeCandidate {
	query := normalizePinyin(composition)
	if query == "" {
		return nil
	}
	entries := pinyinDictionary[query]
	seen := map[string]bool{}
	var out []imeCandidate
	for _, e := range entries {
		if seen[e.text] {
			continue
		}
		seen[e.text] = true
		out = append(out, imeCandidate{Text: e.text, Index: len(out), Annotation: e.code})
		if len(out) == 9 {
			return out
		}
	}
	if len(out) > 0 {
		return out
	}
	// Segment multi-syllable pinyin into dictionary entries, ranking phrase
	// combinations by the same additive word-weight heuristic as the TS engine.
	type partial struct {
		text   string
		weight float64
	}
	paths := make([][]partial, len(query)+1)
	paths[0] = []partial{{text: "", weight: 0}}
	for offset := 0; offset < len(query); offset++ {
		if len(paths[offset]) == 0 {
			continue
		}
		for _, code := range pinyinCodes {
			words := pinyinDictionary[code]
			if !strings.HasPrefix(query[offset:], code) {
				continue
			}
			next := offset + len(code)
			top := len(words)
			if top > 4 {
				top = 4
			}
			for _, prefix := range paths[offset] {
				for _, word := range words[:top] {
					paths[next] = append(paths[next], partial{text: prefix.text + word.text, weight: prefix.weight + word.weight - 100})
				}
			}
			sort.SliceStable(paths[next], func(i, j int) bool { return paths[next][i].weight > paths[next][j].weight })
			if len(paths[next]) > 24 {
				paths[next] = paths[next][:24]
			}
		}
	}
	for _, path := range paths[len(query)] {
		if seen[path.text] {
			continue
		}
		seen[path.text] = true
		out = append(out, imeCandidate{Text: path.text, Index: len(out)})
		if len(out) == 9 {
			return out
		}
	}
	if len(out) > 0 {
		return out
	}
	// Prefix fallback gives useful candidates while the user is still typing.
	var codes []string
	for code := range pinyinDictionary {
		if strings.HasPrefix(code, query) {
			codes = append(codes, code)
		}
	}
	sort.Slice(codes, func(i, j int) bool {
		a, b := pinyinDictionary[codes[i]], pinyinDictionary[codes[j]]
		if len(a) == 0 {
			return false
		}
		if len(b) == 0 {
			return true
		}
		return a[0].weight > b[0].weight
	})
	for _, code := range codes {
		for _, e := range pinyinDictionary[code] {
			if seen[e.text] {
				continue
			}
			seen[e.text] = true
			out = append(out, imeCandidate{Text: e.text, Index: len(out), Annotation: e.code})
			if len(out) == 9 {
				return out
			}
		}
	}
	return out
}

func reduceIME(state imeState, key string) imeReduction {
	candidates := candidatesFor(state.composition)
	if len(key) == 1 && ((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= 'a' && key[0] <= 'z')) {
		return imeReduction{state: imeState{composition: state.composition + strings.ToLower(key)}, consume: true}
	}
	switch key {
	case "Backspace":
		if state.composition == "" {
			return imeReduction{state: state}
		}
		return imeReduction{state: imeState{composition: state.composition[:len(state.composition)-1]}, consume: true}
	case "Delete":
		if state.composition == "" {
			return imeReduction{state: state}
		}
		return imeReduction{state: imeState{composition: state.composition[1:]}, consume: true}
	case "Escape":
		if state.composition == "" {
			return imeReduction{state: state}
		}
		return imeReduction{state: imeState{}, consume: true}
	case "ArrowUp", "ArrowLeft", "ArrowDown", "ArrowRight":
		if state.composition == "" || len(candidates) == 0 {
			return imeReduction{state: state}
		}
		delta := 1
		if key == "ArrowUp" || key == "ArrowLeft" {
			delta = -1
		}
		selected := state.selected + delta
		if selected < 0 {
			selected = 0
		}
		if selected >= len(candidates) {
			selected = len(candidates) - 1
		}
		state.selected = selected
		return imeReduction{state: state, consume: true}
	case " ", "Spacebar", "Enter":
		if state.composition == "" {
			return imeReduction{state: state}
		}
		text := state.composition
		if len(candidates) > 0 {
			i := state.selected
			if i < 0 || i >= len(candidates) {
				i = 0
			}
			text = candidates[i].Text
		}
		return imeReduction{state: imeState{}, consume: true, commit: &text}
	}
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' && state.composition != "" {
		i := int(key[0] - '1')
		if i >= len(candidates) {
			return imeReduction{state: state}
		}
		text := candidates[i].Text
		return imeReduction{state: imeState{}, consume: true, commit: &text}
	}
	return imeReduction{state: state}
}

func keyForVK(vk uintptr) string {
	if vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	if vk >= '0' && vk <= '9' {
		return string(rune(vk))
	}
	switch vk {
	case 0x08:
		return "Backspace"
	case 0x2e:
		return "Delete"
	case 0x1b:
		return "Escape"
	case 0x26:
		return "ArrowUp"
	case 0x25:
		return "ArrowLeft"
	case 0x28:
		return "ArrowDown"
	case 0x27:
		return "ArrowRight"
	case 0x20:
		return " "
	case 0x0d:
		return "Enter"
	}
	return ""
}

func (t *tipImpl) runLocalStrategy(kind string, vk uintptr) (string, bool) {
	key := keyForVK(vk)
	if key == "" {
		// Nothing the reducer can do with this key, so it stays with the
		// application. Claiming it in the test phase and then declining to eat
		// it in OnKeyDown would leave the two phases disagreeing about every
		// unmapped key in the application.
		return "", false
	}
	state := imeState{composition: t.imeComposition, selected: t.imeSelected}
	r := reduceIME(state, key)
	if kind == "testKeyDown" {
		return "", r.consume
	}
	if kind != "keyDown" {
		return "", false
	}
	t.imeComposition, t.imeSelected = r.state.composition, r.state.selected
	candidates := candidatesFor(r.state.composition)
	if r.commit != nil {
		candidates = candidatesFor(state.composition)
	}
	response := struct {
		Consume     bool           `json:"consume"`
		Composition string         `json:"composition"`
		Candidates  []imeCandidate `json:"candidates"`
		Selected    int            `json:"selectedCandidate"`
		Commit      *string        `json:"commit,omitempty"`
	}{r.consume, r.state.composition, candidates, r.state.selected, r.commit}
	b, err := json.Marshal(response)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func strategySummary() string {
	return fmt.Sprintf("embedded pinyin entries=%d", len(pinyinDictionary))
}
