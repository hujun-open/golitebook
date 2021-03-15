// package liteview is a fyne widget for reading text
package liteview

import (
	"fmt"
	"image"
	"image/color"
	"sync/atomic"

	"bytes"
	// "log"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const measureChar = "我"

type renderAction int

const (
	actScrollLineUp renderAction = iota
	actScrollLineDown
	actScrollPageUp
	actScrollPageDown
	actScrollTop
	actScrollBottom
	actScrollToPoS
	actScrollToPoSCental
	actSetUnderline
	actSetVal
)

// getNumLeadingSpaces return number of spaces that has equal width as leadingCount Chinese chars
func getNumLeadingSpaces(leadingCount int) int {
	measureStr := ""
	for i := 0; i < leadingCount; i++ {
		measureStr += measureChar
	}
	targetwidth := fyne.MeasureText(measureStr, theme.TextSize(), fyne.TextStyle{}).Width
	spaceWidth := fyne.MeasureText(" ", theme.TextSize(), fyne.TextStyle{}).Width
	return int(targetwidth/spaceWidth) + 1
}

// input val is a single line of rune, return number of chars n
// so that val[txtpos:txtpos+n] fits in width wid;
// unitWid is the avg single char widwth;
func lineChars(val []rune, txtpos int, wid, unitWid float32) int {
	amount := int(wid / unitWid)
	lineLen := len(val)
	if txtpos >= lineLen || txtpos < 0 {
		return 0
	}
	for {
		if txtpos+amount > lineLen {
			amount = lineLen - txtpos
		}
		if txtpos >= lineLen || txtpos < 0 {
			return 0
		}
		tsize := fyne.MeasureText(string(val[txtpos:txtpos+amount]), theme.TextSize(), fyne.TextStyle{})
		if tsize.Width > wid {
			amount--
		} else {
			if wid-tsize.Width < unitWid {
				return amount
			}
			amount++
			if txtpos+amount > lineLen {
				return lineLen - txtpos
			}
		}

	}
}

// breakLine break a single line into a list of lines,
// so that each line fid into width wid;
// unitWid is the avg single char width;
// if reverse if true, then working on the line[:txtpos] part,
// otherwise working on line[txtpos:] part
func breakLine(line []rune, lineid, txtpos int, wid, unitWid float32, reverse bool) (lines []*renderLine, lastpos int) {
	if len(line) == 0 {
		lines = append(lines,
			&renderLine{
				text:        []rune{},
				runeLine:    lineid,
				runeLinePos: 0,
			})
		return
	}
	var workLine []rune
	if !reverse {
		workLine = line[txtpos:]
	} else {
		workLine = line[:txtpos]
	}
	lastpos = 0
	step := 1
	pos := 0
	for {
		step = lineChars(workLine, lastpos, wid, unitWid)
		if step == 0 {
			break
		}
		pos = lastpos
		if !reverse {
			pos += txtpos
		}
		lines = append(lines, &renderLine{
			text:        workLine[lastpos : lastpos+step],
			runeLine:    lineid,
			runeLinePos: pos,
		})
		lastpos += step
	}
	return
}

type renderLine struct {
	text                  []rune
	runeLine, runeLinePos int //runeLinePos is the pos of first rune
}

func (rl renderLine) String() string {
	return fmt.Sprintf("line %d, pos %d", rl.runeLine, rl.runeLinePos)
}

type liteViewRender struct {
	lv                                                       *LiteView
	curSize                                                  fyne.Size
	overallContainer                                         *fyne.Container
	curStartLine, curStartLinePos, curEndLine, curEndLinePos int
	posMux                                                   *sync.RWMutex
	// lineList is the current on-screen broken lines
	lineList []*renderLine
	unitSize fyne.Size
}

func newLiteViewRender(lv *LiteView) *liteViewRender {
	lvr := new(liteViewRender)
	lvr.lv = lv
	lvr.overallContainer = fyne.NewContainerWithoutLayout()
	lvr.posMux = new(sync.RWMutex)
	lvr.curStartLine = lv.StartLine
	lvr.curStartLinePos = lv.StartLinePos
	lvr.calUnitSize()
	return lvr
}

// calAllowedLines calculate the max lines allowed with height h
func (lvr *liteViewRender) calAllowedLines(h float32) int {
	return int(h / lvr.unitSize.Height)
}

func (lvr *liteViewRender) calUnitSize() (usize fyne.Size) {
	fontSize := fyne.MeasureText(measureChar, theme.TextSize(), fyne.TextStyle{})
	lvr.unitSize = fontSize
	lvr.unitSize.Height += 2*lvr.lv.lineVerticalPadding + 1
	// log.Printf("calculated UnitSize is %v", lvr.unitSize)
	return fontSize
}

func (lvr *liteViewRender) Layout(layoutsize fyne.Size) {
	fsize := lvr.calUnitSize()
	workingArea := fyne.NewSize(layoutsize.Width-2*lvr.lv.sidePadding, layoutsize.Height-2*lvr.lv.verticalPadding)
	if workingArea.Width <= 0 || workingArea.Height <= 0 {
		return
	}
	allowedLines := int(lvr.calAllowedLines(workingArea.Height))
	if allowedLines == 0 {
		return
	}
	lvr.posMux.Lock()
	defer lvr.posMux.Unlock()
	lvr.curSize = workingArea
	lvr.overallContainer = fyne.NewContainerWithoutLayout()
	numFormatLines := 0
	lvr.curEndLine = lvr.curStartLine
	lvr.curEndLinePos = lvr.curStartLinePos
	lvr.lineList = []*renderLine{}
	underLineMode := lvr.lv.GetUnderLine()
	var dashImg image.Image
	if underLineMode == UnderLineDash {
		dashImg = NewDashedLine(workingArea.Width,
			DefaultDashlineHeight, DefaultDashlineWidth,
			DefaultDashlineInterval, theme.TextColor())
	}
L1:
	for txtLine := lvr.curStartLine; txtLine < len(lvr.lv.Val()); txtLine++ {
		brokenlines := []*renderLine{}
		brokenlines, lvr.curEndLinePos = breakLine(bytes.Runes(lvr.lv.Val()[txtLine]),
			txtLine, lvr.curEndLinePos, workingArea.Width, lvr.unitSize.Width, false)
		for _, line := range brokenlines {
			t := canvas.NewText(string(line.text), theme.TextColor())
			txtLineStartPos := fyne.NewPos(lvr.lv.sidePadding,
				lvr.lv.verticalPadding+float32(numFormatLines)*(lvr.unitSize.Height)+lvr.lv.lineVerticalPadding)
			// log.Printf("line %d pos at Y %d", txtLine, txtLineStartPos.Y)
			t.Move(txtLineStartPos)
			lvr.overallContainer.Add(t)

			switch underLineMode {
			case UnderLineSolid, UnderLineDash:
				pos1 := fyne.NewPos(txtLineStartPos.X, txtLineStartPos.Y+fsize.Height+lvr.lv.lineVerticalPadding)
				pos2 := fyne.NewPos(pos1.X+workingArea.Width, pos1.Y)
				switch underLineMode {
				case UnderLineSolid:
					underLine := canvas.NewLine(theme.TextColor())
					underLine.Position1 = pos1
					underLine.Position2 = pos2
					// log.Printf("line %d's underline pos at Y %d", txtLine, underLine.Position1.Y)
					lvr.overallContainer.Add(underLine)
				case UnderLineDash:
					uline := canvas.NewImageFromImage(dashImg)
					uline.Resize(fyne.NewSize(workingArea.Width, DefaultDashlineHeight))
					uline.Move(pos1)
					lvr.overallContainer.Add(uline)
				}

			}
			lvr.lineList = append(lvr.lineList, line)
			numFormatLines++
			if numFormatLines >= allowedLines {
				break L1
			}
		}
		lvr.curEndLinePos = 0
		lvr.curEndLine++
	}
	lvr.lv.valMux.Lock()
	lvr.lv.StartLine = lvr.curStartLine
	lvr.lv.StartLinePos = lvr.curStartLinePos
	lvr.lv.valMux.Unlock()
	if lvr.lv.posEvtHandler != nil {
		lvr.lv.posEvtHandler(lvr.lv.StartLine, lvr.lv.StartLinePos)
	}

}

func (lvr *liteViewRender) BackgroundColor() color.Color {
	return color.Transparent
}
func (lvr *liteViewRender) Destroy() {

}
func (lvr *liteViewRender) MinSize() fyne.Size {
	return fyne.NewSize(lvr.unitSize.Width*20, (lvr.unitSize.Height+theme.Padding())*10)
}

func (lvr *liteViewRender) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{lvr.overallContainer}
}
func (lvr *liteViewRender) lineUp() {
	needRefresh := false
	lvr.posMux.Lock()
	defer func() {
		lvr.posMux.Unlock()
		if needRefresh {
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		}
	}()
	var newline, newpos int
	switch len(lvr.lineList) {
	case 0:
		return
	default:
		if lvr.lineList[0].runeLinePos > 0 {
			llist, _ := breakLine(bytes.Runes(lvr.lv.Val()[lvr.lineList[0].runeLine]),
				lvr.lineList[0].runeLine,
				lvr.lineList[0].runeLinePos,
				lvr.curSize.Width,
				lvr.unitSize.Width,
				true,
			)
			newline = llist[len(llist)-1].runeLine
			newpos = llist[len(llist)-1].runeLinePos
		} else {
			//current first line start from the begging of the rune line
			if lvr.lineList[0].runeLine < 1 {
				//reached the top
				return
			}
			llist, _ := breakLine(bytes.Runes(lvr.lv.Val()[lvr.lineList[0].runeLine-1]),
				lvr.lineList[0].runeLine-1,
				0,
				lvr.curSize.Width,
				lvr.unitSize.Width,
				false,
			)
			if len(llist) == 0 {
				return
			}
			newline = llist[len(llist)-1].runeLine
			newpos = llist[len(llist)-1].runeLinePos
		}
	}
	lvr.curStartLine = newline
	lvr.curStartLinePos = newpos
	needRefresh = true
}

func (lvr *liteViewRender) bottom() {
	needRefresh := false
	lvr.posMux.Lock()
	defer func() {
		lvr.posMux.Unlock()
		if needRefresh {
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		}
	}()
	lastlineIndex := len(lvr.lv.Val()) - 1
	lastlineLen := len(lvr.lv.Val()[lastlineIndex])
	if lvr.lineList[len(lvr.lineList)-1].runeLine == lastlineIndex &&
		lvr.lineList[len(lvr.lineList)-1].runeLinePos+
			len(lvr.lineList[len(lvr.lineList)-1].text) == lastlineLen {
		// already bottom
		return
	}

	lvr.curStartLine = lastlineIndex
	llist, _ := breakLine(bytes.Runes(lvr.lv.Val()[lastlineIndex]),
		lastlineIndex,
		0,
		lvr.curSize.Width,
		lvr.unitSize.Width,
		false,
	)
	allowedLines := lvr.calAllowedLines(lvr.curSize.Height)
	if len(llist) >= allowedLines {
		lvr.curStartLinePos = llist[len(llist)-allowedLines].runeLinePos
	} else {
		lvr.curStartLinePos = 0
	}
	needRefresh = true
}

func (lvr *liteViewRender) top() {
	needRefresh := false
	lvr.posMux.Lock()
	if lvr.curStartLine == 0 && lvr.curStartLinePos == 0 {
		// already top
		return
	}
	defer func() {
		lvr.posMux.Unlock()
		if needRefresh {
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		}
	}()
	lvr.curStartLine = 0
	lvr.curStartLinePos = 0
	needRefresh = true
}
func (lvr *liteViewRender) pageUp() {
	needRefresh := false
	lvr.posMux.Lock()
	defer func() {
		lvr.posMux.Unlock()
		if needRefresh {
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		}
	}()
	if len(lvr.lineList) == 0 {
		return
	}
	if lvr.lineList[0].runeLine == 0 && lvr.lineList[0].runeLinePos == 0 {
		//reached top
		return
	}
	allowedLines := lvr.calAllowedLines(lvr.curSize.Height)
	i := 0
	if lvr.lineList[0].runeLinePos > 0 {
		llist, _ := breakLine(bytes.Runes(lvr.lv.Val()[lvr.lineList[0].runeLine]),
			lvr.lineList[0].runeLine,
			lvr.lineList[0].runeLinePos,
			lvr.curSize.Width,
			lvr.unitSize.Width,
			true,
		)
		if len(llist) >= allowedLines {
			startIndex := len(llist) - allowedLines
			lvr.curStartLine = llist[startIndex].runeLine
			lvr.curStartLinePos = llist[startIndex].runeLinePos
			needRefresh = true
			return
		}
		i = len(llist)
	}
	if lvr.lineList[0].runeLine == 0 {
		lvr.curStartLine = 0
		lvr.curStartLinePos = 0
		needRefresh = true
		return
	}
	for rline := lvr.lineList[0].runeLine - 1; rline >= 0; rline-- {
		llist, _ := breakLine(bytes.Runes(lvr.lv.Val()[rline]),
			rline,
			0,
			lvr.curSize.Width,
			lvr.unitSize.Width,
			false,
		)
		if i+len(llist) >= allowedLines {
			lvr.curStartLine = rline
			lvr.curStartLinePos = llist[len(llist)-(allowedLines-i)].runeLinePos
			needRefresh = true
			return
		}
		i += len(llist)
	}
	lvr.curStartLine = 0
	lvr.curStartLinePos = 0
	needRefresh = true

}
func (lvr *liteViewRender) pageDown() {

	needRefresh := false
	lvr.posMux.Lock()
	defer func() {
		lvr.posMux.Unlock()
		if needRefresh {
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		}
	}()
	if len(lvr.lineList) < lvr.calAllowedLines(lvr.curSize.Height) {
		// reached bottom
		return
	}
	lvr.curStartLine = lvr.lineList[len(lvr.lineList)-1].runeLine
	lvr.curStartLinePos = lvr.lineList[len(lvr.lineList)-1].runeLinePos + len(lvr.lineList[len(lvr.lineList)-1].text)
	needRefresh = true
}

func (lvr *liteViewRender) lineDown() {
	needRefresh := false
	lvr.posMux.Lock()
	defer func() {
		lvr.posMux.Unlock()
		if needRefresh {
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		}
	}()
	var newline, newpos int
	switch len(lvr.lineList) {
	case 0, 1:
		return
	// case 1:
	// 	newpos = lvr.lineList[0].runeLinePos + len(lvr.lineList[0].text)
	// 	if newpos >= len(lvr.lv.Val()[lvr.lineList[0].runeLine]) {
	// 		//go to next line
	// 		if lvr.lineList[0].runeLine+1 >= len(lvr.lv.Val()) {
	// 			//reached end
	// 			return
	// 		}
	// 		newline = lvr.lineList[0].runeLine + 1
	// 		newpos = 0
	// 	} else {
	// 		//still same line
	// 		newline = lvr.lineList[0].runeLine
	// 	}
	default:
		newline = lvr.lineList[1].runeLine
		newpos = lvr.lineList[1].runeLinePos
	}

	lvr.curStartLine = newline
	lvr.curStartLinePos = newpos
	needRefresh = true
}

func (lvr *liteViewRender) scrollToPos(center bool) {
	needRefresh := false
	lvr.posMux.Lock()
	lvr.lv.valMux.RLock()
	defer func() {
		lvr.lv.valMux.RUnlock()
		lvr.posMux.Unlock()
		if needRefresh {
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		}
	}()
	if !center {
		lvr.curStartLine = lvr.lv.StartLine
		lvr.curStartLinePos = lvr.lv.StartLinePos
		needRefresh = true
		return
	}
	allowedLines := lvr.calAllowedLines(lvr.curSize.Height) / 2
	i := 0
	if lvr.lv.StartLinePos > 0 {
		llist, _ := breakLine(bytes.Runes(lvr.lv.Val()[lvr.lv.StartLine]),
			lvr.lv.StartLine,
			lvr.lv.StartLinePos,
			lvr.curSize.Width,
			lvr.unitSize.Width,
			true,
		)
		if len(llist) >= allowedLines {
			startIndex := len(llist) - allowedLines
			lvr.curStartLine = llist[startIndex].runeLine
			lvr.curStartLinePos = llist[startIndex].runeLinePos
			needRefresh = true
			return
		}
		i = len(llist)
	}
	if lvr.lv.StartLine == 0 {
		lvr.curStartLine = 0
		lvr.curStartLinePos = 0
		needRefresh = true
		return
	}
	for rline := lvr.lv.StartLine - 1; rline >= 0; rline-- {
		llist, _ := breakLine(bytes.Runes(lvr.lv.Val()[rline]),
			rline,
			0,
			lvr.curSize.Width,
			lvr.unitSize.Width,
			false,
		)
		if i+len(llist) >= allowedLines {
			lvr.curStartLine = rline
			lvr.curStartLinePos = llist[len(llist)-(allowedLines-i)].runeLinePos
			needRefresh = true
			return
		}
		i += len(llist)
	}
	lvr.curStartLine = 0
	lvr.curStartLinePos = 0
	needRefresh = true

}

func (lvr *liteViewRender) addLeadingSpaces() {
	lvr.lv.valMux.Lock()
	defer lvr.lv.valMux.Unlock()
	count := getNumLeadingSpaces(int(atomic.LoadUint32(lvr.lv.numberOfLeadingSpaces)))
	spaces := []byte{}
	for i := 0; i < count; i++ {
		spaces = append(spaces, []byte(" ")...)
	}

	for i, line := range lvr.lv.lineList {
		line = bytes.TrimLeft(line, " ")
		line = append(spaces, line...)
		lvr.lv.lineList[i] = line
	}
}

func (lvr *liteViewRender) Refresh() {
	lvr.calUnitSize()
	if lvr.curSize.Height == 0 || lvr.curSize.Width == 0 {
		return
	}
	select {
	case act := <-lvr.lv.actionChan:
		switch act {
		case actScrollLineDown:
			lvr.lineDown()
		case actScrollLineUp:
			lvr.lineUp()
		case actScrollPageDown:
			lvr.pageDown()
		case actScrollPageUp:
			lvr.pageUp()
		case actScrollTop:
			lvr.top()
		case actScrollBottom:
			lvr.bottom()
		case actScrollToPoS:
			lvr.scrollToPos(false)
		case actScrollToPoSCental:
			lvr.scrollToPos(true)
		case actSetUnderline:
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		case actSetVal:
			lvr.addLeadingSpaces()
			lvr.Layout(lvr.lv.Size())
			canvas.Refresh(lvr.lv)
		}
	default:
	}

}

type UnderLineMode uint32

const (
	UnderLineNone UnderLineMode = iota
	UnderLineSolid
	UnderLineDash
)

const (
	DefaultDashlineHeight   = 1
	DefaultDashlineWidth    = 5
	DefaultDashlineInterval = 2
)

const (
	DefaultSidePadding         = 50
	DefaultVerticalPadding     = 20
	DefaultLineVerticalPadding = 5
)

type LiteView struct {
	widget.DisableableWidget
	lineList   [][]byte
	actionChan chan renderAction
	valMux     *sync.RWMutex
	// actMux                                            *sync.RWMutex
	underLine                                         *uint32
	keyEvtHandler                                     func(*fyne.KeyEvent)
	StartLine, StartLinePos                           int
	posEvtHandler                                     func(int, int)
	parent                                            fyne.Window
	verticalPadding, sidePadding, lineVerticalPadding float32
	numberOfLeadingSpaces                             *uint32
}

func newLiteView(p fyne.Window) *LiteView {
	lv := new(LiteView)
	lv.ExtendBaseWidget(lv)
	lv.valMux = new(sync.RWMutex)
	lv.actionChan = make(chan renderAction, 16)
	lv.keyEvtHandler = lv.defaultKeyEvtHandler
	lv.parent = p
	lv.verticalPadding = DefaultVerticalPadding
	lv.sidePadding = DefaultSidePadding
	lv.lineVerticalPadding = DefaultLineVerticalPadding
	lv.underLine = new(uint32)
	atomic.StoreUint32(lv.underLine, uint32(UnderLineNone))
	lv.numberOfLeadingSpaces = new(uint32)
	atomic.StoreUint32(lv.numberOfLeadingSpaces, 0)
	return lv
}

type Option func(lv *LiteView)

func WithUnderline(u UnderLineMode) Option {
	return func(lv *LiteView) {
		atomic.StoreUint32(lv.underLine, uint32(u))
	}
}
func WithValBytes(val [][]byte) Option {
	return func(lv *LiteView) {
		lv.SetBytes(val)
	}
}
func WithLeadingSpaces(count int) Option {
	return func(lv *LiteView) {
		atomic.StoreUint32(lv.numberOfLeadingSpaces, uint32(count))
	}
}

func WithStartingPos(lineid, linepos int) Option {
	return func(lv *LiteView) {
		if len(lv.Val()) > 0 {
			targetline := lineid
			if targetline >= len(lv.lineList) {
				targetline = len(lv.lineList) - 1
			}
			if targetline < 0 {
				targetline = 0
			}
			targetlinepos := linepos
			if targetlinepos >= len(lv.lineList[targetline]) {
				targetlinepos = 0
			}

			lv.StartLine = targetline
			lv.StartLinePos = targetlinepos
		} else {
			lv.StartLine = 0
			lv.StartLinePos = 0
		}
	}
}

func NewLiteViewCustom(p fyne.Window, options ...Option) *LiteView {
	lv := newLiteView(p)
	for _, o := range options {
		o(lv)
	}
	return lv
}

func (lv *LiteView) CreateRenderer() fyne.WidgetRenderer {
	lv.ExtendBaseWidget(lv)
	return newLiteViewRender(lv)
}
func (lv *LiteView) SetStr(val string) {
	s := bytes.ReplaceAll([]byte(val), []byte("\r\n"), []byte("\n")) // convert dos to unix line ending
	slist := bytes.Split(s, []byte("\n"))
	rlist := [][]byte{}
	for _, line := range slist {
		rlist = append(rlist, line)
	}
	lv.SetBytes(rlist)
}
func (lv *LiteView) SetBytes(val [][]byte) {
	lv.valMux.Lock()
	lv.lineList = val
	lv.valMux.Unlock()
	lv.renderAct(actSetVal)
}

func (lv *LiteView) Val() [][]byte {
	lv.valMux.RLock()
	defer lv.valMux.RUnlock()
	return lv.lineList
}

func (lv *LiteView) renderAct(act renderAction) {
	select {
	case lv.actionChan <- act:
		lv.Refresh()
	default:
	}
}
func (lv *LiteView) defaultKeyEvtHandler(evt *fyne.KeyEvent) {

}

func (lv *LiteView) SetKeyEvtHandler(f func(*fyne.KeyEvent)) {
	lv.keyEvtHandler = f
}

func (lv *LiteView) TypedKey(evt *fyne.KeyEvent) {
	switch evt.Name {
	case fyne.KeyDown:
		lv.renderAct(actScrollLineDown)
	case fyne.KeyUp:
		lv.renderAct(actScrollLineUp)
	case fyne.KeyRight, fyne.KeyPageDown, fyne.KeySpace:
		lv.renderAct(actScrollPageDown)
	case fyne.KeyLeft, fyne.KeyPageUp:
		lv.renderAct(actScrollPageUp)
	case fyne.KeyHome:
		lv.renderAct(actScrollTop)
	case fyne.KeyEnd:
		lv.renderAct(actScrollBottom)
	default:
		lv.keyEvtHandler(evt)
	}
}

func (lv *LiteView) TypedRune(c rune) {
}
func (lv *LiteView) FocusGained() {

}
func (lv *LiteView) FocusLost() {

}

func (lv *LiteView) Focused() bool {
	return true
}
func (lv *LiteView) SetUnderLine(underline UnderLineMode) {
	atomic.StoreUint32(lv.underLine, uint32(underline))
	lv.renderAct(actSetUnderline)
}

func (lv *LiteView) GetUnderLine() UnderLineMode {
	return UnderLineMode(atomic.LoadUint32(lv.underLine))
}

// func (lv *LiteView) Dragged(evt *fyne.DragEvent) {
// 	log.Printf("point is %v, x is %d, y is %d", evt.PointEvent.Position, evt.DraggedX, evt.DraggedY)
// }
// func (lv *LiteView) DragEnd() {
// 	log.Printf("drag is ended")
// }

// JumpTo display the text starting from line lineid and postion within line specified by linepos;
// if certralview is true, the specified postion is postioned verifically centrally,
// otherwise the specified postion starts from top of viewarea;
// if the specified postion is invalid, do nothing
func (lv *LiteView) JumpTo(lineid, linepos int, centralview bool) {
	if lineid < 0 || lineid >= len(lv.Val()) {
		return
	}
	if linepos < 0 {
		return
	} else {
		if len(lv.Val()[lineid]) != 0 {
			if linepos >= len(lv.Val()[lineid]) {
				return
			}
		}
	}
	lv.valMux.Lock()
	lv.StartLine = lineid
	lv.StartLinePos = linepos
	lv.valMux.Unlock()
	if centralview {
		lv.renderAct(actScrollToPoSCental)
	} else {
		lv.renderAct(actScrollToPoS)
	}
}

func (lv *LiteView) Scrolled(evt *fyne.ScrollEvent) {
	if evt.Scrolled.DY < 0 {
		lv.renderAct(actScrollLineDown)
	} else {
		lv.renderAct(actScrollLineUp)
	}
}

// GetPos returns current starting rune line and rune line pos
func (lv *LiteView) GetPos() (int, int) {
	lv.valMux.RLock()
	defer lv.valMux.RUnlock()
	return lv.StartLine, lv.StartLinePos
}

func (lv *LiteView) SetPosEvtHandler(f func(int, int)) {
	lv.posEvtHandler = f
}

func (lv *LiteView) MouseIn(evt *desktop.MouseEvent) {
	lv.parent.Canvas().Focus(lv)
}

func (lv *LiteView) MouseMoved(evt *desktop.MouseEvent) {
}

func (lv *LiteView) MouseOut() {

}
func (lv *LiteView) GetVal() [][]byte {
	lv.valMux.RLock()
	defer lv.valMux.RUnlock()
	return lv.lineList
}
