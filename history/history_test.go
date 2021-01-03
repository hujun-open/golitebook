// history_test
package history

import (
	"testing"
)

func TestHistory(t *testing.T) {
	History.Update("book1", 10)
	History.Update("book2", 10)
	err := History.Save()
	if err != nil {
		t.Fatal(err)
	}
}
