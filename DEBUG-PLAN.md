# Candidate Window Debug Plan

## Goal

Find why the Walk-based candidate window never appears. Do not assume the GUI is running just because the project builds. Add diagnostic logging first, reproduce once, and use the logs to identify the first broken link in the pipeline.

## Rules for the debugging session

- First inspect the current implementation and working-tree diff; preserve unrelated user changes.
- Logs across the TSF DLL and `ime-gui.exe` should include local timestamp with milliseconds, PID, component, stage, and result/error. Avoid logging raw user-entered text; candidate count and selected index are enough.
- Log successful milestones and failures. Include Windows error codes / HRESULTs where available.
- Write logs to a stable writable location, preferably `%LOCALAPPDATA%\PortGoWindowsIME\candidate-debug.log`. Create the directory and report initialization failures. Use process-specific files if concurrent writes are unsafe. Do not rely only on stdout/stderr because TSF is hosted by another process.
- Logging must be best-effort and must not crash a TSF callback or GUI. Flush important startup/error messages.
- Do not run install/uninstall or change system registration until code-level diagnosis and a build are ready, unless explicitly requested.

## Phase 0 — Establish what is deployed

1. Inspect `install.ps1`, `uninstall.ps1`, `tip/candidate_presenter.go`, `tip/candidate_ui.go`, `cmd/ime-gui/main.go`, and `cmd/ime-gui/candidate_window.go`.
2. Record the working-tree diff and current build/test status.
3. Verify the actual registered TSF DLL path and whether `ime-gui.exe` exists beside that exact DLL. The build output directory may not be the installed runtime directory.
4. Check whether the running input method loads the newly built DLL or an older versioned DLL. Build success alone does not prove deployment.

## Phase 1 — Instrument the TSF/candidate event path

Add logs for:

1. Candidate UI/presenter initialization and teardown.
2. Every candidate update entry: trigger reason, candidate count, selected index, and whether the list is empty. Do not log candidate contents.
3. Candidate selection and hide/clear events.
4. State snapshot/update completion and each early return, with its reason.
5. Presenter send/update entry and exit, including pipe/process state and returned errors.

Questions: Is the candidate callback invoked? Is the list non-empty? Does execution reach the presenter? Is an update intentionally suppressed?

## Phase 2 — Instrument process launch

At the `ime-gui.exe` launch boundary, log:

- Resolved executable path, existence check, and working directory.
- Launch arguments and strategy, omitting sensitive input text.
- Process-start success/error, PID if available, and whether the child exits immediately.
- Failure resolving the registered DLL directory or locating the GUI executable.
- Whether an existing process is reused or another instance is started.

Do not silently fall back when the executable path cannot be resolved. Record a specific error.

## Phase 3 — Instrument named-pipe IPC

At both sender and receiver, log:

- Pipe name and connect/listen start, success, timeout, and failure.
- Pipe create/open return values and Windows error codes.
- Message write/read start and completion, byte count, framing/version, decode errors, EOF, and disconnect.
- Candidate count and selected index after decoding; do not dump candidate strings.
- Reconnect/retry decisions and shutdown paths.

Verify both ends use the same pipe name, message framing, and schema. Use bounded timeouts; avoid blocking a TSF callback indefinitely.

## Phase 4 — Instrument GUI and window lifecycle

In `cmd/ime-gui/main.go` and `cmd/ime-gui/candidate_window.go`, log:

1. Process startup, command-line mode selection, and initialization failures.
2. Candidate-window mode entry versus normal desktop-console mode.
3. Window creation and native API return values / `GetLastError` where applicable.
4. HWND, initial size, DPI, monitor/work area, target position, and calculated rectangle.
5. Each received update and resulting UI state (candidate count, selected index).
6. `ShowWindow`, `SetWindowPos`, visibility/topmost/non-activating style operations and results.
7. Hide/close/message-loop entry and exit, plus panic/recover or fatal error paths where appropriate.

Check common invisible-window causes: window never created, update loop never entered, empty list causing immediate hide, off-screen coordinates, zero size, owner/parent lifetime, style flags, and GUI process exiting after initialization.

## Phase 5 — Reproduce and classify

1. Build and run unit tests without registering/installing anything yet.
2. If feasible, test GUI candidate mode with a synthetic candidate message to separate window creation from TSF/IPC.
3. After a controlled deployment/restart, start a fresh log and type a short sequence expected to produce multiple candidates.
4. Collect logs from the TSF-host process and GUI. If processes run under different users/sessions, verify log path and pipe security/session assumptions.
5. Find the **first missing stage** and fix only that boundary before repeating.

### Decision table

| Evidence | Likely area | Next check |
|---|---|---|
| No TSF/presenter startup record | Stale/wrong DLL or logging setup | Confirm registered path and loaded module; check log initialization |
| Startup appears, no candidate update | Candidate event/state integration | Trace the actual composition/candidate callback and early returns |
| Candidate update appears, no send attempt | Presenter wiring/branch condition | Follow update call chain and inspect suppression conditions |
| Launch attempt fails | Path, permissions, process creation | Use exact path and Windows error code; verify GUI is beside registered DLL |
| GUI starts, no pipe connection | Pipe name/security/framing/server-client roles | Compare both ends and record OS error/timeout |
| Pipe receives data, no window creation | GUI mode dispatch or message parsing | Check CLI mode, decode errors, early exit/panic |
| Window created, no show record | GUI update/render path | Trace empty-list handling and show/hide conditions |
| Show succeeds, window still invisible | Geometry, styles, z-order, desktop/session | Log HWND, rect, monitor bounds, styles, visibility, topmost result |
| Window appears then disappears | Clear/hide event or process lifetime | Correlate empty updates, hide calls, and process exit |

## Phase 6 — Verification criteria

- Both TSF-host and GUI logs exist and include timestamps/PIDs.
- One reproduction can be traced from candidate update through send, process/pipe, decode, window creation, and show/hide.
- Errors include actionable Windows error codes or HRESULTs.
- `gofmt`, `go test ./...`, and GUI/TSF builds pass where supported. If race tests fail in existing unsafe COM callback code, report that separately rather than attributing it to the candidate window.
- Reduce noisy per-keystroke logs after identifying the root cause, while retaining useful error diagnostics.

## Suggested event names

Use stable labels such as `presenter.init`, `candidate.update.begin`, `candidate.update.end`, `presenter.send.begin`, `gui.launch.begin`, `gui.launch.error`, `pipe.connect.begin`, `pipe.connect.ok`, `pipe.read.error`, `gui.start`, `gui.mode.candidate`, `window.create.ok`, `window.show`, `window.hide`, and `gui.exit`.

## Handoff checklist for the next context

- Read this file first.
- Inspect current source and `git diff`; do not assume logging described here has already been implemented.
- Implement logging in small stages and compile after each boundary.
- Ask the user to reproduce only after a diagnostic build is ready; inspect logs before making speculative fixes.
- Report the first confirmed broken boundary, evidence, and smallest next fix.