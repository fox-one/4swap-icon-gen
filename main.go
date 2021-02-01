package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"strings"

	"github.com/nfnt/resize"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// usage
// 	./4swap-icon-gen -c1 "#F7931B" -c2 "#000000" -ic1 ./btc.png -ic2 ./dot.png -t sBTC-DOT

var color1 = flag.String("c1", "#FF0000", "the first color")
var color2 = flag.String("c2", "#0000FF", "the second color")
var icon1 = flag.String("ic1", "icon1.png", "the first icon")
var icon2 = flag.String("ic2", "icon2.png", "the second icon")
var tokenName = flag.String("t", "sAAA-BBB", "the lp token name")

const (
	SIZE      = 520
	ICON_SIZE = 240
	GAP       = 40
)

func loadSVG(c1, c2 string) []byte {
	data, err := ioutil.ReadFile("tpl.svg")
	if err != nil {
		panic(err)
	}
	s := string(data)
	s = strings.ReplaceAll(s, "#DE400B", c1)
	s = strings.ReplaceAll(s, "#19C0FC", c2)
	return []byte(s)
}

func loadIcons() (image.Image, image.Image) {
	imgFile1, err := os.Open(*icon1)
	if err != nil {
		fmt.Println(err)
	}
	imgFile2, err := os.Open(*icon2)
	if err != nil {
		fmt.Println(err)
	}

	img1, _, err := image.Decode(imgFile1)
	if err != nil {
		fmt.Println(err)
	}
	img2, _, err := image.Decode(imgFile2)
	if err != nil {
		fmt.Println(err)
	}
	img1 = resize.Resize(ICON_SIZE, ICON_SIZE, img1, resize.Lanczos3)
	img2 = resize.Resize(ICON_SIZE, ICON_SIZE, img2, resize.Lanczos3)

	return img1, img2
}

func genPNG(svg []byte, img1, img2 image.Image) {

	// load from svg tpl
	icon, _ := oksvg.ReadIconStream(bytes.NewReader(svg))
	icon.SetTarget(0, 0, float64(SIZE), float64(SIZE))
	rgba := image.NewRGBA(image.Rect(0, 0, SIZE, SIZE))
	icon.Draw(rasterx.NewDasher(SIZE, SIZE, rasterx.NewScannerGV(SIZE, SIZE, rgba, rgba.Bounds())), 1)

	// draw icons
	img1Size := img1.Bounds().Size()
	r1 := image.Rectangle{image.Point{SIZE/2 - ICON_SIZE/2, 0}, image.Point{SIZE/2 - ICON_SIZE/2 + img1Size.X, img1Size.Y}}
	img2Size := img1.Bounds().Size()
	r2 := image.Rectangle{image.Point{SIZE/2 - ICON_SIZE/2, SIZE/2 + GAP/2}, image.Point{SIZE/2 - ICON_SIZE/2 + img2Size.X, SIZE/2 + GAP/2 + img2Size.Y}}

	draw.Draw(rgba, r1, img1, image.Point{0, 0}, draw.Over)
	draw.Draw(rgba, r2, img2, image.Point{0, 0}, draw.Over)

	// write png
	out, err := os.Create(fmt.Sprintf("%s.png", *tokenName))
	if err != nil {
		panic(err)
	}
	defer out.Close()

	err = png.Encode(out, rgba)
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	base := loadSVG(*color1, *color2)
	img1, img2 := loadIcons()
	genPNG(base, img1, img2)
}
