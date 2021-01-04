// mainwindow
package mainwindow

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"time"

	"github.com/hujun-open/golitebook/char"
	"github.com/hujun-open/golitebook/conf"
	"github.com/hujun-open/golitebook/history"
	"github.com/hujun-open/golitebook/plugin"
	"github.com/hujun-open/golitebook/searchdown"
	"github.com/hujun-open/golitebook/toc"

	"fyne.io/fyne"
	// "fyne.io/fyne/canvas"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/driver/desktop"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/storage"

	"github.com/hujun-open/golitebook/liteview"

	"github.com/hujun-open/sbar"
	"github.com/hujun-open/tiledback"
)

var VERSION string

type LBWindow struct {
	fyne.Window
	lv                 *liteview.LiteView
	scrollBar          *sbar.SBar
	userInitatedScroll *uint32
	det                *char.DetChar
	cfg                *conf.Config
	downloader         *searchdown.Downloader
	actMap             actionMap
	helpWin            dialog.Dialog
	tocWin             *toc.ToCDialog
	openFileDiag       *dialog.FileDialog
	selectFontFileDiag *dialog.FileDialog
	currentBook        string
	tocUnchanged       bool
}

func NewLBWindow(myApp fyne.App, filename string) (*LBWindow, error) {
	r := new(LBWindow)
	r.Window = myApp.NewWindow("LiteBook")
	r.SetOnClosed(r.onClose)
	r.det = char.NewDetChar()

	r.cfg = new(conf.Config)
	r.actMap = make(map[liteActType]*liteAct)

	r.loadDefaultKeyMap()
	for _, s := range r.actMap {
		r.Canvas().AddShortcut(s.skey, s.handler)
	}
	var err error
	r.cfg, err = conf.LoadConfigFile()
	if err != nil {
		log.Printf("failed to load config, %v", err)
		if r.cfg == nil {
			return nil, fmt.Errorf("failed to load default config, %v", err)
		}
		log.Printf("using default config")
	}
	myApp.Settings().SetTheme(r.cfg.Theme)
	r.lv = liteview.NewLiteViewCustom(r,
		liteview.WithUnderline(liteview.UnderLineMode(r.cfg.Theme.GetUnderline())),
		liteview.WithLeadingSpaces(2),
	)
	r.lv.SetKeyEvtHandler(r.onKey)
	r.lv.SetPosEvtHandler(r.onChangePos)
	r.userInitatedScroll = new(uint32)
	atomic.StoreUint32(r.userInitatedScroll, 0)
	r.scrollBar = sbar.NewSBar(r.onScrollChange, false)
	lvcontainer := fyne.NewContainerWithLayout(layout.NewMaxLayout())
	if r.cfg.BackgroundFile != "" {
		back, err := tiledback.NewTileBackgroundFromFile(r.cfg.BackgroundFile)
		if err != nil {
			log.Printf("unable to load background image, %v", err)
		}
		lvcontainer.Add(back)
	}
	lvcontainer.Add(r.lv)
	r.SetContent(fyne.NewContainerWithLayout(
		layout.NewBorderLayout(nil, nil, nil, r.scrollBar),
		r.scrollBar, lvcontainer))
	uri := storage.NewURI(r.cfg.LastFile)
	if filename != "" {
		uri = storage.NewFileURI(filename)
	}
	r.loadFileFromURI(uri)
	r.helpWin = dialog.NewInformation("帮助", r.getHelpStr(), r)
	r.helpWin.Hide()
	icon, _ := fyne.LoadResourceFromPath(filepath.Join(conf.GetExecDir(), "icon.png"))
	r.Resize(r.cfg.LastWinSize)
	r.SetMaster()
	r.SetIcon(icon)
	r.Canvas().Focus(r.lv)
	return r, nil
}

type liteActType int

//NOTE to add new function, add const actXXX and update loadDefaultKeyMap()
const (
	actOpenFile liteActType = iota
	actSearchAndDownload
	actShowSubscriptionWin
	actFormatTxt
	actHelp
	actShowTOC
	actShowUnderline
	actFullScreen
	actQuit
	actSelectFontFile
)

func (at liteActType) String() string {
	switch at {
	case actOpenFile:
		return "打开文件"
	case actSearchAndDownload:
		return "网络搜索"
	case actShowSubscriptionWin:
		return "订阅管理"
	case actFormatTxt:
		return "智能分段"
	case actHelp:
		return "显示帮助"
	case actShowTOC:
		return "章节列表"
	case actShowUnderline:
		return "设置下划线"
	case actFullScreen:
		return "全屏显示"
	case actQuit:
		return "退出"
	case actSelectFontFile:
		return "选择字体文件"
	}
	return "未知"
}

type liteAct struct {
	skey    *desktop.CustomShortcut
	handler func(fyne.Shortcut)
}
type actionMap map[liteActType]*liteAct

func (actmap actionMap) String() string {
	r := ""
	skeystr := func(act *liteAct) string {
		mod := ""
		if act.skey.Modifier&desktop.ControlModifier != 0 {
			mod += "Ctrl+"
		}
		if act.skey.Modifier&desktop.AltModifier != 0 {
			mod += "Alt+"
		}
		if act.skey.Modifier&desktop.ShiftModifier != 0 {
			mod += "Shift+"
		}

		return mod + string(act.skey.KeyName)
	}
	keys := make([]int, 0, len(actmap))
	for k := range actmap {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		r += fmt.Sprintf("%v:%v\n", liteActType(k), skeystr(actmap[liteActType(k)]))

	}
	return r
}

func (win *LBWindow) getHelpStr() string {
	ctrlHelpStr := `
滚行: Up, Down, J, K, 鼠标滚轮
翻页: PageUp, PageDown, Left, Right, Space
首页: Home
末页: End
放大缩小字体： =/-
	`
	verStr := VERSION
	if verStr == "" {
		verStr = "internal"
	}
	return fmt.Sprintf("%v\n %v\n ver %v\n\n Hu Jun@2021\ngithub.com/hujun-open/golitebook", ctrlHelpStr, win.actMap.String(), verStr)
}

func (win *LBWindow) loadDefaultKeyMap() {
	win.actMap = map[liteActType]*liteAct{
		actOpenFile: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyO,
				Modifier: desktop.ControlModifier,
			},
			handler: win.openFileviaShortcut,
		},
		actSearchAndDownload: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyC,
				Modifier: desktop.AltModifier,
			},
			handler: win.SearchAndDownload,
		},
		actShowSubscriptionWin: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyY,
				Modifier: desktop.ControlModifier,
			},
			handler: win.ShowSubs,
		},
		actFormatTxt: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyF,
				Modifier: desktop.ControlModifier | desktop.AltModifier,
			},
			handler: win.FormatVal,
		},
		actHelp: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyH,
				Modifier: desktop.ControlModifier,
			},
			handler: win.ShowHelp,
		},
		actShowTOC: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyU,
				Modifier: desktop.ControlModifier,
			},
			handler: win.ShowTOC,
		},
		actShowUnderline: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyL,
				Modifier: desktop.ControlModifier,
			},
			handler: win.ShowUnderline,
		},
		actFullScreen: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyP,
				Modifier: desktop.ControlModifier,
			},
			handler: win.fullScreen,
		},
		actQuit: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyW,
				Modifier: desktop.ControlModifier,
			},
			handler: win.quit,
		},
		actSelectFontFile: &liteAct{
			skey: &desktop.CustomShortcut{
				KeyName:  fyne.KeyZ,
				Modifier: desktop.AltModifier,
			},
			handler: win.showFontFileSelectionDiag,
		},
	}
}
func (win *LBWindow) initFromValue(val [][]byte, bookname string) {
	win.tocUnchanged = false
	if win.currentBook != "" {
		history.History.Update(win.currentBook, win.lv.StartLine)
	}
	win.currentBook = bookname
	win.lv.SetBytes(val)
	win.lv.JumpTo(history.History.GetStartLine(bookname), 0, false)
	win.setTitle(bookname)
	win.Canvas().Focus(win.lv)
}

func (win *LBWindow) onClose() {
	if win.currentBook != "" {
		history.History.Update(win.currentBook, win.lv.StartLine)
	}
	history.History.Save()
	plugin.CurrentSubscriptions.Save()
	if win.downloader != nil {
		win.downloader.CloseAllWindow()
	}
	if win.tocWin != nil {
		win.tocWin.Close()
	}

	os.MkdirAll(conf.ConfDir(), 0755)
	win.cfg.LastWinSize = win.Canvas().Size()
	win.cfg.Theme.SetTextSize(fyne.CurrentApp().Settings().Theme().TextSize())
	err := conf.SaveConfigFile(win.cfg)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to save config, %v", err), win)
	}
	plugin.LoadedPlugins.KillAll()
}

const FontSizeMin = 6

func (win *LBWindow) changeFontSize(increase bool) {
	t := conf.NewLookFromTheme(fyne.CurrentApp().Settings().Theme())
	if increase {
		t.SetTextSize(t.TextSize() + 1)
	} else {
		t.SetTextSize(t.TextSize() - 1)
		if t.TextSize() < FontSizeMin {
			t.SetTextSize(FontSizeMin)
		}
	}
	fyne.CurrentApp().Settings().SetTheme(t)

}

func (win *LBWindow) onScrollChange(newpos uint32) {

	win.Canvas().Focus(win.lv)
	defer win.Canvas().Focus(win.lv)
	// if atomic.LoadUint32(win.userInitatedScroll) != 1 {
	// 	atomic.StoreUint32(win.userInitatedScroll, 1)
	// 	return
	// }
	newlineid := ((len(win.lv.Val()) * int(newpos)) / int(sbar.OffsetResolution))
	if newlineid >= len(win.lv.Val()) {
		newlineid = len(win.lv.Val()) - 1
	}
	// log.Printf("onScrollChange offset %d, jump to line %d", newpos, newlineid)
	win.lv.JumpTo(newlineid, 0, false)

}

func (win *LBWindow) onChangePos(lineid, linepos int) {
	atomic.StoreUint32(win.userInitatedScroll, 0)
	lines := uint32(len(win.lv.Val()))
	if lines > 0 {
		win.scrollBar.SetOffset((uint32(lineid) * sbar.OffsetResolution) / lines)
	}
}

func (win *LBWindow) onKey(evt *fyne.KeyEvent) {
	switch evt.Name {
	case fyne.KeyEqual:
		win.changeFontSize(true)
	case fyne.KeyMinus:
		win.changeFontSize(false)

	}
}

func (win *LBWindow) loadFileFromPath(p string) {
	err := win.loadFileFromURI(storage.NewFileURI(p))
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to load %v, %v", p, err), win)
	}
}

func (win *LBWindow) loadFont(furl fyne.URIReadCloser, err error) {
	if err != nil {
		dialog.ShowError(err, win)
		return
	}
	if furl == nil {
		return
	}
	res, err := storage.LoadResourceFromURI(furl.URI())
	if err != nil {
		dialog.ShowError(err, win)
		return
	}
	win.cfg.Theme.SetFont(res)
	fyne.CurrentApp().Settings().SetTheme(win.cfg.Theme)
	time.Sleep(100 * time.Millisecond) //this delay is needed, otherwise the font change won't take effect
	win.lv.SetBytes(win.lv.GetVal())
	win.cfg.Theme.RegularFontPath = furl.URI().String()
}

func (win *LBWindow) loadFile(furl fyne.URIReadCloser, err error) {
	if err != nil {
		dialog.ShowError(err, win)
		return
	}
	if furl == nil {
		return
	}
	er := win.loadFileFromURI(furl.URI())
	if er != nil {
		dialog.ShowError(er, win)
	}
}

func (win *LBWindow) loadFileFromURI(furl fyne.URI) error {
	ureader, err := storage.OpenFileFromURI(furl)
	if err != nil {
		return err
	}
	buf, err := ioutil.ReadAll(ureader)
	if err != nil {
		return err
	}
	r, err := win.det.ToByteList(buf)
	if err != nil {
		return err
	}
	win.initFromValue(r, furl.Name())
	win.Canvas().Focus(win.lv)
	win.cfg.LastFile = furl.String()
	// win.SetTitle(fmt.Sprintf("Litebook %v", furl.Name()))
	return nil
}

func (win *LBWindow) setTitle(bookname string) {
	win.SetTitle(fmt.Sprintf("Litebook %v     ------ Ctrl-H 帮助", bookname))
}

func (win *LBWindow) openFileviaShortcut(fyne.Shortcut) {
	win.openFile()
}

func (win *LBWindow) openFile() {
	if win.openFileDiag == nil {
		win.openFileDiag = dialog.NewFileOpen(win.loadFile, win)
		win.openFileDiag.SetDismissText("打开")
	}
	if win.cfg.LastFile != "" {
		lastdiruri, err := storage.Parent(storage.NewURI(win.cfg.LastFile))
		if err == nil {
			lurl, err := storage.ListerForURI(lastdiruri)
			if err == nil {
				win.openFileDiag.SetLocation(lurl)
			} else {
				log.Printf("failed to get lister, %v", err)
			}
		} else {
			log.Printf("failed to get parent, %v", err)
		}
	}
	win.openFileDiag.Show()
}

func (win *LBWindow) SearchAndDownload(fyne.Shortcut) {
	if win.downloader == nil {
		win.downloader = searchdown.NewDownloader(win, win.loadFileFromPath)
	}

	win.downloader.ShowSearchDiag()
}

func (win *LBWindow) ShowSubs(fyne.Shortcut) {
	if win.downloader == nil {
		win.downloader = searchdown.NewDownloader(win, win.loadFileFromPath)

	}

	win.downloader.ShowSubsWin()
}

func (win *LBWindow) FormatVal(fyne.Shortcut) {
	diag := dialog.NewProgress("分段", "智能分段中...", win)
	diag.Show()
	newval := plugin.FormatTxt(win.lv.GetVal(),
		plugin.DefaultFormatMinimalLineWidthInChars, diag.SetValue)
	win.initFromValue(newval, win.currentBook)
	diag.Hide()

}
func (win *LBWindow) ShowHelp(fyne.Shortcut) {
	win.helpWin.Show()
}

func (win *LBWindow) jumptoChapter(name string, line int) {
	win.lv.JumpTo(line, 0, true)
}

func (win *LBWindow) ShowTOC(fyne.Shortcut) {
	if win.tocWin == nil {
		win.tocWin = toc.NewToCDialog(win.lv.GetVal(), win, win.jumptoChapter)
	}
	if !win.tocUnchanged {
		win.tocWin.Set(win.lv.GetVal())
		win.tocUnchanged = true
	}
	win.tocWin.SetSelection(win.lv.StartLine)
	win.tocWin.Show()
}

func (win *LBWindow) ShowUnderline(fyne.Shortcut) {
	curUnder := win.lv.GetUnderLine()
	switch curUnder {
	case liteview.UnderLineNone:
		curUnder = liteview.UnderLineSolid
	case liteview.UnderLineSolid:
		curUnder = liteview.UnderLineDash
	case liteview.UnderLineDash:
		curUnder = liteview.UnderLineNone

	}
	win.cfg.Theme.SetUnderline(int(curUnder))
	win.lv.SetUnderLine(curUnder)
}
func (win *LBWindow) fullScreen(fyne.Shortcut) {
	win.SetFullScreen(!win.FullScreen())
}
func (win *LBWindow) quit(fyne.Shortcut) {
	win.Close()
}

func (win *LBWindow) showFontFileSelectionDiag(fyne.Shortcut) {
	if win.selectFontFileDiag == nil {
		win.selectFontFileDiag = dialog.NewFileOpen(win.loadFont, win)
		win.selectFontFileDiag.SetFilter(storage.NewExtensionFileFilter([]string{".ttf"}))
	}
	win.selectFontFileDiag.Show()
}
