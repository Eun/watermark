package watermark

import (
	"fmt"
	"image"

	"os"
	"testing"
)

func TestWatermarkImage(t *testing.T) {
	f, err := os.Open("sample.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	source, format, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%T", source)
	fmt.Println(format)
}
