// plugin
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hujun-open/golitebook/api"
	"github.com/hujun-open/golitebook/char"
	"github.com/hujun-open/golitebook/conf"

	// "fyne.io/fyne/dialog"

	pptypes "github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

const SubsFileName = "golitebook.subs"

func GetSubsSavePath() string {
	return filepath.Join(GetLocalSavePath(), SubsFileName)
}

func GetLocalSavePath() string {
	return filepath.Join(conf.ConfDir(), "savedbook")
}
func GetLocalSavedFilePath(bookname string) string {
	return filepath.Join(GetLocalSavePath(), bookname+".txt")
}

const defaultStartingPort = 30000

type SearchResult struct {
	PluginName  string
	BookName    string
	BookPageURL string
	BookSize    string
	AuthorName  string
	Status      string
	LastUpdate  time.Time
}

type SearchResultList []*SearchResult

// Len return total number of items
func (srl SearchResultList) Len() int {
	return len(srl)
}

// Fields return the field names
func (srl SearchResultList) Fields() []string {
	return []string{"书名", "URL", "大小", "作者", "状态", "最后更新"}
}

// Item return item with index i, as a slice of strings, each string represents a field's value
func (srl SearchResultList) Item(id int) []string {
	if id < 0 || id >= srl.Len() {
		return nil
	}
	return []string{
		srl[id].BookName,
		srl[id].BookPageURL,
		srl[id].BookSize,
		srl[id].AuthorName,
		srl[id].Status,
		srl[id].LastUpdate.Format("2006-01-02"),
	}
}
func (srl SearchResultList) Sort(field int, ascend bool) {
	//TODO
}

func (srl SearchResultList) Filter(kw string, i int) {
}

type Plugin struct {
	Name   string
	Cmd    *exec.Cmd
	Port   int
	Client api.GoLitebookPluginClient
}

const (
	rpcTimeout      = 10 * time.Second
	downloadTimeout = 30 * time.Minute
)

func NewPlugin(p string, port int) (*Plugin, error) {
	logfpath := filepath.Join(conf.ConfDir(), filepath.Base(p)+".log")
	cmd := exec.Command(p, "-p", fmt.Sprintf("%d", port), "-logf", logfpath)
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Second)
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", port), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to create API connection to plugin, %v", err)
	}
	return &Plugin{Cmd: cmd, Port: port, Name: filepath.Base(p), Client: api.NewGoLitebookPluginClient(conn)}, nil

}

func (p *Plugin) keepalive() {
	stream, err := p.Client.Keepalive(context.Background())
	if err != nil {
		return
	}
	req := new(api.Empty)
	for {
		err = stream.Send(req)
		if err != nil {
			return
		}
		time.Sleep(10 * time.Second)
	}
}

func (p *Plugin) GetDesc() (string, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(rpcTimeout))
	resp, err := p.Client.GetDesc(ctx, &api.Empty{})
	if err != nil {
		return "", err
	}
	return resp.Desc, nil
}

func (p *Plugin) SearchBook(kw string) (SearchResultList, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(rpcTimeout))
	resp, err := p.Client.Search(ctx, &api.SearchReq{Keyword: kw})
	if err != nil {
		return nil, err
	}
	var rlist SearchResultList
	for _, r := range resp.GetResultList() {
		result := &SearchResult{
			PluginName:  p.Name,
			BookName:    r.BookName,
			BookPageURL: r.BookPageURL,
			BookSize:    r.BookSize,
			AuthorName:  r.AuthorName,
			Status:      r.Status,
		}
		result.LastUpdate, _ = pptypes.Timestamp(r.LastUpdate)
		rlist = append(rlist, result)
	}
	return rlist, nil
}

// GetBookInfo returns total chapter count and name of last chapter
func (p *Plugin) GetBookInfo(bookurl string) (total int, lastch, indexurl string, err error) {
	req := new(api.GetBookInfoReq)
	req.BookPageURL = bookurl
	var resp *api.GetBookInfoResp
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(rpcTimeout))
	resp, err = p.Client.GetBookInfo(ctx, req)
	if err != nil {
		return
	}
	total = int(resp.TotalChapterCount)
	lastch = resp.LastChapterName
	indexurl = resp.BookIndexURL
	return
}

type PluginList map[string]*Plugin

// load plugins from dir p
func newPluginList(p string) (PluginList, error) {
	list := make(map[string]*Plugin)
	fileList, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, err
	}
	nextPort := defaultStartingPort
	for _, f := range fileList {
		fullpath := filepath.Join(p, f.Name())
		p, err := NewPlugin(fullpath, nextPort)
		if err != nil {
			return nil, err
		}
		list[p.Name] = p
		nextPort++
	}
	return list, nil
}

func (list PluginList) NameList() []string {
	r := []string{}
	for _, p := range list {
		r = append(r, p.Name)
	}
	return r
}

func (list PluginList) KillAll() {
	for _, p := range list {
		p.Cmd.Process.Kill()
	}
}

func getPluginDirectory() string {
	return filepath.Join(conf.ConfDir(), "plugins")
}

var LoadedPlugins PluginList

func init() {
	LoadedPlugins, _ = newPluginList(getPluginDirectory())
	CurrentSubscriptions = NewSubscriptionList()
	CurrentSubscriptions.Load()
}

const (
	DownloadResultNotStarted uint32 = iota
	DownloadResultWorking
	DownloadResultFailed
	DownloadResultFinished
)

// key is the bookname
type SubscriptionList struct {
	list []*Subscription
	mux  *sync.RWMutex
}

func NewSubscriptionList() *SubscriptionList {
	return &SubscriptionList{
		list: []*Subscription{},
		mux:  new(sync.RWMutex),
	}
}

var CurrentSubscriptions *SubscriptionList

func (slist *SubscriptionList) Len() int {
	return len(slist.list)
}
func (slist *SubscriptionList) Fields() []string {
	return []string{
		"书名",
		"状态",
		"URL",
		"最新章节",
		"已有章节数",
		"最后下载时间",
	}
}
func (slist *SubscriptionList) Item(id int) []string {
	slist.mux.RLock()
	defer slist.mux.RUnlock()
	slist.list[id].mux.RLock()
	defer slist.list[id].mux.RUnlock()
	return []string{
		slist.list[id].bookName,
		slist.list[id].statusTxt,
		slist.list[id].bookURL,
		slist.list[id].lastChapterName,
		fmt.Sprintf("%d", slist.list[id].totalChapter),
		slist.list[id].lastDownloadTime.Format("2006-01-02 15:04:05"),
	}
}

func (slist *SubscriptionList) Sort(field int, ascend bool) {
}

func (slist *SubscriptionList) Filter(kw string, i int) {
}

func (slist *SubscriptionList) Update(id int, newval *Subscription) {
	slist.mux.Lock()
	defer slist.mux.Unlock()
	if id < 0 || id >= len(slist.list) {
		return
	}
	slist.list[id] = newval
}
func (slist *SubscriptionList) Remove(id int) {
	slist.mux.Lock()
	defer slist.mux.Unlock()
	if id < 0 || id >= len(slist.list) {
		return
	}
	if id < len(slist.list)-1 {
		copy(slist.list[id:], slist.list[id+1:])
		// slist.list[len(slist.list)-1] = nil
	}
	slist.list = slist.list[:len(slist.list)-1]
}

func (slist *SubscriptionList) Append(val *Subscription) {
	slist.mux.Lock()
	defer slist.mux.Unlock()
	slist.list = append(slist.list, val)
}

func (slist *SubscriptionList) Get() []*Subscription {
	slist.mux.RLock()
	defer slist.mux.RUnlock()
	r := []*Subscription{}
	for _, s := range slist.list {
		sub := s
		r = append(r, sub)
	}
	return r
}

func (slist *SubscriptionList) Save() error {
	os.MkdirAll(GetLocalSavePath(), 0755)
	buf, err := json.MarshalIndent(slist.list, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(GetSubsSavePath(), buf, 0644)
}

func (slist *SubscriptionList) Load() error {
	buf, err := ioutil.ReadFile(GetSubsSavePath())
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &slist.list)
	if err != nil {
		return err
	}
	for i, subs := range CurrentSubscriptions.Get() {
		subs.finished = DownloadResultNotStarted
		subs.statusTxt = ""
		CurrentSubscriptions.Update(i, subs)
		subs.mux = new(sync.RWMutex)
		subs.client = LoadedPlugins[subs.pluginName].Client
	}
	return nil
}

type Subscription struct {
	bookName string
	bookURL  string
	//StartingChapter start from 1, 0 means need to download whole book
	startingChapter int
	// TotalChapter get from getbookinfo, means total number chapter, only meaningful after Update() is called
	totalChapter int
	//get called after every chapter is downloaded
	progressHandler func(url string, finished, total int)
	pluginName      string
	client          api.GoLitebookPluginClient
	finished        uint32
	statusTxt       string
	mux             *sync.RWMutex
	//ResultStr only meanful when Update() is called
	resultByteList   [][]byte
	lastChapterName  string
	lastDownloadTime time.Time
}

type subscriptionJSONType struct {
	BookName         string
	BookURL          string
	StartingChapter  int
	TotalChapter     int
	PluginName       string
	LastChapterName  string
	LastDownloadTime time.Time
}

func (sub *Subscription) MarshalJSON() ([]byte, error) {
	output := subscriptionJSONType{
		BookName:         sub.bookName,
		BookURL:          sub.bookURL,
		StartingChapter:  sub.startingChapter,
		TotalChapter:     sub.totalChapter,
		PluginName:       sub.pluginName,
		LastChapterName:  sub.lastChapterName,
		LastDownloadTime: sub.lastDownloadTime,
	}
	return json.MarshalIndent(output, "", "  ")
}
func (sub *Subscription) UnmarshalJSON(buf []byte) error {
	out := new(subscriptionJSONType)
	err := json.Unmarshal(buf, out)
	if err != nil {
		return err
	}
	sub.bookName = out.BookName
	sub.bookURL = out.BookURL
	sub.startingChapter = out.StartingChapter
	sub.totalChapter = out.TotalChapter
	sub.pluginName = out.PluginName
	sub.lastChapterName = out.LastChapterName
	sub.lastDownloadTime = out.LastDownloadTime
	sub.finished = DownloadResultNotStarted
	sub.mux = new(sync.RWMutex)
	sub.client = LoadedPlugins[sub.pluginName].Client
	sub.statusTxt = ""
	return nil
}

// NewSubscription create a new subscription, download whole book, via book url
func NewSubscription(name, bookurl string, pluginName string, h func(string, int, int)) *Subscription {
	return &Subscription{
		bookName:         name,
		bookURL:          bookurl,
		startingChapter:  0,
		progressHandler:  h,
		pluginName:       pluginName,
		client:           LoadedPlugins[pluginName].Client,
		finished:         DownloadResultNotStarted,
		lastDownloadTime: time.Now(),
		mux:              new(sync.RWMutex),
	}
}
func (task *Subscription) SetHandler(h func(string, int, int)) {
	task.mux.Lock()
	defer task.mux.Unlock()
	task.progressHandler = h
}

func (task *Subscription) SetStatusTxt(t string) {
	task.mux.Lock()
	defer task.mux.Unlock()
	task.statusTxt = t
}

func (task *Subscription) Status() uint32 {
	task.mux.RLock()
	defer task.mux.RUnlock()
	return task.finished
}

// Update use sub.StartingChapter and getbookinfo() to update the book to the latest
func (sub *Subscription) Update() {
	failed := true
	defer func() {
		sub.mux.Lock()
		defer sub.mux.Unlock()
		if failed {
			sub.finished = DownloadResultFailed
		} else {
			sub.finished = DownloadResultFinished
			sub.lastDownloadTime = time.Now()
		}

	}()
	sub.mux.Lock()
	sub.finished = DownloadResultWorking

	sub.mux.Unlock()
	// get book info
	req := new(api.GetBookInfoReq)
	req.BookPageURL = sub.bookURL
	var resp *api.GetBookInfoResp
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(rpcTimeout))
	resp, err := sub.client.GetBookInfo(ctx, req)
	if err != nil {
		// dialog.ShowError(err, nil)
		log.Printf("failed to get book info, %v", err)
		return
	}
	sub.mux.Lock()
	sub.totalChapter = int(resp.TotalChapterCount)
	sub.lastChapterName = resp.LastChapterName
	handler := sub.progressHandler
	sub.mux.Unlock()
	if sub.totalChapter-sub.startingChapter == 0 {
		handler(sub.bookURL, 0, 0)
		failed = false
		return
	}
	getReq := new(api.GetBookReq)
	getReq.BookIndexURL = resp.BookIndexURL
	getReq.CurrentChaptCount = uint32(sub.startingChapter)
	ctx, _ = context.WithDeadline(context.Background(), time.Now().Add(downloadTimeout))
	stream, err := sub.client.GetBook(ctx, getReq)
	if err != nil {
		log.Printf("failed to get book stream, %v", err)
		// dialog.ShowError(fmt.Errorf("failed to download, %v", err), nil)
		return
	}
	resultList := make([]string, sub.totalChapter-sub.startingChapter)
	finished := 0
	for {
		handler(sub.bookURL, finished, sub.totalChapter-sub.startingChapter)
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// dialog.ShowError(fmt.Errorf("downloading error, %v", err), nil)
			log.Printf("downloading error, %v", err)
			return
		}
		resultList[int(resp.ChapterId)-sub.startingChapter] = "\n" + BookMarkChar + resp.ChapterName + "\n" + resp.ChapterContent
		finished++
	}

	filename := GetLocalSavedFilePath(sub.bookName)
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open file to save, %v", err)
		// dialog.ShowError(err, nil)
		return
	}
	defer f.Close()

	sub.mux.Lock()
	sub.resultByteList = char.SplitLinesBytes([]byte(strings.Join(resultList, "\n\n")))
	sub.resultByteList = FormatTxt(sub.resultByteList, DefaultFormatMinimalLineWidthInChars, nil)
	sub.mux.Unlock()
	if _, err := f.Write(bytes.Join(sub.resultByteList, []byte("\n"))); err != nil {
		log.Printf("failed to write save file, %v", err)
		// dialog.ShowError(err, nil)
		return
	}
	sub.mux.Lock()
	sub.startingChapter += finished
	sub.mux.Unlock()
}

const (
	// default value of minimal line length in Chars for FormatTxt()
	DefaultFormatMinimalLineWidthInChars = 500
	// max allowed number of newlines only line in a row for FormatTxt()
	MaxChainedNewlines = 0
	BookMarkChar       = "»"
	ParagraphPrefix    = ""
)

func strByteList(l [][]byte) string {
	r := "\n"
	for i, buf := range l {
		r += fmt.Sprintf("line %d: %v\n", i, string(buf))
	}
	return r
}

func FormatTxt(lineList [][]byte, linewidth int, h func(float64)) [][]byte {

	// remove emptyes lines in a row
	newlinesInRow := 0
	newLineList := [][]byte{}
	combinedLine := []byte(ParagraphPrefix) //two spaces leading a paragraph
	defer func() {
		if h != nil {
			h(1.0)
		}
	}()
	bookMarkRune := []rune(BookMarkChar)[0]
	for i := 0; i < len(lineList); i++ {
		// if i > 10 {
		// 	break
		// }
		// log.Printf("working on line %d, combine line is %v, newlinelist is %v", i, string(combinedLine), strByteList(newLineList))
		if h != nil {
			h(float64(i) / float64(len(lineList)))
		}
		trimedLine := bytes.TrimSpace(lineList[i])
		if len(trimedLine) == 0 {
			// log.Printf("line %d is empty, newlinesInRow is %d", i, newlinesInRow)
			newlinesInRow++
			if newlinesInRow <= MaxChainedNewlines {
				newLineList = append(newLineList, []byte{})
				// log.Printf("add a new line for line %d", i)
			}

		} else {
			// combine short lines
			if IsPrecreatedChapterTitle(lineList[i], bookMarkRune) {
				newLineList = append(newLineList, combinedLine)
				newLineList = append(newLineList, []byte{})
				newLineList = append(newLineList, trimedLine)
				newLineList = append(newLineList, []byte{})
				combinedLine = []byte(ParagraphPrefix)
				newlinesInRow = 0
				continue
			}
			combinedLine = append(combinedLine, trimedLine...)
			if len(combinedLine) > linewidth {
				newLineList = append(newLineList, combinedLine)
				combinedLine = []byte(ParagraphPrefix)
				newlinesInRow = 0
			}
		}
	}
	return newLineList
}

func IsPrecreatedChapterTitle(line []byte, bookMarkRune rune) bool {
	trimedLine := bytes.TrimLeft(line, " ")
	return bytes.IndexRune(trimedLine, bookMarkRune) == 0
}
