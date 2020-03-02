package watermark

import (
	"image"
	"image/color"

	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/gift"
	"github.com/pkg/errors"
	"github.com/pwaller/go-hexcolor"
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

type Options struct {
	Text  string
	Scale float64
	Color color.Color
}

func WatermarkReader(src io.Reader, dst io.Writer, options Options) error {
	source, format, err := image.Decode(src)
	if err != nil {
		return errors.Wrapf(err, "unable to decode image")
	}
	markedImage, err := WatermarkImage(source, options)
	if err != nil {
		return errors.Wrap(err, "unable to mark image")
	}
	switch format {
	case "bmp":
		err = bmp.Encode(dst, markedImage)
	case "gif":
		err = gif.Encode(dst, markedImage, &gif.Options{NumColors: 265})
	case "jpeg":
		err = jpeg.Encode(dst, markedImage, &jpeg.Options{Quality: jpeg.DefaultQuality})
	case "png":
		err = png.Encode(dst, markedImage)
	default:
		return errors.Errorf("unable to encode image to %s: unknown format", format)
	}
	if err != nil {
		return errors.Wrapf(err, "unable to encode image to %s", format)
	}
	return nil
}

func WatermarkImage(src image.Image, options Options) (image.Image, error) {
	watermark := createWatermark(options.Text, options.Scale, options.Color)
	sourceBounds := src.Bounds()
	watermarkBounds := watermark.Bounds()
	markedImage := image.NewRGBA(sourceBounds)
	draw.Draw(markedImage, sourceBounds, src, image.ZP, draw.Src)

	// horizontal
	var offset image.Point
	for offset.X = watermarkBounds.Max.X / -2; offset.X < sourceBounds.Max.X; offset.X += watermarkBounds.Max.X {
		for offset.Y = watermarkBounds.Max.Y / -2; offset.Y < sourceBounds.Max.Y; offset.Y += watermarkBounds.Max.Y {
			draw.Draw(markedImage, watermarkBounds.Add(offset), watermark, image.ZP, draw.Over)
		}
	}

	return markedImage, nil
}

func ParseColor(str string) color.Color {
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
