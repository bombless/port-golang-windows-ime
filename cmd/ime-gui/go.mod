module port-golang-windows-ime/cmd/ime-gui

go 1.23

require github.com/lxn/walk v0.0.0-20210112085537-c389da54e794

require (
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	golang.org/x/sys v0.0.0-20220330033206-e17cdc41300f // indirect
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
)

replace github.com/lxn/walk => ../../third_party/walk
