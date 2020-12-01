package main

import (
	"flag"
	"fmt"

	"log"
	"os"
	"time"

	"github.com/Eun/watermark"
)

var flagText = flag.String("text", "", "Text to use for watermark")
var flagIn = flag.String("in", "-", "File to read from (- for stdin)")
var flagOut = flag.String("out", "-", "File to write to (- for stdout)")
var flagColor = flag.String("color", "#FF0000AA", "Color to use for the text")
var flagScale = flag.Float64("scale", 1, "Scale text")

func main() {
	flag.Parse()

	var options watermark.Options

	options.Color = watermark.ParseColor(*flagColor)

	reader := os.Stdin
	if *flagIn != "-" {
		var err error
		reader, err = os.Open(*flagIn)
		if err != nil {
			log.Fatalf("failed to open: %s", err)
		}
		defer reader.Close()
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

	options.Text = *flagText
	options.Scale = *flagScale

	if err := watermark.WatermarkReader(reader, writer, &options); err != nil {
		log.Fatalf("watermark failed: %s", err)
	}
}
