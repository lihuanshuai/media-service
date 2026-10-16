// Command genicons 生成 MediaServiceTray 使用的托盘图标（assets/*.ico）。
// 用法：go run ./tools/genicons
//
// 图标为 32 位 BGRA（含 Alpha），纯标准库实现，无外部依赖。
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
)

// supersample 抗锯齿超采样倍数。
const supersample = 4

var outSizes = []int{16, 20, 24, 32, 48, 64}

type variant struct {
	file  string
	base  color.RGBA // 图标主体颜色
	edge  color.RGBA // 外圈描边颜色
	glyph string     // play / square / ring
}

var glyphColor = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

var variants = []variant{
	{file: "icon_on.ico", base: color.RGBA{0x27, 0xAE, 0x60, 0xFF}, edge: color.RGBA{0x12, 0x4E, 0x2C, 0xFF}, glyph: "play"},
	{file: "icon_off.ico", base: color.RGBA{0x7F, 0x8C, 0x8D, 0xFF}, edge: color.RGBA{0x3A, 0x41, 0x42, 0xFF}, glyph: "square"},
	{file: "icon_unknown.ico", base: color.RGBA{0xF3, 0x9C, 0x12, 0xFF}, edge: color.RGBA{0x7E, 0x4F, 0x02, 0xFF}, glyph: "ring"},
}

func main() {
	dir := "assets"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	for _, v := range variants {
		imgs := make([]icoImage, 0, len(outSizes))
		for _, size := range outSizes {
			imgs = append(imgs, icoImage{w: size, h: size, m: render(size, v)})
		}
		data := encodeICO(imgs)
		path := filepath.Join(dir, v.file)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			panic(err)
		}
		fmt.Printf("生成 %s（%d 字节）\n", path, len(data))
	}
}

// render 以超采样方式绘制单个尺寸的图标。
func render(size int, v variant) *image.RGBA {
	big := size * supersample
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	cx := float64(big-1) / 2
	cy := cx
	R := float64(big)/2 - 0.5 // 圆盘半径（含边缘描边）

	type acc struct{ r, g, b, a float64 }
	grid := make([]acc, size*size)

	for y := 0; y < big; y++ {
		for x := 0; x < big; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			d := math.Hypot(dx, dy)
			if d > R {
				continue
			}
			c := v.base
			if d > R-float64(supersample) {
				c = v.edge
			}
			if inGlyph(v.glyph, dx, dy, R) {
				c = glyphColor
			}
			i := (y/supersample)*size + (x / supersample)
			grid[i].r += float64(c.R)
			grid[i].g += float64(c.G)
			grid[i].b += float64(c.B)
			grid[i].a++
		}
	}

	cells := float64(supersample * supersample)
	for i := range grid {
		if grid[i].a == 0 {
			continue
		}
		img.SetRGBA(i%size, i/size, color.RGBA{
			R: uint8(math.Round(grid[i].r / grid[i].a)),
			G: uint8(math.Round(grid[i].g / grid[i].a)),
			B: uint8(math.Round(grid[i].b / grid[i].a)),
			A: uint8(math.Round(255 * grid[i].a / cells)),
		})
	}
	return img
}

func inGlyph(glyph string, dx, dy, R float64) bool {
	switch glyph {
	case "play":
		// 向右的播放三角形
		g := R * 0.5
		return inTriangle(dx, dy,
			-0.85*g, -g,
			-0.85*g, g,
			0.95*g, 0)
	case "square":
		// 圆角方块（停止）
		g := R * 0.5
		return inRoundedSquare(dx, dy, 0.62*g, 0.16*g)
	case "ring":
		// 圆环（加载中 / 未知）
		d := math.Hypot(dx, dy)
		return math.Abs(d-0.38*R) <= 0.12*R
	default:
		return false
	}
}

// inTriangle 判断点是否落在三角形内（含边）。
func inTriangle(px, py, ax, ay, bx, by, cx, cy float64) bool {
	hasNeg := false
	hasPos := false
	for _, t := range [][6]float64{
		{ax, ay, bx, by, px, py},
		{bx, by, cx, cy, px, py},
		{cx, cy, ax, ay, px, py},
	} {
		s := (t[4]-t[2])*(t[1]-t[3]) - (t[0]-t[2])*(t[5]-t[3])
		if s < 0 {
			hasNeg = true
		} else if s > 0 {
			hasPos = true
		}
	}
	return !(hasNeg && hasPos)
}

// inRoundedSquare 判断点是否落在圆角矩形内。
func inRoundedSquare(dx, dy, half, rad float64) bool {
	qx := math.Abs(dx) - (half - rad)
	qy := math.Abs(dy) - (half - rad)
	out := math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - rad
	return out <= 0
}

type icoImage struct {
	w, h int
	m    *image.RGBA
}

// encodeICO 将多张 RGBA 图像编码为 32 位（BGRA + Alpha）ICO 文件。
func encodeICO(imgs []icoImage) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // type: icon
	binary.Write(&buf, binary.LittleEndian, uint16(len(imgs)))

	blobs := make([][]byte, len(imgs))
	for i, im := range imgs {
		blobs[i] = encodeBMP(im.m)
	}

	offset := 6 + 16*len(imgs)
	for i, im := range imgs {
		buf.WriteByte(byte(im.w))
		buf.WriteByte(byte(im.h))
		buf.WriteByte(0)                                    // color count
		buf.WriteByte(0)                                    // reserved
		binary.Write(&buf, binary.LittleEndian, uint16(1))  // planes
		binary.Write(&buf, binary.LittleEndian, uint16(32)) // bit count
		binary.Write(&buf, binary.LittleEndian, uint32(len(blobs[i])))
		binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(blobs[i])
	}
	for _, b := range blobs {
		buf.Write(b)
	}
	return buf.Bytes()
}

// encodeBMP 将 RGBA 图像编码为 ICO 内的 32 位 DIB（bottom-up BGRA + AND mask）。
func encodeBMP(m *image.RGBA) []byte {
	w := m.Bounds().Dx()
	h := m.Bounds().Dy()
	xorSize := w * h * 4
	andSize := ((w + 31) / 32) * 4 * h

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(40)) // BITMAPINFOHEADER size
	binary.Write(&buf, binary.LittleEndian, int32(w))
	binary.Write(&buf, binary.LittleEndian, int32(h*2)) // XOR + AND
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(32))
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // BI_RGB
	binary.Write(&buf, binary.LittleEndian, uint32(xorSize+andSize))
	binary.Write(&buf, binary.LittleEndian, int32(0))  // X pixels/meter
	binary.Write(&buf, binary.LittleEndian, int32(0))  // Y pixels/meter
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // color used
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // important colors

	row := make([]byte, 0, w*4)
	for y := h - 1; y >= 0; y-- {
		row = row[:0]
		for x := 0; x < w; x++ {
			c := m.RGBAAt(x, y)
			row = append(row, c.B, c.G, c.R, c.A)
		}
		buf.Write(row)
	}
	buf.Write(make([]byte, andSize)) // AND mask：全 0（透明由 Alpha 通道表达）
	return buf.Bytes()
}
