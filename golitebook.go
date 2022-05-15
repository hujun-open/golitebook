// main
package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hujun-open/golitebook/conf"
	"github.com/hujun-open/golitebook/mainwindow"

	// _ "github.com/hujun-open/golitebook/plugin"

	"fyne.io/fyne/v2/app"
)

func main() {
	if mainwindow.VERSION != "" {
		logfpath := filepath.Join(conf.ConfDir(), "litebook.log")
		f, err := os.OpenFile(logfpath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("failed to open log file")
		} else {
			log.SetOutput(f)
			defer f.Close()
		}
	}
	profile := flag.Bool("p", false, "enable profiling")
	fileToOpen := flag.String("f", "", "file to open")
	flag.Parse()
	log.SetFlags(log.Ltime | log.Lshortfile)
	if *profile || mainwindow.VERSION == "" {
		runtime.SetBlockProfileRate(1000000000)
		go func() {
			log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}
	os.Setenv("FYNE_SCALE", "1.0")
	myApp := app.NewWithID("golitebook")
	myWindow, err := mainwindow.NewLBWindow(myApp, *fileToOpen)
	if err != nil {
		log.Fatal(err)
	}
	myWindow.ShowAndRun()
}
