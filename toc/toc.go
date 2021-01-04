// toc
package toc

import (
	"github.com/hujun-open/golitebook/plugin"
	// "log"
	"sort"

	"fyne.io/fyne"
	// "fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"github.com/hujun-open/dvlist"
)

// key is the chapter name, int the is the its line id
type ChapterLocation struct {
	Name      string
	StartLine int
}

type ChapterLocationList []ChapterLocation

func (clist ChapterLocationList) Swap(i, j int) {
	clist[i], clist[j] = clist[j], clist[i]
}
func (clist ChapterLocationList) Less(i, j int) bool {
	return clist[i].StartLine < clist[j].StartLine
}
func (clist ChapterLocationList) Len() int {
	return len(clist)
}
func (clist ChapterLocationList) Fields() []string {
	return []string{"章节"}
}
func (clist ChapterLocationList) Item(id int) []string {
	return []string{clist[id].Name}
}
func (clist ChapterLocationList) Sort(field int, ascend bool) {
	sort.Sort(clist)
}
func (clist ChapterLocationList) Filter(kw string, i int) {

}

type GOTOChapterHandler func(name string, lineid int)
type ToCDialog struct {
	fyne.Window
	toc  ChapterLocationList
	list *dvlist.DVList
	h    GOTOChapterHandler
}

func (tocdiag *ToCDialog) read(chapid int) {
	if tocdiag.h != nil {
		tocdiag.h(tocdiag.toc[chapid].Name, tocdiag.toc[chapid].StartLine)
	}
	tocdiag.Hide()
}

func (tocdiag *ToCDialog) Set(linelist [][]byte) {
	tocdiag.toc = newChapterLocationListviaByteLineList(linelist)
	tocdiag.list.SetData(tocdiag.toc)
}

func (tocdiag *ToCDialog) SetSelection(curStartline int) {
	var i int
	var ch ChapterLocation
	for i, ch = range tocdiag.toc {
		if curStartline < ch.StartLine {
			break
		}
	}
	i--
	if i < 0 {
		i = 0
	}
	tocdiag.list.ScrollTo(i)
	tocdiag.list.SetSelection(i, true)
}
func (tocdiag *ToCDialog) Refresh() {
	tocdiag.list.Refresh()
}

func newToCDialog(toc ChapterLocationList, parent fyne.Window, h GOTOChapterHandler) *ToCDialog {
	r := new(ToCDialog)
	r.Window = fyne.CurrentApp().NewWindow("章节列表")
	r.SetCloseIntercept(func() { r.Hide() })
	r.toc = toc
	r.h = h
	r.list, _ = dvlist.NewDVList(r.toc, dvlist.WithDoubleClickHandler(r.read))
	button := widget.NewButton("关闭", r.Hide)
	r.SetContent(fyne.NewContainerWithLayout(layout.NewBorderLayout(nil, button, nil, nil), button, r.list))
	r.Resize(fyne.NewSize(500, 800))
	r.Canvas().SetOnTypedKey(r.list.TypedKey)
	return r
}
func NewToCDialog(val [][]byte, parent fyne.Window, h GOTOChapterHandler) *ToCDialog {
	return newToCDialog(newChapterLocationListviaByteLineList(val), parent, h)
}

func newChapterLocationListviaByteLineList(val [][]byte) ChapterLocationList {
	toc := ChapterLocationList{}
	bookmarkRune := []rune(plugin.BookMarkChar)[0]
	for i, line := range val {
		if plugin.IsPrecreatedChapterTitle(line, bookmarkRune) {
			toc = append(toc, ChapterLocation{Name: string(line), StartLine: i})
		}
	}
	return toc
}
