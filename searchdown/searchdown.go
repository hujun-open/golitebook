// searchdown is the search & download
package searchdown

import (
	"fmt"
	"golitebook/plugin"
	"sync"

	// "log"
	"strings"

	// "time"

	"fyne.io/fyne"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"github.com/hujun-open/dvlist"
)

var defaultDialogSize = fyne.NewSize(1000, 200)

type searchInputDiag struct {
	dialog.Dialog
	form                    *widget.Form
	kwEntry, pluginDesc     *widget.Entry
	pluginNameList          *widget.Select
	keyword, selectedPlugin string
	closeHandler            func(bool)
}

func (sd *searchInputDiag) onKWChange(s string) {
	sd.keyword = strings.TrimSpace(s)
}
func (sd *searchInputDiag) onSelectChange(s string) {
	if s == allPluginLabelTxt {
		sd.selectedPlugin = allPluginLabelTxt
		sd.pluginDesc.SetText("所有插件")
		return
	}
	desc, err := plugin.LoadedPlugins[s].GetDesc()
	if err == nil {
		sd.pluginDesc.SetText(desc)
		sd.selectedPlugin = plugin.LoadedPlugins[s].Name

	} else {
		sd.pluginDesc.SetText(err.Error())
		sd.selectedPlugin = ""
	}
}

const allPluginLabelTxt = "全部"

func newSearchInputDiag(parent fyne.Window, closehandler func(bool)) *searchInputDiag {
	r := new(searchInputDiag)
	r.form = widget.NewForm()
	r.kwEntry = widget.NewEntry()
	r.kwEntry.OnChanged = r.onKWChange
	r.pluginDesc = widget.NewMultiLineEntry()
	nameList := append([]string{allPluginLabelTxt}, plugin.LoadedPlugins.NameList()...)
	r.pluginNameList = widget.NewSelect(nameList, r.onSelectChange)
	r.pluginNameList.SetSelectedIndex(0)
	r.form.Append("关键词：", r.kwEntry)
	r.form.Append("插件：", r.pluginNameList)
	r.form.Append("", r.pluginDesc)
	r.Dialog = dialog.NewCustomConfirm("搜索", "搜索", "取消", r.form, closehandler, parent)
	return r

}

type searchResultDiag struct {
	win                                                fyne.Window
	data                                               plugin.SearchResultList
	lv                                                 *dvlist.DVList
	cancelButton, downloadButton                       *widget.Button
	overallContainer, innerBContainer, buttonContainer *fyne.Container
	sep                                                *widget.Separator
	downloadHandler                                    func(pluginName, name, bookurl string)
	selected                                           int
	mux                                                *sync.RWMutex
}

func (srd *searchResultDiag) setData(d plugin.SearchResultList) {
	srd.mux.Lock()
	defer srd.mux.Unlock()
	srd.data = d
	srd.lv.SetData(d)

}
func (srd *searchResultDiag) onClose() {
	srd.win.Hide()
}

func (srd *searchResultDiag) onOK() {
	srd.selected = srd.lv.FirstSelected()
	if srd.selected == -1 {
		return
	}
	srd.mux.RLock()
	defer srd.mux.RUnlock()
	srd.downloadHandler(
		srd.data[srd.selected].PluginName,
		srd.data[srd.selected].BookName,
		srd.data[srd.selected].BookPageURL)
}

func newSearchResultDiag(data plugin.SearchResultList, dh func(string, string, string)) *searchResultDiag {
	r := new(searchResultDiag)
	r.data = data
	r.mux = new(sync.RWMutex)
	r.lv, _ = dvlist.NewDVList(data)
	r.win = fyne.CurrentApp().NewWindow("搜索结果")
	r.downloadButton = widget.NewButton("下载", r.onOK)
	r.cancelButton = widget.NewButton("取消", r.onClose)
	r.innerBContainer = fyne.NewContainerWithLayout(
		layout.NewGridLayout(2),
		r.downloadButton, r.cancelButton)
	r.sep = widget.NewSeparator()
	r.buttonContainer = fyne.NewContainerWithLayout(
		layout.NewVBoxLayout(),
		r.sep,
		r.innerBContainer,
	)

	r.overallContainer = fyne.NewContainerWithLayout(
		layout.NewBorderLayout(nil, r.buttonContainer, nil, nil),
		r.buttonContainer, r.lv)
	r.win.SetContent(r.overallContainer)
	r.win.Resize(defaultDialogSize)
	r.downloadHandler = dh
	r.win.SetCloseIntercept(r.onClose)
	r.win.Canvas().SetOnTypedKey(r.lv.TypedKey)
	return r
}

type Downloader struct {
	parent     fyne.Window
	searchDiag *searchInputDiag
	resultDiag *searchResultDiag
	subsDiag   *SubscriptionWin
	// subList           plugin.SubscriptionList
	lvLoadFileHandler func(string)
}

func NewDownloader(win fyne.Window, h func(string)) *Downloader {
	r := &Downloader{parent: win, lvLoadFileHandler: h}
	r.initSubs()
	return r
}

// ShowSearchDiag returns the name of selected plugin and keyword
func (down *Downloader) ShowSearchDiag() {
	if down.searchDiag == nil {
		down.searchDiag = newSearchInputDiag(down.parent, down.onCloseSearchInputDiag)
	} else {
		down.searchDiag.kwEntry.SetText(down.searchDiag.keyword)
	}
	down.searchDiag.Show()
}

func (down *Downloader) ShowSubsWin() {
	if down.subsDiag == nil {
		down.subsDiag = NewSubscriptionWin(plugin.CurrentSubscriptions, down)
		down.subsDiag.Resize(fyne.NewSize(1000, 200))
	}
	down.subsDiag.Show()
}

func (down *Downloader) onCloseSearchInputDiag(confirm bool) {
	if !confirm {
		return
	}
	if down.searchDiag.keyword == "" || down.searchDiag.selectedPlugin == "" {
		dialog.ShowError(fmt.Errorf("selected plugin or kw is empty, %v, %v", down.searchDiag.selectedPlugin, down.searchDiag.keyword), down.parent)
		return
	}
	//do the search
	var rlist plugin.SearchResultList
	var err error
	if down.searchDiag.selectedPlugin == allPluginLabelTxt {
		diag := dialog.NewProgress("搜索", "搜索中...", down.parent)
		diag.Show()
		i := 0
		for _, p := range plugin.LoadedPlugins {
			results, err := p.SearchBook(down.searchDiag.keyword)
			if err != nil {
				dialog.ShowError(err, down.parent)
			} else {
				rlist = append(rlist, results...)
			}
			diag.SetValue(float64(i) / float64(len(plugin.LoadedPlugins)))
			i++
		}
		diag.Hide()

	} else {
		rlist, err = plugin.LoadedPlugins[down.searchDiag.selectedPlugin].SearchBook(down.searchDiag.keyword)
		if err != nil {
			dialog.ShowError(err, down.parent)
			return
		}
	}
	if down.resultDiag == nil {
		down.resultDiag = newSearchResultDiag(rlist, down.download)
	} else {
		down.resultDiag.setData(rlist)
	}
	down.resultDiag.win.Show()
}

func (down *Downloader) initSubs() {
	if plugin.CurrentSubscriptions.Len() > 0 {
		for _, s := range plugin.CurrentSubscriptions.Get() {
			s.SetHandler(down.onDownloadProgress)
		}
	}
}

func (down *Downloader) download(pluginName, name, url string) {
	if down.resultDiag != nil {
		down.resultDiag.onClose()
	}
	sub := plugin.NewSubscription(name, url, pluginName, down.onDownloadProgress)
	plugin.CurrentSubscriptions.Append(sub)
	if down.subsDiag == nil {
		down.subsDiag = NewSubscriptionWin(plugin.CurrentSubscriptions, down)
	} else {
		down.subsDiag.lv.SetData(plugin.CurrentSubscriptions)
	}
	down.subsDiag.Show()
	sub.Update()
}

func (down *Downloader) onDownloadProgress(bookurl string, done, total int) {
	if total == 0 {
		dialog.ShowError(fmt.Errorf("没有更新的章节"), down.subsDiag)
		return
	}
	found := false
	for i := 0; i < plugin.CurrentSubscriptions.Len(); i++ {
		if plugin.CurrentSubscriptions.Item(i)[2] == bookurl {
			newstatus := fmt.Sprintf("下载中 %d/%d", done, total)
			if done == total {
				newstatus = fmt.Sprintf("下载完毕 %d/%d", done, total)
			}
			plugin.CurrentSubscriptions.Get()[i].SetStatusTxt(newstatus)
			found = true
			break
		}
	}
	if found {
		down.subsDiag.lv.SetData(plugin.CurrentSubscriptions)
	}
}
func (down *Downloader) CloseAllWindow() {
	if down.resultDiag != nil {
		down.resultDiag.win.Close()
	}
	if down.subsDiag != nil {
		down.subsDiag.Close()
	}
}
