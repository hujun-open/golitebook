module github.com/hujun-open/golitebook

go 1.14

replace github.com/hujun-open/dvlist => c:\hujun\gomodules\src\dvlist

replace github.com/hujun-open/sbar => c:\hujun\gomodules\src\sbar

replace github.com/hujun-open/tiledback => c:\hujun\gomodules\src\tiledback

require (
	fyne.io/fyne v1.4.2
	fyne.io/fyne/v2 v2.0.0
	github.com/gogs/chardet v0.0.0-20191104214054-4b6791f73a28
	github.com/golang/protobuf v1.4.3
	github.com/hujun-open/dvlist v0.1.0
	github.com/hujun-open/sbar v0.1.1
	github.com/hujun-open/tiledback v0.1.0
	golang.org/x/text v0.3.2
	google.golang.org/grpc v1.34.0
	google.golang.org/protobuf v1.25.0
)
