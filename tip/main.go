// In-process text service for the strategy-generated-bindings input method.
// Its CLSID and profile GUID are not the C++ or Go TIP identities.
package main

/*
#include <stdint.h>
#cgo LDFLAGS: -lole32 -luuid
#define DllGetClassObject WindowsSdkDllGetClassObject
#define DllCanUnloadNow WindowsSdkDllCanUnloadNow
#include <windows.h>
#include <msctf.h>
#include <objbase.h>
#include <string.h>
#undef DllGetClassObject
#undef DllCanUnloadNow

static const CLSID kClsid = {0xc2d4a9b1, 0x7f63, 0x4a85, {0x9e, 0x20, 0x4b, 0x6d, 0x8c, 0x1f, 0x30, 0x72}};
static const GUID kProfile = {0xe731b5d4, 0x2a90, 0x4f68, {0xac, 0x13, 0x7d, 0x5e, 0x9b, 0x20, 0x46, 0xf1}};
static const wchar_t kName[] = L"Go Chinese Input Method";

static const GUID kIidTextInputProcessorEx = {0x6e4e2102, 0xf9cd, 0x433d, {0xb4, 0x96, 0x30, 0x3c, 0xe0, 0x3a, 0x65, 0x07}};
static const GUID kCatTipKeyboard = {0x34745c63, 0xb2f0, 0x4784, {0x8b, 0x67, 0x5e, 0x12, 0xc8, 0x70, 0x1a, 0x31}};
static const GUID kCatTipComless = {0x364215d9, 0x75bc, 0x11d7, {0xa6, 0xef, 0x00, 0x06, 0x5b, 0x84, 0x43, 0x5c}};
static const GUID kCatTipImmersive = {0x13a016df, 0x560b, 0x46cd, {0x94, 0x7a, 0x4c, 0x3a, 0xf1, 0xe0, 0xe3, 0x5d}};
static const GUID kCatTipSecure = {0x49d2f9ce, 0x1f5e, 0x11d7, {0xa6, 0xd3, 0x00, 0x06, 0x5b, 0x84, 0x43, 0x5c}};
static const GUID kCatTipSystray = {0x25504fb4, 0x7bab, 0x4bc1, {0x9c, 0x69, 0xcf, 0x81, 0x89, 0x0f, 0x0e, 0xf5}};
static const GUID kCatTipUIElement = {0x49d2f9cf, 0x1f5e, 0x11d7, {0xa6, 0xd3, 0x00, 0x06, 0x5b, 0x84, 0x43, 0x5c}};
static void getTextInputProcessorExIID(void* out) { memcpy(out, &kIidTextInputProcessorEx, sizeof(GUID)); }

static HRESULT adviseKeySink(void* threadMgr, DWORD clientId, void* sink) {
    ITfKeystrokeMgr* keys = NULL;
    HRESULT hr = ((IUnknown*)threadMgr)->lpVtbl->QueryInterface((IUnknown*)threadMgr, &IID_ITfKeystrokeMgr, (void**)&keys);
    if (FAILED(hr)) return hr;
    ITfKeyEventSink* checkedSink = NULL; hr = ((IUnknown*)sink)->lpVtbl->QueryInterface((IUnknown*)sink, &IID_ITfKeyEventSink, (void**)&checkedSink); if (FAILED(hr)) { keys->lpVtbl->Release(keys); return hr; } checkedSink->lpVtbl->Release(checkedSink); hr = keys->lpVtbl->AdviseKeyEventSink(keys, clientId, (ITfKeyEventSink*)sink, TRUE);
    keys->lpVtbl->Release(keys);
    return hr;
}

static HRESULT unadviseKeySink(void* threadMgr, DWORD clientId) {
    ITfKeystrokeMgr* keys = NULL;
    HRESULT hr = ((IUnknown*)threadMgr)->lpVtbl->QueryInterface((IUnknown*)threadMgr, &IID_ITfKeystrokeMgr, (void**)&keys);
    if (FAILED(hr)) return hr;
    hr = keys->lpVtbl->UnadviseKeyEventSink(keys, clientId);
    keys->lpVtbl->Release(keys);
    return hr;
}
static const wchar_t kClsidKey[] = L"CLSID\\{C2D4A9B1-7F63-4A85-9E20-4B6D8C1F3072}";
static const wchar_t kInprocKey[] = L"CLSID\\{C2D4A9B1-7F63-4A85-9E20-4B6D8C1F3072}\\InprocServer32";

static HRESULT writeString(HKEY root, const wchar_t* subKey, const wchar_t* valueName, const wchar_t* value) {
    HKEY key = NULL;
    LONG result = RegCreateKeyExW(root, subKey, 0, NULL, REG_OPTION_NON_VOLATILE, KEY_SET_VALUE, NULL, &key, NULL);
    if (result != ERROR_SUCCESS) return HRESULT_FROM_WIN32(result);
    DWORD bytes = (DWORD)((wcslen(value) + 1) * sizeof(wchar_t));
    result = RegSetValueExW(key, valueName, 0, REG_SZ, (const BYTE*)value, bytes);
    RegCloseKey(key);
    return HRESULT_FROM_WIN32(result);
}

static HRESULT registerComServer(void) {
    HMODULE module = NULL;
    wchar_t path[MAX_PATH];
    DWORD length;
    HRESULT hr;
    if (!GetModuleHandleExW(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS | GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
            (LPCWSTR)(uintptr_t)registerComServer, &module)) {
        return HRESULT_FROM_WIN32(GetLastError());
    }
    length = GetModuleFileNameW(module, path, ARRAYSIZE(path));
    if (length == 0 || length >= ARRAYSIZE(path)) return HRESULT_FROM_WIN32(GetLastError());
    hr = writeString(HKEY_CLASSES_ROOT, kClsidKey, NULL, kName);
    if (FAILED(hr)) return hr;
    hr = writeString(HKEY_CLASSES_ROOT, kInprocKey, NULL, path);
    if (FAILED(hr)) return hr;
    return writeString(HKEY_CLASSES_ROOT, kInprocKey, L"ThreadingModel", L"Apartment");
}

static HRESULT unregisterComServer(void) {
    LSTATUS result = RegDeleteTreeW(HKEY_CLASSES_ROOT, kClsidKey);
    if (result == ERROR_FILE_NOT_FOUND) return S_OK;
    return HRESULT_FROM_WIN32(result);
}

static HRESULT registerTip(void) {
    ITfInputProcessorProfiles* profiles = NULL;
    ITfCategoryMgr* categories = NULL;
    const LANGID language = 0x0804; // Simplified Chinese (zh-CN) only.
    HRESULT hr = CoCreateInstance(&CLSID_TF_InputProcessorProfiles, NULL, CLSCTX_INPROC_SERVER,
        &IID_ITfInputProcessorProfiles, (void**)&profiles);
    if (FAILED(hr)) return hr;
    hr = profiles->lpVtbl->Register(profiles, &kClsid);
    if (SUCCEEDED(hr)) {
        // Remove legacy profiles from previous builds so they cannot remain
        // visible under English or Japanese after upgrading.
        profiles->lpVtbl->RemoveLanguageProfile(profiles, &kClsid, 0x0409, &kProfile);
        profiles->lpVtbl->RemoveLanguageProfile(profiles, &kClsid, 0x0411, &kProfile);
        hr = profiles->lpVtbl->AddLanguageProfile(profiles, &kClsid, language, &kProfile,
            kName, (ULONG)(wcslen(kName)), NULL, 0, 0);
    }
    if (SUCCEEDED(hr)) {
        hr = CoCreateInstance(&CLSID_TF_CategoryMgr, NULL, CLSCTX_INPROC_SERVER,
            &IID_ITfCategoryMgr, (void**)&categories);
        if (SUCCEEDED(hr)) {
            const GUID* tipCategories[] = {
                &kCatTipKeyboard,
                &kCatTipComless,
                &kCatTipImmersive,
                &kCatTipSecure,
                &kCatTipSystray,
                &kCatTipUIElement,
            };
            for (size_t i = 0; i < sizeof(tipCategories) / sizeof(tipCategories[0]); ++i) {
                HRESULT categoryHr = categories->lpVtbl->RegisterCategory(categories, &kClsid, tipCategories[i], &kClsid);
                if (SUCCEEDED(hr) && FAILED(categoryHr)) hr = categoryHr;
            }
            categories->lpVtbl->Release(categories);
        }
    }
    profiles->lpVtbl->Release(profiles);
    return hr;
}

static HRESULT unregisterTip(void) {
    ITfCategoryMgr* categories = NULL;
    HRESULT categoryHr = CoCreateInstance(&CLSID_TF_CategoryMgr, NULL, CLSCTX_INPROC_SERVER,
        &IID_ITfCategoryMgr, (void**)&categories);
    if (SUCCEEDED(categoryHr)) {
        const GUID* tipCategories[] = {
            &kCatTipKeyboard,
            &kCatTipComless,
            &kCatTipImmersive,
            &kCatTipSecure,
            &kCatTipSystray,
            &kCatTipUIElement,
        };
        for (size_t i = 0; i < sizeof(tipCategories) / sizeof(tipCategories[0]); ++i)
            categories->lpVtbl->UnregisterCategory(categories, &kClsid, tipCategories[i], &kClsid);
        categories->lpVtbl->Release(categories);
    }
    ITfInputProcessorProfiles* profiles = NULL;
    HRESULT hr = CoCreateInstance(&CLSID_TF_InputProcessorProfiles, NULL, CLSCTX_INPROC_SERVER,
        &IID_ITfInputProcessorProfiles, (void**)&profiles);
    if (FAILED(hr)) return hr;
    hr = profiles->lpVtbl->Unregister(profiles, &kClsid);
    profiles->lpVtbl->Release(profiles);
    return hr;
}

static HRESULT runRegistration(BOOL unregister) {
    HRESULT init = CoInitializeEx(NULL, COINIT_APARTMENTTHREADED);
    HRESULT hr;
    if (FAILED(init) && init != RPC_E_CHANGED_MODE) return init;
    if (unregister) {
        hr = unregisterTip();
        if (SUCCEEDED(hr)) hr = unregisterComServer();
    } else {
        hr = registerComServer();
        if (SUCCEEDED(hr)) hr = registerTip();
        if (FAILED(hr)) unregisterComServer();
    }
    if (SUCCEEDED(init)) CoUninitialize();
    return hr;
}
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/zzl/go-com/com"
	"github.com/zzl/go-com/com/comimpl"
	"github.com/zzl/go-win32api/v2/win32"
	"port-golang-windows-ime/closedtsf"
)

// GetKeyState is a user32 export, not a kernel32 one. Looking it up in the
// wrong DLL panics on the first key event, which a guarded callback turns into a
// silent E_FAIL for every keystroke.
var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	user32                 = syscall.NewLazyDLL("user32.dll")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
	procGetKeyState        = user32.NewProc("GetKeyState")
)

func threadID() uintptr {
	id, _, _ := procGetCurrentThreadId.Call()
	return id
}

// Everything the TIP has to say lands in one shared file so a fault inside a
// host application can be reconstructed after the fact. The Host writes to the
// same path by default, which means one file interleaves both sides of the
// conversation. TypeScriptWindowsIMELogPath overrides it, which is how a test
// or a second TIP instance is kept out of the shared file.
const strategyLogPath = "TypeScriptWindowsIME.log"

const strategyLogPathEnvVar = "TypeScriptWindowsIMELogPath"

func strategyLogPathFull() string {
	if override := os.Getenv(strategyLogPathEnvVar); override != "" {
		return override
	}
	return filepath.Join(os.TempDir(), strategyLogPath)
}

func strategyLog(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s][pid=%d tid=%d] %s\r\n",
		time.Now().Format("2006-01-02 15:04:05.000"),
		os.Getpid(), threadID(), message)
	path, err := syscall.UTF16PtrFromString(strategyLogPathFull())
	if err != nil {
		return
	}
	file, err := syscall.CreateFile(path, syscall.FILE_APPEND_DATA,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return
	}
	var written uint32
	_ = syscall.WriteFile(file, []byte(line), &written, nil)
	_ = syscall.CloseHandle(file)
}

// Every failure here has to be survivable. This DLL runs inside whatever
// application the user typed into, so a panic that unwinds through a native
// frame takes that process down with it. Callbacks therefore recover, dump every
// goroutine, and answer E_FAIL, which TSF reports as a plain activation or key
// failure instead of a crash.
const strategyEFail int32 = -2147467259 // 0x80004005

// hresultToUintptr converts a signed HRESULT into the unsigned 32-bit value COM
// returns in the register. It has to be a function: Go rejects converting a
// negative constant straight to uintptr at compile time.
func hresultToUintptr(hr int32) uintptr {
	return uintptr(uint32(hr))
}

// writeStackDump appends every goroutine, not just the failing one. The blocked
// pipe reader and a callback waiting on a dead lock are only visible in a full
// dump, and this DLL cannot be inspected with a source-level debugger once it
// is loaded into an application.
func writeStackDump(reason string) {
	size := 1 << 18
	for {
		buffer := make([]byte, size)
		written := runtime.Stack(buffer, true)
		if written < len(buffer) {
			strategyLog("goroutine dump (%s):\r\n%s", reason, string(buffer[:written]))
			return
		}
		size *= 2
		if size > 32<<20 {
			strategyLog("goroutine dump (%s): truncated at %d bytes", reason, size)
			return
		}
	}
}

// guardedCall runs one COM callback and keeps a fault inside it. The deferred
// recover belongs to this frame, so it still fires after run has unwound, and
// the HRESULT written to result is what the caller receives. A recovered panic
// must never escape past here: the frames underneath belong to the OS.
func guardedCall(name string, run func() uintptr) (result uintptr) {
	result = hresultToUintptr(strategyEFail)
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		strategyLog("PANIC %s: %v", name, recovered)
		writeStackDump("panic in " + name)
		result = hresultToUintptr(strategyEFail)
	}()
	return run()
}

// dumpWatchPath is polled so a stuck TIP inside an application that cannot be
// restarted can still be inspected: create this file and the next poll appends
// every goroutine stack to the log. Deleting it afterwards arms the next dump.
func dumpWatchPath() string {
	return filepath.Join(os.TempDir(), "TypeScriptWindowsIME.dump")
}

func watchForDumpRequests() {
	path := dumpWatchPath()
	for {
		if _, err := os.Stat(path); err == nil {
			writeStackDump("dump requested by " + path)
			_ = os.Remove(path)
		}
		time.Sleep(time.Second)
	}
}

// strategyPauseAt freezes the calling thread at a named point so a dump can be
// taken while the object is still alive. It is off unless the variable is set
// to that exact point name, because a blocked TSF callback hangs the host
// application and that is only ever wanted deliberately.
func strategyPauseAt(point string) {
	wanted := os.Getenv("TypeScriptWindowsIMEPauseAt")
	if wanted == "" || wanted != point {
		return
	}
	strategyLog("StrategyTip paused at %s pid=%d tid=%d", point, os.Getpid(), threadID())
	for {
		time.Sleep(time.Hour)
	}
}

func init() {
	C.getTextInputProcessorExIID(unsafe.Pointer(&iidTextInputProcessorEx))
	// Default Go traceback behaviour prints to stderr, which a GUI application
	// never shows, and only the crashing goroutine. Both sinks are useless here.
	if os.Getenv("GOTRACEBACK") == "" {
		_ = os.Setenv("GOTRACEBACK", "all")
	}
	debug.SetTraceback("all")
	strategyLog("StrategyTip DLL loaded pid=%d tid=%d GOTRACEBACK=%s dumpWatch=%s",
		os.Getpid(), threadID(), os.Getenv("GOTRACEBACK"), dumpWatchPath())
	go watchForDumpRequests()
}

// replyFieldValue locates a numeric or boolean protocol field. Only an
// occurrence that starts a field — right after { or , — counts, so a
// composition or candidate string that happens to contain the field text cannot
// be mistaken for one. The remainder is scanned as text rather than parsed
// because these lines are produced by this DLL's own requests.
func replyFieldValue(line, field string) (string, bool) {
	for offset := 0; ; {
		index := strings.Index(line[offset:], field)
		if index < 0 {
			return "", false
		}
		index += offset
		if index == 0 || line[index-1] == ',' || line[index-1] == '{' {
			return line[index+len(field):], true
		}
		offset = index + len(field)
	}
}

// replyConsume reports whether the reducer kept the key. It is the field the
// whole pipeline exists to deliver.
func replyConsume(line string) bool {
	rest, ok := replyFieldValue(line, `"consume":`)
	if !ok {
		return false
	}
	return strings.HasPrefix(rest[skipSpaces(rest, 0):], "true")
}

func skipSpaces(text string, from int) int {
	for from < len(text) && (text[from] == ' ' || text[from] == '\t') {
		from++
	}
	return from
}

// keyDecision is what a key callback writes into pfEaten. Both halves matter: a
// reduction that never produced a reply must leave the key with the application,
// which is the same rule the C++ service applied with *eaten = ok && consume in
// native/TsIme.cpp:971.
func keyDecision(reply string, replied bool) bool {
	return replied && replyConsume(reply)
}

// describeReply renders a reducer reply with the fields that decide behaviour:
// whether the key was eaten, what is being composed, and which candidate is
// selected. The line used to be truncated at 200 bytes, which cut off exactly
// those fields.
func describeReply(text string) string {
	fields := []struct {
		key   string
		label string
	}{
		{`"consume":`, "consume"},
		{`"composition":`, "composition"},
		{`"commit":`, "commit"},
		{`"selectedCandidate":`, "selected"},
	}
	var parts []string
	for _, field := range fields {
		index := strings.Index(text, field.key)
		if index < 0 {
			continue
		}
		rest := text[index+len(field.key):]
		end := strings.IndexAny(rest, ",}")
		if end < 0 {
			end = len(rest)
		}
		parts = append(parts, field.label+"="+rest[:end])
	}
	candidates := 0
	if index := strings.Index(text, `"candidates":[`); index >= 0 {
		candidates = strings.Count(text[index+len(`"candidates":[`):], "{")
	}
	summary := fmt.Sprintf("bytes=%d", len(text))
	if len(parts) > 0 {
		summary += " " + strings.Join(parts, " ")
	}
	summary += fmt.Sprintf(" candidates=%d", candidates)
	return summary + " raw=" + text
}

// {C2D4A9B1-7F63-4A85-9E20-4B6D8C1F3072}
var clsidStrategy = syscall.GUID{Data1: 0xC2D4A9B1, Data2: 0x7F63, Data3: 0x4A85, Data4: [8]byte{0x9E, 0x20, 0x4B, 0x6D, 0x8C, 0x1F, 0x30, 0x72}}

var iidTextInputProcessor = syscall.GUID{Data1: 0xAA80E7F7, Data2: 0x2021, Data3: 0x11D2, Data4: [8]byte{0x93, 0xE0, 0x00, 0x60, 0xB0, 0x67, 0xB8, 0x6E}}
var iidTextInputProcessorEx syscall.GUID

var iidKeyEventSink = closedtsf.IID_ITfKeyEventSink

// guidText names the IIDs that matter and prints the rest, because a bare GUID
// in the log is useless when the question is "which interface did TSF ask for".
func guidText(guid syscall.GUID) string {
	switch guid {
	case clsidStrategy:
		return "CLSID_StrategyTip"
	case iidTextInputProcessor:
		return "IID_ITfTextInputProcessor"
	case iidTextInputProcessorEx:
		return "IID_ITfTextInputProcessorEx"
	case iidKeyEventSink:
		return "IID_ITfKeyEventSink"
	}
	return fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
		guid.Data1, guid.Data2, guid.Data3,
		guid.Data4[0], guid.Data4[1], guid.Data4[2], guid.Data4[3],
		guid.Data4[4], guid.Data4[5], guid.Data4[6], guid.Data4[7])
}

type classFactoryImpl struct {
	comimpl.IClassFactoryImpl
	locks atomic.Int32
}

func (f *classFactoryImpl) CreateInstance(outer *win32.IUnknown, iid *syscall.GUID, out unsafe.Pointer) win32.HRESULT {
	if out == nil {
		strategyLog("CreateInstance E_POINTER: null out")
		return win32.E_POINTER
	}
	*(*unsafe.Pointer)(out) = nil
	if iid == nil {
		strategyLog("CreateInstance E_POINTER: null iid")
		return win32.E_POINTER
	}
	if outer != nil {
		strategyLog("CreateInstance CLASS_E_NOAGGREGATION: TSF asked for aggregation")
		return win32.CLASS_E_NOAGGREGATION
	}
	obj := com.NewComObj[tipComObj](&tipImpl{})
	hr := obj.QueryInterface(iid, out)
	obj.Release()
	// A wrong IID here is the classic silent failure: TSF drops the text
	// service without ever calling Activate, so nothing else would report it.
	strategyLog("CreateInstance iid=%s out=%p hr=0x%08x", guidText(*iid), *(*unsafe.Pointer)(out), uint32(hr))
	return win32.HRESULT(hr)
}

func (f *classFactoryImpl) LockServer(lock win32.BOOL) win32.HRESULT {
	if lock != 0 {
		f.locks.Add(1)
	} else if f.locks.Load() > 0 {
		f.locks.Add(-1)
	}
	strategyLog("LockServer lock=%d serverLocks=%d", int32(lock), f.locks.Load())
	return win32.S_OK
}

type tipImpl struct {
	com.IUnknownImpl
	threadMgr   unsafe.Pointer
	clientID    uint32
	sinkAdvised bool
	// Input-method policy is owned by this TSF instance; no Node/pipe state.
	imeComposition string
	imeSelected    int

	// Key callbacks are serialized: the composition, the candidate selection and
	// the candidate UI model are all single-threaded state, and TSF may well
	// deliver keys from the application's thread while another application types.
	keyMu sync.Mutex
	// Candidate state is private to this TSF service instance. The model is
	// presentation-ready, even when a platform presenter is unavailable.
	candidateUI candidateUIModel
	// *ITfComposition of the composition on screen, plus the context it lives
	// in. 0 means there is none.
	composition           atomic.Uintptr
	compositionContext    atomic.Uintptr
	compositionText       atomic.Value
	compositionTerminated atomic.Bool
	// The text the composition range was last given to write. The range does
	// not grow with it, so the next write has to replace it by hand, and a
	// cancel has to clear it. Empty means a composition writing for the first
	// time, which must still be allowed to overwrite a selection.
	compositionWritten atomic.Value // string
	// When text is committed, a clone of the range collapsed at its end awaits
	// the SetSelection that moves the caret behind that text. It is parked here
	// because the edit session object is released before the key sink, which
	// holds no lock at that point, runs the SetSelection against the context of
	// the OnKeyDown call that parked the clone. See placeSelectionAtRangeEnd.
	pendingCaretRange atomic.Uintptr
}

func (t *tipImpl) Activate(threadMgr *win32.IUnknown, clientID uint32) win32.HRESULT {
	// Reachability here decides whether any further logging exists at all, so
	// it has to come before the first thing that can fail.
	strategyLog("StrategyTip Activate enter threadMgr=%p clientId=%d", threadMgr, clientID)
	strategyPauseAt("Activate")
	if threadMgr == nil {
		strategyLog("StrategyTip Activate E_POINTER: no ITfThreadMgr")
		return win32.E_POINTER
	}
	obj := t.ComObj.(*tipComObj)
	hr := C.adviseKeySink(unsafe.Pointer(threadMgr), C.DWORD(clientID), unsafe.Pointer(&obj.keySinkComObj))
	if hr < 0 {
		strategyLog("StrategyTip AdviseKeyEventSink failed hr=0x%08x", uint32(hr))
		writeStackDump("AdviseKeyEventSink failed")
		return win32.HRESULT(hr)
	}
	t.threadMgr = unsafe.Pointer(threadMgr)
	t.clientID = clientID
	t.sinkAdvised = true
	t.candidateUI.presenter.setEnabled(true)
	strategyLog("StrategyTip key sink advised clientId=%d; %s", clientID, strategySummary())
	return win32.S_OK
}

func (t *tipImpl) Deactivate() win32.HRESULT {
	strategyLog("StrategyTip Deactivate enter sinkAdvised=%v clientId=%d", t.sinkAdvised, t.clientID)
	t.imeComposition, t.imeSelected = "", 0
	t.candidateUI.presenter.setEnabled(false)
	t.candidateUI.hide()
	strategyPauseAt("Deactivate")
	if t.sinkAdvised && t.threadMgr != nil {
		hr := C.unadviseKeySink(t.threadMgr, C.DWORD(t.clientID))
		strategyLog("StrategyTip UnadviseKeyEventSink hr=0x%08x", uint32(hr))
		if hr < 0 {
			writeStackDump("UnadviseKeyEventSink failed")
		}
		t.sinkAdvised = false
		t.threadMgr = nil
	} else {
		// Activation never completed, so there is nothing to unadvise. Saying so
		// keeps a missing UnadviseKeyEventSink from looking like a silent leak.
		strategyLog("StrategyTip Deactivate had no advised sink to retire")
	}
	// A composition on screen belongs to a document that is going away; taking
	// it down and dropping our reference keeps a dead ITfComposition from being
	// used by a later activation. This is the one place the remembered context
	// is used on its own, and it runs after the sink is retired so no key
	// callback can be in flight any more.
	if t.composition.Load() != 0 && t.compositionContext.Load() != 0 && t.clientID != 0 {
		t.endComposition(uintptr(t.compositionContext.Load()), "")
	}
	t.dropComposition()
	t.compositionContext.Store(0)
	t.compositionText.Store("")
	return win32.S_OK
}

type tipVtbl struct {
	win32.IUnknownVtbl
	Activate   uintptr
	Deactivate uintptr
	ActivateEx uintptr
}

type legacyTipVtbl struct {
	win32.IUnknownVtbl
	Activate   uintptr
	Deactivate uintptr
}

type tipComObj struct {
	com.IUnknownComObj
	keySinkComObj   keySinkComObj
	legacyTipComObj legacyTipComObj
}

func (t *tipComObj) GetSubComObjs() []com.ComObjInterface {
	return []com.ComObjInterface{&t.keySinkComObj, &t.legacyTipComObj}
}

type legacyTipComObj struct{ com.IUnknownComObj }

func (t *legacyTipComObj) IID() *syscall.GUID { return &iidTextInputProcessor }
func (t *legacyTipComObj) GetVtbl() *win32.IUnknownVtbl {
	return (*win32.IUnknownVtbl)(unsafe.Pointer(t.BuildLegacyTipVtbl()))
}
func (t *legacyTipComObj) BuildLegacyTipVtbl() *legacyTipVtbl {
	com.MuVtbl.Lock()
	defer com.MuVtbl.Unlock()
	if legacyTipVTable == nil {
		v := (*legacyTipVtbl)(com.Malloc(unsafe.Sizeof(legacyTipVtbl{})))
		v.IUnknownVtbl = *t.IUnknownComObj.BuildVtbl(false)
		v.Activate = syscall.NewCallback(legacyTipActivateCallback)
		v.Deactivate = syscall.NewCallback(legacyTipDeactivateCallback)
		legacyTipVTable = v
	}
	return legacyTipVTable
}

// keyName mirrors the native TSF implementation's key-name mapping.
func keyName(vk uintptr) string {
	switch vk {
	case 0x08:
		return "Backspace"
	case 0x2E:
		return "Delete"
	case 0x1B:
		return "Escape"
	case 0x0D:
		return "Enter"
	case 0x20:
		return " "
	case 0x25:
		return "ArrowLeft"
	case 0x26:
		return "ArrowUp"
	case 0x27:
		return "ArrowRight"
	case 0x28:
		return "ArrowDown"
	default:
		if (vk >= '1' && vk <= '9') || (vk >= 'A' && vk <= 'Z') {
			return string(rune(vk))
		}
		return ""
	}
}

// winHeld reports whether either Windows key is currently down.
func winHeld() bool {
	for _, vk := range []uintptr{0x5B, 0x5C} {
		if state, _, _ := procGetKeyState.Call(vk); int16(state) < 0 {
			return true
		}
	}
	return false
}

// winSpaceBypass keeps Win+Space out of the IME. Windows owns that combination
// to switch input methods, and once keys are really consumed a live composition
// would swallow it. This mirrors the guard at the top of
// TextService::TestKey and TextService::HandleKey in native/TsIme.cpp.
func winSpaceBypass(vk uintptr) bool {
	return vk == 0x20 && winHeld()
}

type keySinkComObj struct{ com.IUnknownComObj }

func (k *keySinkComObj) IID() *syscall.GUID { return &iidKeyEventSink }
func (k *keySinkComObj) GetVtbl() *win32.IUnknownVtbl {
	return (*win32.IUnknownVtbl)(unsafe.Pointer(k.BuildKeySinkVtbl(true)))
}

var keySinkVTable *closedtsf.ITfKeyEventSinkVtbl

func (k *keySinkComObj) BuildKeySinkVtbl(lock bool) *closedtsf.ITfKeyEventSinkVtbl {
	if lock {
		com.MuVtbl.Lock()
		defer com.MuVtbl.Unlock()
	}
	if keySinkVTable == nil {
		keySinkVTable = (*closedtsf.ITfKeyEventSinkVtbl)(com.Malloc(unsafe.Sizeof(closedtsf.ITfKeyEventSinkVtbl{})))
		base := k.IUnknownComObj.BuildVtbl(false)
		keySinkVTable.IUnknownVtbl = closedtsf.IUnknownVtbl{QueryInterface: base.QueryInterface, AddRef: base.AddRef, Release: base.Release}
		keySinkVTable.OnSetFocus = syscall.NewCallback(keySinkOnSetFocus)
		keySinkVTable.OnTestKeyDown = syscall.NewCallback(keySinkOnTestKeyDown)
		keySinkVTable.OnTestKeyUp = syscall.NewCallback(keySinkOnTestKeyUp)
		keySinkVTable.OnKeyDown = syscall.NewCallback(keySinkOnKeyDown)
		keySinkVTable.OnKeyUp = syscall.NewCallback(keySinkOnKeyUp)
		keySinkVTable.OnPreservedKey = syscall.NewCallback(keySinkOnPreservedKey)
	}
	return keySinkVTable
}

var tipVTable *tipVtbl
var legacyTipVTable *legacyTipVtbl

func (t *tipComObj) IID() *syscall.GUID { return &iidTextInputProcessorEx }

func (t *tipComObj) BuildTipVtbl() *tipVtbl {
	com.MuVtbl.Lock()
	defer com.MuVtbl.Unlock()
	if tipVTable == nil {
		v := (*tipVtbl)(com.Malloc(unsafe.Sizeof(tipVtbl{})))
		v.IUnknownVtbl = *t.IUnknownComObj.BuildVtbl(false)
		v.Activate = syscall.NewCallback(tipActivateCallback)
		v.Deactivate = syscall.NewCallback(tipDeactivateCallback)
		v.ActivateEx = syscall.NewCallback(tipActivateExCallback)
		tipVTable = v
	}
	return tipVTable
}

func (t *tipComObj) GetVtbl() *win32.IUnknownVtbl {
	return (*win32.IUnknownVtbl)(unsafe.Pointer(t.BuildTipVtbl()))
}

func (t *tipComObj) activateCallback(_ uintptr, threadMgr uintptr, clientID uintptr) uintptr {
	strategyLog("StrategyTip Activate callback entered threadMgr=0x%x clientId=%d", threadMgr, clientID)
	return uintptr(t.Impl().(*tipImpl).Activate((*win32.IUnknown)(unsafe.Pointer(threadMgr)), uint32(clientID)))
}

func (t *tipComObj) deactivateCallback(_ uintptr) uintptr {
	return uintptr(t.Impl().(*tipImpl).Deactivate())
}

//export DllGetClassObject
func DllGetClassObject(clsid unsafe.Pointer, iid unsafe.Pointer, out unsafe.Pointer) C.int32_t {
	return C.int32_t(guardedCall("DllGetClassObject", func() uintptr {
		if out == nil {
			strategyLog("DllGetClassObject E_POINTER: null out")
			return hresultToUintptr(int32(win32.E_POINTER))
		}
		*(*unsafe.Pointer)(out) = nil
		if clsid == nil || iid == nil {
			strategyLog("DllGetClassObject E_POINTER: null clsid or iid")
			return hresultToUintptr(int32(win32.E_POINTER))
		}
		// TSF asks for a class it does not know as part of normal probing, so
		// this path is expected and only worth a trace, not a dump.
		if *(*syscall.GUID)(clsid) != clsidStrategy {
			strategyLog("DllGetClassObject CLASS_E_CLASSNOTAVAILABLE for %s", guidText(*(*syscall.GUID)(clsid)))
			return hresultToUintptr(int32(win32.CLASS_E_CLASSNOTAVAILABLE))
		}
		factory := com.NewComObj[comimpl.IClassFactoryComObj](&classFactoryImpl{})
		hr := factory.QueryInterface((*syscall.GUID)(iid), out)
		factory.Release()
		strategyLog("DllGetClassObject clsid=StrategyTip iid=%s out=%p hr=0x%08x",
			guidText(*(*syscall.GUID)(iid)), *(*unsafe.Pointer)(out), uint32(hr))
		return uintptr(hr)
	}))
}

//export DllCanUnloadNow
func DllCanUnloadNow() C.int32_t {
	return C.int32_t(guardedCall("DllCanUnloadNow", func() uintptr {
		// Always S_FALSE: the Go runtime owns threads and a package-level
		// registry that outlive the last COM reference, so the image must stay
		// mapped for the life of the host process.
		strategyLog("DllCanUnloadNow -> S_FALSE (module stays mapped for this process)")
		return hresultToUintptr(int32(win32.S_FALSE))
	}))
}

//export DllRegisterServer
func DllRegisterServer() C.int32_t {
	return C.int32_t(guardedCall("DllRegisterServer", func() uintptr {
		hr := int32(C.runRegistration(0))
		strategyLog("DllRegisterServer hr=0x%08x", uint32(hr))
		return hresultToUintptr(hr)
	}))
}

//export DllUnregisterServer
func DllUnregisterServer() C.int32_t {
	return C.int32_t(guardedCall("DllUnregisterServer", func() uintptr {
		hr := int32(C.runRegistration(1))
		strategyLog("DllUnregisterServer hr=0x%08x", uint32(hr))
		return hresultToUintptr(hr)
	}))
}

func main() {}

// Use free functions for syscall callbacks: the native COM this pointer is
// an explicit first ABI argument and must not be confused with a receiver.
// Every one of them is wrapped, because a panic crossing this boundary takes
// the host application down instead of returning a failure HRESULT to TSF.
func tipActivateCallback(this uintptr, threadMgr uintptr, clientID uintptr) uintptr {
	return guardedCall("ITfTextInputProcessor.Activate", func() uintptr {
		obj := (*tipComObj)(unsafe.Pointer(this))
		return uintptr(obj.Impl().(*tipImpl).Activate((*win32.IUnknown)(unsafe.Pointer(threadMgr)), uint32(clientID)))
	})
}

func tipDeactivateCallback(this uintptr) uintptr {
	return guardedCall("ITfTextInputProcessor.Deactivate", func() uintptr {
		return uintptr((*tipComObj)(unsafe.Pointer(this)).Impl().(*tipImpl).Deactivate())
	})
}

func tipActivateExCallback(this uintptr, threadMgr uintptr, clientID uintptr, flags uintptr) uintptr {
	return guardedCall("ITfTextInputProcessorEx.ActivateEx", func() uintptr {
		strategyLog("StrategyTip ActivateEx flags=0x%08x secure=%v uiElementOnly=%v noActivateTip=%v console=%v wow16=%v",
			uint32(flags), flags&0x2 != 0, flags&0x4 != 0, flags&0x1 != 0, flags&0x40 != 0, flags&0x10 != 0)
		obj := (*tipComObj)(unsafe.Pointer(this))
		hr := obj.Impl().(*tipImpl).Activate((*win32.IUnknown)(unsafe.Pointer(threadMgr)), uint32(clientID))
		strategyLog("StrategyTip ActivateEx result hr=0x%08x", uint32(hr))
		return uintptr(hr)
	})
}

func legacyTipActivateCallback(this uintptr, threadMgr uintptr, clientID uintptr) uintptr {
	return guardedCall("ITfTextInputProcessor.Activate", func() uintptr {
		obj := (*legacyTipComObj)(unsafe.Pointer(this))
		return uintptr(obj.Impl().(*tipImpl).Activate((*win32.IUnknown)(unsafe.Pointer(threadMgr)), uint32(clientID)))
	})
}

func legacyTipDeactivateCallback(this uintptr) uintptr {
	return guardedCall("ITfTextInputProcessor.Deactivate", func() uintptr {
		return uintptr((*legacyTipComObj)(unsafe.Pointer(this)).Impl().(*tipImpl).Deactivate())
	})
}

// eat writes the BOOL TSF reads back through pfEaten and logs what was decided,
// so a key the pipeline could not consume shows up as a decision rather than as
// silence.
func eat(callback string, pfEaten uintptr, eaten bool) {
	if pfEaten != 0 {
		value := int32(0)
		if eaten {
			value = 1
		}
		*(*int32)(unsafe.Pointer(pfEaten)) = value
	}
	strategyLog("TSF %s pfEaten=0x%x -> eaten=%v", callback, pfEaten, eaten)
}

func keySinkOnSetFocus(this uintptr, foreground uintptr) uintptr {
	return guardedCall("ITfKeyEventSink.OnSetFocus", func() uintptr {
		service := (*keySinkComObj)(unsafe.Pointer(this)).Impl().(*tipImpl)
		strategyLog("TSF OnSetFocus foreground=%v pid=%d", foreground != 0, os.Getpid())
		// The candidate list belongs to one document: TSF reports the focus
		// move here and the list has to go while another application owns the
		// text. Coming back puts it back while the composition is still alive,
		// exactly like the C++ service.
		if foreground != 0 {
			service.restoreCandidates()
		} else {
			service.candidateUI.suspend()
		}
		return hresultToUintptr(int32(win32.S_OK))
	})
}

func keySinkOnTestKeyDown(this uintptr, _ uintptr, key uintptr, lParam uintptr, eaten uintptr) uintptr {
	return guardedCall("ITfKeyEventSink.OnTestKeyDown", func() uintptr {
		service := (*keySinkComObj)(unsafe.Pointer(this)).Impl().(*tipImpl)
		service.keyMu.Lock()
		defer service.keyMu.Unlock()
		if winSpaceBypass(key) {
			strategyLog("TSF OnTestKeyDown bypass Win+Space vk=0x%x", key)
			eat("OnTestKeyDown", eaten, false)
			return hresultToUintptr(int32(win32.S_OK))
		}
		_, consume := service.runLocalStrategy("testKeyDown", key)
		eat("OnTestKeyDown", eaten, consume)
		return hresultToUintptr(int32(win32.S_OK))
	})
}

func keySinkOnTestKeyUp(this uintptr, _ uintptr, key uintptr, _ uintptr, eaten uintptr) uintptr {
	return guardedCall("ITfKeyEventSink.OnTestKeyUp", func() uintptr {
		eat("OnTestKeyUp", eaten, false)
		return hresultToUintptr(int32(win32.S_OK))
	})
}

func keySinkOnKeyDown(this uintptr, context uintptr, key uintptr, lParam uintptr, eaten uintptr) uintptr {
	return guardedCall("ITfKeyEventSink.OnKeyDown", func() uintptr {
		service := (*keySinkComObj)(unsafe.Pointer(this)).Impl().(*tipImpl)
		// One key at a time per service: the request id, the pipe handle and the
		// composition are all single-threaded state, and RequestEditSession
		// blocks until the application accepts the edit session.
		service.keyMu.Lock()
		consume := service.applyKeyLocked(context, key, lParam)
		// SetSelection moves the caret behind committed text, but it notifies
		// the application, which can re-enter the TIP through a key callback.
		// Holding keyMu across that call would deadlock against this very
		// callback, so the lock is released before the caret is placed.
		service.keyMu.Unlock()
		service.placePendingCaret(context)
		eat("OnKeyDown", eaten, consume)
		return hresultToUintptr(int32(win32.S_OK))
	})
}

// applyKeyLocked runs one key under keyMu and reports whether the TIP kept it.
func (t *tipImpl) applyKeyLocked(context uintptr, key, lParam uintptr) bool {
	if winSpaceBypass(key) {
		strategyLog("TSF OnKeyDown bypass Win+Space vk=0x%x", key)
		return false
	}
	strategyLog("StrategyTip OnKeyDown begin context=%#x vk=0x%x key=%q lParam=0x%x compositionHandle=%#x written=%q", context, key, keyName(key), lParam, uintptr(t.composition.Load()), t.writtenComposition())
	reply, replied := t.runLocalStrategy("keyDown", key)
	consume := keyDecision(reply, replied)
	strategyLog("StrategyTip OnKeyDown reply context=%#x vk=0x%x replied=%v consume=%v reply=%s", context, key, replied, consume, describeReply(reply))
	// The composition is only written for a key the engine keeps, and only
	// with the context this very callback was handed.
	if context != 0 {
		if consume {
			caret := t.applyComposition(context, reply)
			t.notifyCandidates(reply, caret)
		} else {
			t.forgetComposition(context)
		}
	}
	return consume
}

func keySinkOnKeyUp(this uintptr, _ uintptr, key uintptr, lParam uintptr, eaten uintptr) uintptr {
	return guardedCall("ITfKeyEventSink.OnKeyUp", func() uintptr {
		service := (*keySinkComObj)(unsafe.Pointer(this)).Impl().(*tipImpl)
		service.keyMu.Lock()
		defer service.keyMu.Unlock()
		eat("OnKeyUp", eaten, false)
		return hresultToUintptr(int32(win32.S_OK))
	})
}

func keySinkOnPreservedKey(this uintptr, _ uintptr, key uintptr, eaten uintptr) uintptr {
	return guardedCall("ITfKeyEventSink.OnPreservedKey", func() uintptr {
		strategyLog("TSF OnPreservedKey vk=0x%x", key)
		eat("OnPreservedKey", eaten, false)
		return hresultToUintptr(int32(win32.S_OK))
	})
}
