package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	//"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/disintegration/gift"
	"github.com/pwaller/go-hexcolor"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"

	"golang.org/x/image/draw"
)

var flagText = flag.String("text", "", "Text to use for watermark")
var flagIn = flag.String("in", "-", "File to read from (- for stdin)")
var flagOut = flag.String("out", "-", "File to write to (- for stdout)")
var flagColor = flag.String("color", "#FF0000AA", "Color to use for the text")
var flagScale = flag.Float64("scale", 1, "Scale text")

func main() {
	flag.Parse()

	textColor := parseColor(*flagColor)

	reader := os.Stdin
	if *flagIn != "-" {
		var err error
		reader, err = os.Open(*flagIn)
		if err != nil {
			log.Fatalf("failed to open: %s", err)
		}
		defer reader.Close()
	}

	source, format, err := image.Decode(reader)
	if err != nil {
		log.Fatalf("unable to decode image: %s", err)
	}

	writer := os.Stdout
	if *flagOut != "-" {
		var err error
		writer, err = os.Create(*flagOut)
		if err != nil {
			log.Fatalf("failed to open: %s", err)
		}
		defer writer.Close()
	}

	if len(*flagText) <= 0 {
		t := time.Now()
		*flagText = fmt.Sprintf("%02d.%02d.%04d", t.Day(), t.Month(), t.Year())
	}

	watermark := createWatermark(*flagText, *flagScale, textColor)

	sourceBounds := source.Bounds()
	watermarkBounds := watermark.Bounds()
	markedImage := image.NewRGBA(sourceBounds)
	draw.Draw(markedImage, sourceBounds, source, image.ZP, draw.Src)

	// horrizontal
	var offset image.Point
	for offset.X = watermarkBounds.Max.X / -2; offset.X < sourceBounds.Max.X; offset.X += watermarkBounds.Max.X {
		for offset.Y = watermarkBounds.Max.Y / -2; offset.Y < sourceBounds.Max.Y; offset.Y += watermarkBounds.Max.Y {
			draw.Draw(markedImage, watermarkBounds.Add(offset), watermark, image.ZP, draw.Over)
		}
	}

	switch format {
	case "png":
		err = png.Encode(writer, markedImage)
	case "gif":
		err = gif.Encode(writer, markedImage, &gif.Options{NumColors: 265})
	case "jpeg":
		err = jpeg.Encode(writer, markedImage, &jpeg.Options{Quality: jpeg.DefaultQuality})
	default:
		log.Fatalf("unknown format %s", format)
	}
	if err != nil {
		log.Fatalf("unable to encode image: %s", err)
	}
}

func parseColor(str string) color.Color {
	r, g, b, a := hexcolor.HexToRGBA(hexcolor.Hex(str))
	return color.RGBA{
		A: a,
		R: r,
		G: g,
		B: b,
	}
}

func createWatermark(text string, scale float64, textColor color.Color) image.Image {
	var padding float64 = 2
	w := 8 * (float64(len(text)) + (padding * 2))
	h := 16 * padding
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	point := fixed.Point26_6{fixed.Int26_6(64 * padding), fixed.Int26_6(h * 64)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: inconsolata.Regular8x16,
		Dot:  point,
	}
	d.DrawString(text)

	bounds := img.Bounds()
	scaled := image.NewRGBA(image.Rect(0, 0, int(float64(bounds.Max.X)*scale), int(float64(bounds.Max.Y)*scale)))
	draw.BiLinear.Scale(scaled, scaled.Bounds(), img, bounds, draw.Src, nil)

	g := gift.New(
		gift.Rotate(45, color.Transparent, gift.CubicInterpolation),
	)
	rot := image.NewNRGBA(g.Bounds(scaled.Bounds()))
	g.Draw(rot, scaled)
	return rot
}
