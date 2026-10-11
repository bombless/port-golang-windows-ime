//go:build windows

// registerhost invokes the COM registration exports of the TSF DLL.
package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: registerhost register|unregister <dll>")
		os.Exit(2)
	}

	var export string
	switch os.Args[1] {
	case "register":
		export = "DllRegisterServer"
	case "unregister":
		export = "DllUnregisterServer"
	default:
		fmt.Fprintf(os.Stderr, "unknown action: %s\n", os.Args[1])
		os.Exit(2)
	}

	path, err := syscall.UTF16PtrFromString(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid DLL path: %v\n", err)
		os.Exit(2)
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	loadLibrary := kernel32.NewProc("LoadLibraryW")
	getProcAddress := kernel32.NewProc("GetProcAddress")
	freeLibrary := kernel32.NewProc("FreeLibrary")
	module, _, callErr := loadLibrary.Call(uintptr(unsafe.Pointer(path)))
	if module == 0 {
		fmt.Fprintf(os.Stderr, "LoadLibraryW failed: %v\n", callErr)
		os.Exit(1)
	}
	defer freeLibrary.Call(module)

	name, _ := syscall.BytePtrFromString(export)
	address, _, callErr := getProcAddress.Call(module, uintptr(unsafe.Pointer(name)))
	if address == 0 {
		fmt.Fprintf(os.Stderr, "GetProcAddress(%s) failed: %v\n", export, callErr)
		os.Exit(1)
	}
	hresult, _, _ := syscall.SyscallN(address)
	hr := int32(uint32(hresult))
	fmt.Fprintf(os.Stderr, "%s hr 0x%08X\n", export, uint32(hr))
	if hr != 0 {
		os.Exit(1)
	}
}
