# port-golang-windows-ime

Standalone Go TSF input-method port based on `typescript-windows-ime/native/golang/strategy-generated-bindings`.

## Current implementation

- The Go TSF processor owns pinyin composition and candidate selection state per service instance.
- The pinyin dictionary is embedded into the service; key decisions are reduced in-process, with no named-pipe round trip to any host.
- `OnTestKeyDown` predicts consumption without mutating state; `OnKeyDown` performs the state transition and applies composition/commit through TSF edit sessions.
- Candidate selection supports arrow keys, `1`–`9`, Space and Enter.
- A per-service candidate model (`tip/candidate_ui.go`) tracks composition, candidates, selection, caret geometry and DPI, and hides on commit, cancel, deactivation and focus loss.
- An in-process Walk presenter (`tip/candidate_presenter.go`) now creates a borderless, topmost, non-activating candidate popup on a dedicated locked OS thread. TSF callbacks publish copied snapshots asynchronously; the popup displays numbered candidates and a selected-item marker.

## Known gaps

- The inherited TSF adapter still uses cgo C thunks for some Windows COM calls; this is not yet a zero-C/cgo implementation. Replacing those thunks with reviewed generated Go vtables is a separate ABI-sensitive migration.
- The candidate popup is implemented but still needs interactive validation in real TSF hosts. The presenter currently uses a Walk label rather than a native TSF `ITfCandidateListUIElement`, and candidate placement/DPI behavior must be verified on multiple monitors and scaling settings.
- Dictionary conversion is not yet equivalent to the TypeScript OpenCC `twp` → `cn` conversion, and phrase-combination ranking is currently simplified to exact-code and prefix candidates.
- Real TSF activation, key handling, composition, and commit must be validated in a TSF-enabled Windows application before treating this as production-ready.

## Build/test

Run `go test ./...` from this directory. The root module uses the vendored Walk source at `third_party/walk`; the TIP itself needs the COM linker flags that `install-dll.ps1` sets:

```
go build -buildmode=c-shared -o build\PortGoWindowsIME.dll ./tip   # CGO_LDFLAGS=-lole32 -luuid -loleaut32
```

Do not register or install this experimental TIP on a production profile until the ABI and real-host acceptance checks pass.

Candidate rendering is in-process in the TSF DLL. The standalone `cmd/ime-gui` is a developer control console only; it is not a candidate renderer and must not be launched by the TSF service. The stale `install\ime-gui.exe` from the former pipe-based path has been removed.