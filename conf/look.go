// look
package conf

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/hujun-open/golitebook/liteview"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
)

func GetExecDir() string {
	return filepath.Dir(os.Args[0])
}

func fontDirList() []string {
	return []string{
		GetExecDir(),
		ConfDir(),
	}
}

const defaultFontName = "default.ttf"

func loadFontViaURI(url fyne.URI) (fyne.Resource, error) {
	return storage.LoadResourceFromURI(url)
}

func loadFontViaName(fname string) (fyne.Resource, string, error) {
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
		return nil, "", fmt.Errorf("failed to load font file %v", fname)
	}
	return res, fontpath, nil
}

func loadDefaultFont() (fyne.Resource, string, error) {
	return loadFontViaName(defaultFontName)
}

type Look struct {
	baseTheme fyne.Theme
	fontSize  float32
	font      fyne.Resource
	fontPath  string
	underLine int
}

type lookConf struct {
	FontPath  string
	FontSize  float32
	UnderLine int
}

func (lk Look) MarshalJSON() ([]byte, error) {
	lkcnf := lookConf{
		FontPath:  lk.fontPath,
		FontSize:  lk.fontSize,
		UnderLine: lk.underLine,
	}
	return json.Marshal(lkcnf)
}

func (lk *Look) UnmarshalJSON(b []byte) error {
	var lkcnf lookConf
	if err := json.Unmarshal(b, &lkcnf); err != nil {
		return err
	}
	var err error
	lkcnf.FontPath = strings.TrimSpace(lkcnf.FontPath)
	lk.font, err = loadFontViaURI(storage.NewURI(lkcnf.FontPath))
	if err != nil {
		lk.font, lk.fontPath, err = loadDefaultFont()
		if err != nil {
			return fmt.Errorf("faild to load font file and default font file")
		}
	}
	lk.fontPath = lkcnf.FontPath
	lk.fontSize = lkcnf.FontSize
	lk.underLine = lkcnf.UnderLine
	return nil

}

func NewLookFromTheme(t fyne.Theme) *Look {
	return &Look{
		baseTheme: t,
		fontSize:  t.Size(theme.SizeNameText),
		font:      t.Font(fyne.TextStyle{}),
	}
}

func DefaultLook() (*Look, error) {
	return defaultLook()
}

func defaultLook() (*Look, error) {
	l := NewLookFromTheme(theme.DarkTheme())
	l.underLine = int(liteview.UnderLineDash)
	var err error
	l.font, l.fontPath, err = loadDefaultFont()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Look) TextSize() float32 {
	return l.fontSize
}
func (l *Look) SetTextSize(s float32) {
	l.fontSize = s
}

// following are methods implmenting fyne.Theme interface
func (l *Look) Color(cname fyne.ThemeColorName, tvar fyne.ThemeVariant) color.Color {
	return l.baseTheme.Color(cname, tvar)
}
func (l *Look) Font(fyne.TextStyle) fyne.Resource {
	return l.font
}
func (l *Look) Icon(iconName fyne.ThemeIconName) fyne.Resource {
	return l.baseTheme.Icon(iconName)
}
func (l *Look) Size(sName fyne.ThemeSizeName) float32 {
	if sName == theme.SizeNameText {
		return l.fontSize
	}
	return l.baseTheme.Size(sName)
}

// end of interface

func (l *Look) SetUnderline(u int) {
	l.underLine = u
}

func (l *Look) GetUnderline() int {
	return l.underLine
}

func (l *Look) SetFont(r fyne.Resource, fpath string) {
	l.font = r
	l.fontPath = fpath
}
