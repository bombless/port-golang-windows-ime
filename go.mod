module port-golang-windows-ime

go 1.23

require github.com/zzl/go-win32api/v2 v2.2.0

require github.com/zzl/go-com v1.5.0

require (
	github.com/lxn/walk v0.0.0
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e
)

require (
	golang.org/x/sys v0.0.0-20220330033206-e17cdc41300f // indirect
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
)

replace github.com/zzl/go-com => ./third_party/go-com

replace github.com/lxn/walk => ./third_party/walk
