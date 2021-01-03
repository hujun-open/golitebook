// history
package history

import (
	"encoding/json"
	"golitebook/conf"
	"io/ioutil"
	"path/filepath"
)

func getHistoryFilePath() string {
	return filepath.Join(conf.ConfDir(), "history")
}

// ReadingHistory key is the base filename of book, value is the starting line
type ReadingHistory map[string]int

func (h ReadingHistory) Update(bookpath string, startline int) {
	h[bookpath] = startline
}
func (h ReadingHistory) GetStartLine(bookpath string) int {
	if line, ok := h[bookpath]; ok {
		return line
	}
	return 0
}

func (h ReadingHistory) Save() error {
	buf, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(getHistoryFilePath(), buf, 0644)
}

func (h ReadingHistory) Load() error {
	buf, err := ioutil.ReadFile(getHistoryFilePath())
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, &h)
}

var History ReadingHistory

func init() {
	History = make(ReadingHistory)
	History.Load()
}
