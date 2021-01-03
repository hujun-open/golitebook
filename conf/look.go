// look
package conf

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"golitebook/liteview"

	"fyne.io/fyne"
	"fyne.io/fyne/storage"
	"fyne.io/fyne/theme"
)

func GetExecDir() string {
	return filepath.Dir(os.Args[0])
}

func fontDirList() []string {
	return []string{
		"",
		ConfDir(),
	}
}

const defaultFontName = "default.ttf"

func loadFontViaURI(url fyne.URI) (fyne.Resource, error) {
	return storage.LoadResourceFromURI(url)
}

func loadFontViaName(fname string) (fyne.Resource, error) {
	loaded := false
	var err error
	var res fyne.Resource
	var fontpath string
	for _, dir := range fontDirList() {
		fontpath = filepath.Join(dir, fname)
		res, err = fyne.LoadResourceFromPath(fontpath)
		if err == nil {
			loaded = true
			break
		}
	}
	if !loaded {
		return nil, fmt.Errorf("failed to load font file %v", fname)
	}
	return res, nil
}

func loadDefaultFont() (fyne.Resource, error) {
	return loadFontViaName(defaultFontName)
}

type Look struct {
	background                                                                    color.Color
	button, disabledButton, text, placeholder, hover, shadow, disabled, scrollBar color.Color
	regularFont                                                                   fyne.Resource
	fontSize, paddingSize, iconInLineSize, scrollBarSize, scrollBarSmallSize      int
	underLine                                                                     int
	RegularFontPath                                                               string
}

type lookConf struct {
	Button                                                                            color.Alpha16
	Background, DisabledButton, Text, Placeholder, Hover, Shadow, Disabled, ScrollBar color.NRGBA
	RegularFontPath                                                                   string
	FontSize, PaddingSize, IconInLineSize, ScrollBarSize, ScrollBarSmallSize          int
	UnderLine                                                                         int
}

func (lk Look) MarshalJSON() ([]byte, error) {
	lkcnf := lookConf{
		Background:         lk.background.(color.NRGBA),
		Button:             lk.button.(color.Alpha16),
		DisabledButton:     lk.disabledButton.(color.NRGBA),
		Text:               lk.text.(color.NRGBA),
		Placeholder:        lk.placeholder.(color.NRGBA),
		Hover:              lk.hover.(color.NRGBA),
		Shadow:             lk.shadow.(color.NRGBA),
		Disabled:           lk.disabled.(color.NRGBA),
		ScrollBar:          lk.scrollBar.(color.NRGBA),
		RegularFontPath:    lk.RegularFontPath,
		FontSize:           lk.fontSize,
		PaddingSize:        lk.paddingSize,
		IconInLineSize:     lk.iconInLineSize,
		ScrollBarSize:      lk.scrollBarSize,
		ScrollBarSmallSize: lk.scrollBarSmallSize,
		UnderLine:          lk.underLine,
	}
	return json.Marshal(lkcnf)
}

func (lk *Look) UnmarshalJSON(b []byte) error {
	var lkcnf lookConf
	if err := json.Unmarshal(b, &lkcnf); err != nil {
		return err
	}
	var err error
	lkcnf.RegularFontPath = strings.TrimSpace(lkcnf.RegularFontPath)
	lk.regularFont, err = loadFontViaURI(storage.NewURI(lkcnf.RegularFontPath))
	if err != nil {
		lk.regularFont, err = loadDefaultFont()
		if err != nil {
			return fmt.Errorf("faild to load font file and default font file")
		}
	}
	lk.RegularFontPath = lkcnf.RegularFontPath

	lk.button = lkcnf.Button
	lk.background = lkcnf.Background
	lk.disabledButton = lkcnf.DisabledButton
	lk.text = lkcnf.Text
	lk.placeholder = lkcnf.Placeholder
	lk.hover = lkcnf.Hover
	lk.shadow = lkcnf.Shadow
	lk.disabled = lkcnf.Disabled
	lk.scrollBar = lkcnf.ScrollBar
	lk.fontSize = lkcnf.FontSize
	lk.paddingSize = lkcnf.PaddingSize
	lk.iconInLineSize = lkcnf.IconInLineSize
	lk.scrollBarSize = lkcnf.ScrollBarSize
	lk.scrollBarSmallSize = lkcnf.ScrollBarSmallSize
	lk.underLine = lkcnf.UnderLine
	return nil

}
func NewLookFromTheme(t fyne.Theme) *Look {
	return &Look{
		background:         t.BackgroundColor(),
		button:             t.ButtonColor(),
		disabledButton:     t.DisabledButtonColor(),
		text:               t.TextColor(),
		placeholder:        t.PlaceHolderColor(),
		hover:              t.HoverColor(),
		shadow:             t.ShadowColor(),
		disabled:           t.DisabledIconColor(),
		scrollBar:          t.ScrollBarColor(),
		regularFont:        t.TextFont(),
		fontSize:           t.TextSize(),
		paddingSize:        t.Padding(),
		iconInLineSize:     t.IconInlineSize(),
		scrollBarSize:      t.ScrollBarSize(),
		scrollBarSmallSize: t.ScrollBarSmallSize(),
	}
}

func DefaultLook() (*Look, error) {
	return defaultLook()
}

func defaultLook() (*Look, error) {
	l := NewLookFromTheme(theme.DarkTheme())
	l.disabledButton = l.disabled
	l.underLine = int(liteview.UnderLineDash)
	var err error
	l.regularFont, err = loadDefaultFont()
	if err != nil {
		return nil, err
	}
	return l, nil
}
func (l *Look) SetTextSize(s int) {
	l.fontSize = s
}

// following are methods implmenting fyne.Theme interface
func (l *Look) BackgroundColor() color.Color {
	return l.background
}
func (l *Look) ButtonColor() color.Color {
	return l.button
}
func (l *Look) DisabledButtonColor() color.Color {
	return l.disabledButton
}
func (l *Look) HyperlinkColor() color.Color {
	return l.PrimaryColor()
}
func (l *Look) TextColor() color.Color {
	return l.text
}
func (l *Look) DisabledTextColor() color.Color {
	return l.disabled
}
func (l *Look) IconColor() color.Color {
	return l.text
}
func (l *Look) DisabledIconColor() color.Color {
	return l.disabled
}
func (l *Look) PlaceHolderColor() color.Color {
	return l.placeholder
}
func (l *Look) PrimaryColor() color.Color {
	return theme.PrimaryColorNamed(fyne.CurrentApp().Settings().PrimaryColor())
}
func (l *Look) HoverColor() color.Color {
	return l.hover
}
func (l *Look) FocusColor() color.Color {
	return l.PrimaryColor()
}
func (l *Look) ScrollBarColor() color.Color {
	return l.scrollBar
}
func (l *Look) ShadowColor() color.Color {
	return l.shadow
}
func (l *Look) TextSize() int {
	return l.fontSize
}
func (l *Look) TextFont() fyne.Resource {
	return l.regularFont
}
func (l *Look) TextBoldFont() fyne.Resource {
	return l.regularFont
}
func (l *Look) TextItalicFont() fyne.Resource {
	return l.regularFont
}
func (l *Look) TextBoldItalicFont() fyne.Resource {
	return l.regularFont
}
func (l *Look) TextMonospaceFont() fyne.Resource {
	return l.regularFont
}

func (l *Look) Padding() int {
	return l.paddingSize
}
func (l *Look) IconInlineSize() int {
	return l.paddingSize
}
func (l *Look) ScrollBarSize() int {
	return l.paddingSize
}
func (l *Look) ScrollBarSmallSize() int {
	return l.paddingSize
}

func (l *Look) SetUnderline(u int) {
	l.underLine = u
}

func (l *Look) GetUnderline() int {
	return l.underLine
}

func (l *Look) SetFont(r fyne.Resource) {
	l.regularFont = r
}
