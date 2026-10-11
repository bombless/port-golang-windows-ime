# Walk control panel

This is a standalone Windows companion desktop app built with Walk and native Windows controls. It is kept in a nested Go module so GUI dependencies do not affect the TSF DLL module.

From this directory, resolve dependencies and run it with:

```powershell
go mod tidy
go run .
```

Build a Windows executable with:

```powershell
go build -o ime-gui.exe .
```

The window can run `go test ./...`, build `build/PortGoWindowsIME.dll`, open the project folder, and open the TSF diagnostic log. It does not register or install the input method.