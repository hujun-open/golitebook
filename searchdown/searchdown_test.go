// searchdown_test
package searchdown

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestLoadSubs(t *testing.T) {
	CurrentSubs = []*Subscription{
		&Subscription{
			BookName:          "休息-1",
			BookIndexURL:      "url-1",
			LastChapterName:   "book-1-chap",
			TotalChapterCount: 10,
			Status:            new(uint32),
			LastDownload:      time.Now(),
			LocalPath:         GetLocalSavePath(),
		},
		&Subscription{
			BookName:          "修仙-2",
			BookIndexURL:      "url-2",
			LastChapterName:   "book-2-chap",
			TotalChapterCount: 20,
			Status:            new(uint32),
			LastDownload:      time.Now().Add(time.Hour),
			LocalPath:         GetLocalSavePath(),
		},
	}
	atomic.StoreUint32(CurrentSubs[0].Status, StatusDownloading)
	atomic.StoreUint32(CurrentSubs[1].Status, StatusReady)
	err := saveSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("now loading")
	err = loadSusbcriptionsFromFile()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range CurrentSubs {
		t.Logf("%+v", s)
	}
}
