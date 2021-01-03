// look_test
package conf

import (
	"encoding/json"

	// "io/ioutil"
	"testing"
)

func TestLook(t *testing.T) {
	lk, err := defaultLook("sarasa-term-slab-sc-regular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	buf, err := json.MarshalIndent(lk, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", string(buf))
	newlk := new(Look)
	if err := json.Unmarshal(buf, newlk); err != nil {
		t.Fatal(err)
	}
	t.Logf("new look is:\n%v", newlk)

}
