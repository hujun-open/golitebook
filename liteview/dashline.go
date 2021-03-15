// dashline
package liteview

import (
	"image"
	"image/color"
)

type DashedLine struct {
	*image.RGBA
	height, overallWidth    float32
	dashWidth, dashInterval float32
	lineColor               color.Color
}

func (line *DashedLine) drawDash(startx float32) {
	for x := startx; x < startx+line.dashWidth; x++ {
		for y := float32(0); y < line.height; y++ {
			if x <= line.overallWidth && y <= line.height {
				line.Set(int(x), int(y), line.lineColor)
			}
		}
	}
}

func NewDashedLine(w, h float32, dashW, dashInt float32, c color.Color) (line *DashedLine) {
	line = new(DashedLine)
	line.overallWidth = w
	line.height = h
	line.RGBA = image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	line.lineColor = c
	line.dashWidth = dashW
	line.dashInterval = dashInt
	for x := float32(0); x < w; x += line.dashWidth + line.dashInterval {
		line.drawDash(x)
	}
	return line
}
