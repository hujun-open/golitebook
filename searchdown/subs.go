// subs
package searchdown

import (
	"fmt"

	"github.com/hujun-open/golitebook/plugin"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/hujun-open/dvlist"
)

type SubscriptionWin struct {
	fyne.Window
	lv          *dvlist.DVList
	subs        *plugin.SubscriptionList
	downloader  *Downloader
	loadingDiag *dialog.ProgressInfiniteDialog
}

func NewSubscriptionWin(subs *plugin.SubscriptionList, d *Downloader) *SubscriptionWin {
	r := new(SubscriptionWin)
	r.Window = fyne.CurrentApp().NewWindow("订阅列表")
	r.lv, _ = dvlist.NewDVList(subs, dvlist.WithDoubleClickHandler(r.onDoubleClicked))
	r.loadingDiag = dialog.NewProgressInfinite("loading", "加载中...", r)
	r.loadingDiag.Hide()
	buttonContainer := fyne.NewContainerWithLayout(layout.NewVBoxLayout(),
		widget.NewSeparator(),
		fyne.NewContainerWithLayout(layout.NewGridLayout(4),
			widget.NewButton("更新", r.onUpdate),
			widget.NewButton("阅读", r.onRead),
			widget.NewButton("删除", r.onDel),
			widget.NewButton("取消", r.Hide),
		),
	)
	r.SetContent(fyne.NewContainerWithLayout(
		layout.NewBorderLayout(nil, buttonContainer, nil, nil),
		buttonContainer, r.lv,
	))
	r.subs = subs
	r.downloader = d
	r.Canvas().SetOnTypedKey(r.lv.TypedKey)
	r.Resize(defaultDialogSize)
	r.SetCloseIntercept(r.onClose)
	return r
}

func (swin *SubscriptionWin) onClose() {
	swin.Hide()
}
func (swin *SubscriptionWin) onDoubleClicked(i int) {
	swin.read(i)
}

func (swin *SubscriptionWin) update(i int) {
	if plugin.CurrentSubscriptions.Get()[i].Status() == plugin.DownloadResultWorking {
		dialog.ShowError(fmt.Errorf("已经在更新中..."), swin)
		return
	}
	go plugin.CurrentSubscriptions.Get()[i].Update()
}

func (swin *SubscriptionWin) onUpdate() {
	i := swin.lv.FirstSelected()
	if i < 0 {
		return
	}
	if swin.subs.Get()[i].Status() == plugin.DownloadResultWorking {
		dialog.ShowError(fmt.Errorf("还在下载中..."), swin)
		return
	}
	swin.update(i)

}
func (swin *SubscriptionWin) read(i int) {
	swin.loadingDiag.Show()
	defer swin.loadingDiag.Hide()
	filename := plugin.GetLocalSavedFilePath(plugin.CurrentSubscriptions.Item(i)[0])
	swin.downloader.lvLoadFileHandler(filename)
	swin.Hide()
}

func (swin *SubscriptionWin) onRead() {
	i := swin.lv.FirstSelected()
	if i < 0 {
		return
	}
	if swin.subs.Get()[i].Status() == plugin.DownloadResultWorking {
		dialog.ShowError(fmt.Errorf("还在下载中..."), swin)
		return
	}
	swin.read(i)
}
func (swin *SubscriptionWin) onDel() {
	i := swin.lv.FirstSelected()
	if i < 0 {
		return
	}
	if swin.subs.Get()[i].Status() == plugin.DownloadResultWorking {
		dialog.ShowError(fmt.Errorf("还在下载中..."), swin)
		return
	}

	dialog.ShowConfirm("删除订阅",
		fmt.Sprintf("确定删除订阅 %v?", plugin.CurrentSubscriptions.Item(i)[0]),
		swin.del,
		swin,
	)

}

func (swin *SubscriptionWin) del(confirm bool) {
	if !confirm {
		return
	}
	i := swin.lv.FirstSelected()
	if i < 0 {
		return
	}
	plugin.CurrentSubscriptions.Remove(i)
	swin.lv.SetData(plugin.CurrentSubscriptions)
}
