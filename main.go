package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/fox-one/mixin-sdk-go"
	color_extractor "github.com/marekm4/color-extractor"
	"github.com/nfnt/resize"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

var (
	config = flag.String("config", "", "keystore file path")
	pin    = flag.String("pin", "", "pin")

	ctx = context.Background()

	outputPath = flag.String("o", "", "the output path")
	newAssetID = flag.String("a0", "", "the output asset id")
	assetID1   = flag.String("a1", "", "the first asset id")
	assetID2   = flag.String("a2", "", "the second asset id")
	color1     = flag.String("c1", "", "the first color (in hex)")
	color2     = flag.String("c2", "", "the second color (in hex)")
	icon1      = flag.String("ic1", "", "the first icon")
	icon2      = flag.String("ic2", "", "the second icon")
)

const (
	SIZE        = 520
	ICON_SIZE   = 240
	GAP         = 40
	OUTPUT_PATH = "output"
)

func hexColor(c color.Color) string {
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
}

func getPath() string {
	base := OUTPUT_PATH
	if *outputPath != "" {
		base = *outputPath
	}
	return path.Join(base, *newAssetID)
}

func loadSVG(c1, c2 string) []byte {
	data, err := ioutil.ReadFile("tpl.svg")
	if err != nil {
		panic(err)
	}
	s := string(data)

	if *color1 != "" {
		c1 = *color1
	}
	if *color2 != "" {
		c2 = *color2
	}
	s = strings.ReplaceAll(s, "#DE400B", c1)
	s = strings.ReplaceAll(s, "#19C0FC", c2)
	return []byte(s)
}

func loadIcons(url1, url2 string) (img1, img2 image.Image, c1, c2 string, err error) {
	resp1, err := http.Get(url1)
	if err != nil {
		return
	}
	defer resp1.Body.Close()

	resp2, err := http.Get(url2)
	if err != nil {
		return
	}
	defer resp2.Body.Close()

	var imgFile1, imgFile2 io.Reader
	if *icon1 != "" {
		imgFile1, err = os.Open(*icon1)
		if err != nil {
			return
		}
	} else {
		imgFile1 = resp1.Body
	}

	if *icon2 != "" {
		imgFile2, err = os.Open(*icon2)
		if err != nil {
			return
		}
	} else {
		imgFile2 = resp2.Body
	}

	img1, _, err = image.Decode(imgFile1)
	if err != nil {
		return
	}
	img2, _, err = image.Decode(imgFile2)
	if err != nil {
		return
	}
	img1 = resize.Resize(ICON_SIZE, ICON_SIZE, img1, resize.Bilinear)
	img2 = resize.Resize(ICON_SIZE, ICON_SIZE, img2, resize.Bilinear)

	colors1 := color_extractor.ExtractColors(img1)
	colors2 := color_extractor.ExtractColors(img2)

	c1 = hexColor(colors1[0])
	c2 = hexColor(colors2[0])
	return
}

func genPNG(symbol string, svg []byte, img1, img2 image.Image) {

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
	assetPath := getPath()
	out, err := os.Create(path.Join(assetPath, "icon.png"))
	if err != nil {
		panic(err)
	}
	defer out.Close()

	err = png.Encode(out, rgba)
	if err != nil {
		panic(err)
	}
}

func loadConfig() *mixin.Client {
	f, err := os.Open(*config)
	if err != nil {
		log.Panicln(err)
	}

	var store mixin.Keystore
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		log.Panicln(err)
	}

	client, err := mixin.NewFromKeystore(&store)
	if err != nil {
		log.Panicln(err)
	}

	return client
}

func getAssets(client *mixin.Client) (asset1 *mixin.Asset, asset2 *mixin.Asset, err error) {
	asset1, err = client.ReadAsset(ctx, *assetID1)
	if err != nil {
		return
	}
	asset2, err = client.ReadAsset(ctx, *assetID2)
	if err != nil {
		return
	}
	return
}

func genName(sym1, sym2 string) string {
	if sym1 == "pUSD" {
		return fmt.Sprintf("s%s-%s", strings.ToUpper(sym2), sym1)
	} else if sym2 == "pUSD" {
		return fmt.Sprintf("s%s-%s", strings.ToUpper(sym1), sym2)
	}
	return fmt.Sprintf("s%s-%s", strings.ToUpper(sym1), strings.ToUpper(sym2))
}

func genJSON() (err error) {
	tpl := `{
	"asset_id":"%s",
	"chain_id": "43d61dcd-e413-450d-80b8-101d5e903357",
	"cmc_id": ""
}
`
	assetPath := getPath()
	os.MkdirAll(assetPath, os.ModePerm)
	data := fmt.Sprintf(tpl, *newAssetID)
	err = ioutil.WriteFile(path.Join(assetPath, "index.json"), []byte(data), os.ModePerm)
	return
}

func main() {
	flag.Parse()
	// load config
	client := loadConfig()

	// get two assets from mixin
	asset1, asset2, err := getAssets(client)
	if err != nil {
		log.Panicln(err)
		return
	}

	// fetch icons

	img1, img2, c1, c2, err := loadIcons(
		strings.ReplaceAll(asset1.IconURL, "=s128", ""),
		strings.ReplaceAll(asset2.IconURL, "=s128", ""))
	if err != nil {
		log.Panicln(err)
		return
	}

	// load template
	tpl := loadSVG(c1, c2)

	// generate token name
	symbol := genName(asset1.Symbol, asset2.Symbol)

	// gen json files
	genJSON()

	// gen png
	genPNG(symbol, tpl, img1, img2)
}
