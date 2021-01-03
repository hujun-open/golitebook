// dashline
package liteview

import (
	"image"
	"image/color"
)

type DashedLine struct {
	*image.RGBA
	height, overallWidth    int
	dashWidth, dashInterval int
	lineColor               color.Color
}

func (line *DashedLine) drawDash(startx int) {
	for x := startx; x < startx+line.dashWidth; x++ {
		for y := 0; y < line.height; y++ {
			if x <= line.overallWidth && y <= line.height {
				line.Set(x, y, line.lineColor)
			}
		}
	}
}

func NewDashedLine(w, h int, dashW, dashInt int, c color.Color) (line *DashedLine) {
	line = new(DashedLine)
	line.overallWidth = w
	line.height = h
	line.RGBA = image.NewRGBA(image.Rect(0, 0, w, h))
	line.lineColor = c
	line.dashWidth = dashW
	line.dashInterval = dashInt
	for x := 0; x < w; x += line.dashWidth + line.dashInterval {
		line.drawDash(x)
	}
	return line
}
