// char_test
package char

import (
	"io/ioutil"
	"testing"
)

func TestDetChar(t *testing.T) {
	d := NewDetChar()
	buf, err := ioutil.ReadFile("gb.txt")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(d.ToRune(buf[:100])))
}
