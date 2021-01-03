// main
package main

import (
	"flag"
	"golitebook/conf"
	"golitebook/mainwindow"
	_ "golitebook/plugin"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/app"
)

//TODO:1. change background 2.change look 4. change font 5.add paragraph prefix via liteview
func main() {
	if mainwindow.VERSION != "" {
		logfpath := filepath.Join(conf.ConfDir(), "litebook.log")
		f, err := os.OpenFile(logfpath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("failed to open log file")
		}
		log.SetOutput(f)
		defer f.Close()
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
