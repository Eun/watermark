package main

import (
	"fmt"

	"strings"

	"io"

	"sync"

	"github.com/Eun/watermark"
	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

var (
	url            = js.Global.Get("URL")
	document       = js.Global.Get("document")
	readableStream = js.Global.Get("ReadableStream")
	response       = js.Global.Get("Response")
	uint8Array     = js.Global.Get("Uint8Array")
)

type marker struct {
	controller *js.Object
	canceled   bool
	mu         sync.Mutex
}

func (m *marker) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.canceled {
		return 0, io.ErrClosedPipe
	}
	size := len(p)
	fmt.Printf("writing %d to bytes\n", size)
	m.controller.Call("enqueue", uint8Array.New(p))
	return size, nil
}

func (m *marker) Start(controller *js.Object) {
	m.controller = controller
}

func (m *marker) Cancel(*js.Object) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.canceled = true
}

func (m *marker) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fmt.Println("closing writer")
	m.controller.Call("close")
	return nil
}

func Mark(file *dom.File, options watermark.Options) (blobUrl *js.Object, err error) {
	var m marker
	fmt.Println("creating ReadableStream")

	readableStream := readableStream.New(map[string]interface{}{
		"start": m.Start,
	})

	fmt.Println("creating Response")
	stream := response.New(readableStream)
	fmt.Println("creating Blob")

	createdUrl := make(chan *js.Object)

	stream.Call("blob").Call("then", func(blob *js.Object) {
		fmt.Println("creating ObjectURL")
		createdUrl <- url.Call("createObjectURL", blob)
	})

	fmt.Println("marking")

	reader := NewFileReader(file)

	if err = watermark.WatermarkReader(reader, &m, options); err != nil {
		return nil, err
	}
	fmt.Println("closing writer stream")
	if err := m.Close(); err != nil {
		return nil, err
	}
	fmt.Println("marking done")
	return <-createdUrl, nil
}

func main() {
	var blobUrl *js.Object
	document.Set("onreadystatechange", func() {
		if !strings.EqualFold(document.Get("readyState").String(), "complete") {
			return
		}
		root := dom.GetWindow().Document()
		src, ok := root.GetElementByID("src").(*dom.HTMLInputElement)
		if !ok {
			panic("#src is not an input field")
		}
		dst, ok := root.GetElementByID("dst").(*dom.HTMLImageElement)
		if !ok {
			panic("#dst is not an image field")
		}
		btn, ok := root.GetElementByID("doit").(*dom.HTMLButtonElement)
		if !ok {
			panic("#doit is not a button")
		}
		alert, ok := root.GetElementByID("alert").(*dom.HTMLDivElement)
		if !ok {
			panic("#alert is not a div")
		}

		btn.AddEventListener("click", true, func(e dom.Event) {
			e.StopPropagation()
			e.PreventDefault()
			files := src.Files()
			if len(files) <= 0 {
				return
			}
			// disable the button to avoid users double clicking
			btn.Disabled = true

			go func() {
				// enable button again
				defer func() {
					btn.Disabled = false
				}()
				if blobUrl != nil {
					// cleanup old url
					url.Call("revokeObjectURL", blobUrl)
					blobUrl = nil
				}
				var err error
				alert.SetTextContent("Working...")
				blobUrl, err = Mark(files[0], watermark.Options{
					Text:  "Hello",
					Scale: 1,
					Color: watermark.ParseColor("#FF0000AA"),
				})
				if err != nil {
					alert.SetTextContent(fmt.Sprintf("Error: %s", err.Error()))
					return
				}
				alert.SetTextContent("Done")
				dst.Set("src", blobUrl)
			}()
		})
	})
}
