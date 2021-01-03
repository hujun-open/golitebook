// char
package char

import (
	"bytes"

	"github.com/gogs/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	// "golang.org/x/text/encoding/unicode"
)

type DetChar struct {
	*chardet.Detector
}

func NewDetChar() *DetChar {
	return &DetChar{
		Detector: chardet.NewTextDetector(),
	}
}

const maxBuftoDet = 100

func (d *DetChar) ToByteList(buf []byte) ([][]byte, error) {
	var e encoding.Encoding
	newbuf := buf
	if len(buf) > maxBuftoDet {
		newbuf = buf[:maxBuftoDet]
	}
	charsetname, err := d.DetectBest(newbuf)
	isUTF8 := false
	if err != nil {
		isUTF8 = true
	} else {

		switch charsetname.Charset {
		case "GB2312", "GBK", "GB18030":
			e = simplifiedchinese.GB18030
		case "Big5":
			e = traditionalchinese.Big5
		default:
			isUTF8 = true
		}
	}
	uft8buf := buf
	if !isUTF8 {
		uft8buf, err = e.NewDecoder().Bytes(buf)
		if err != nil {
			return nil, err
		}
	}
	return SplitLinesBytes(uft8buf), nil
}

func SplitLinesBytes(buf []byte) [][]byte {
	s := bytes.ReplaceAll(buf, []byte("\r\n"), []byte("\n")) // convert dos to unix line ending
	return bytes.Split(s, []byte("\n"))
}
