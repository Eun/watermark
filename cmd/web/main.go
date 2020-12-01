package main

import (
	"fmt"
	"io"
	"sync"

	"strings"

	"os"

	"strconv"

	"github.com/Eun/watermark"
	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

var jsx struct {
	JSON           *js.Object
	ReadableStream *js.Object
	Response       *js.Object
	Uint8Array     *js.Object
	URL            *js.Object
	Worker         *js.Object
}

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
	m.controller.Call("enqueue", jsx.Uint8Array.New(p))
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

func Mark(file *dom.File, options *watermark.Options) (blobUrl string, err error) {
	var m marker
	fmt.Println("creating ReadableStream")

	readableStream := jsx.ReadableStream.New(map[string]interface{}{
		"start": m.Start,
	})

	fmt.Println("creating Response")
	stream := jsx.Response.New(readableStream)
	fmt.Println("creating Blob")

	createdUrl := make(chan string)

	stream.Call("blob").Call("then", func(blob *js.Object) {
		fmt.Println("creating ObjectURL")
		createdUrl <- jsx.URL.Call("createObjectURL", blob).String()
	})

	fmt.Println("marking")

	reader := NewFileReader(file)

	if err = watermark.WatermarkReader(reader, &m, options); err != nil {
		return "", err
	}
	fmt.Println("closing writer stream")
	if err := m.Close(); err != nil {
		return "", err
	}
	fmt.Println("marking done")
	return <-createdUrl, nil
}

func main() {
	if js.Global.Get("document") == js.Undefined {
		worker()
		return
	}
	web()
	return
}

func mustGet(s string, fn func(string)) *js.Object {
	v := js.Global.Get(s)
	if v == nil {
		fn(s)
		return nil
	}
	return v
}

func initJSX(fn func(string)) {
	jsx.JSON = mustGet("JSON", fn)
	jsx.ReadableStream = mustGet("ReadableStream", fn)
	jsx.Response = mustGet("Response", fn)
	jsx.Uint8Array = mustGet("Uint8Array", fn)
	jsx.URL = mustGet("URL", fn)
	jsx.Worker = mustGet("Worker", fn)
}

type options struct {
	*js.Object
	Text  string  `js:"text"`
	Scale float64 `js:"scale"`
	Color string  `js:"color"`
}

func getInputValueAsString(element *dom.HTMLInputElement) string {
	if element.Value == "" {
		return element.GetAttribute("data-default")
	}
	return element.Value
}

func getInputValueAsFloat64(element *dom.HTMLInputElement) (float64, error) {
	if element.Value == "" {
		return strconv.ParseFloat(element.GetAttribute("data-default"), 0)
	}
	return element.ValueAsNumber, nil
}

func web() {
	var blobUrl *js.Object
	document, ok := dom.GetWindow().Document().(dom.HTMLDocument)
	if !ok {
		panic("document is not an HTMLDocument")
	}
	document.AddEventListener("readystatechange", false, func(event dom.Event) {
		if !strings.EqualFold(document.ReadyState(), "complete") {
			return
		}
		app, ok := document.GetElementByID("app").(*dom.HTMLDivElement)
		if !ok {
			panic("#app is not a div")
		}
		form, ok := document.GetElementByID("form").(*dom.HTMLFormElement)
		if !ok {
			panic("#form is not a form")
		}

		srcInput, ok := document.GetElementByID("src").(*dom.HTMLInputElement)
		if !ok {
			panic("#src is not an input field")
		}
		textInput, ok := document.GetElementByID("text").(*dom.HTMLInputElement)
		if !ok {
			panic("#text is not an text field")
		}
		scaleInput, ok := document.GetElementByID("scale").(*dom.HTMLInputElement)
		if !ok {
			panic("#scale is not an text field")
		}
		colorInput, ok := document.GetElementByID("color").(*dom.HTMLInputElement)
		if !ok {
			panic("#color is not an text field")
		}
		opacityInput, ok := document.GetElementByID("opacity").(*dom.HTMLInputElement)
		if !ok {
			panic("#opacity is not an text field")
		}
		dstImage, ok := document.GetElementByID("dst").(*dom.HTMLImageElement)
		if !ok {
			panic("#dst is not an image field")
		}

		alertText, ok := document.GetElementByID("alert").(*dom.HTMLDivElement)
		if !ok {
			panic("#alert is not a div")
		}

		app.Style().Set("display", "block")

		initJSX(func(s string) {
			alertText.SetTextContent(s)
			os.Exit(0)
		})

		form.AddEventListener("submit", true, func(e dom.Event) {
			e.StopPropagation()
			e.PreventDefault()
			files := srcInput.Files()
			if len(files) <= 0 {
				return
			}
			// disable the button to avoid users double clicking
			elements := form.QuerySelectorAll("*")
			for _, el := range elements {
				el.SetAttribute("disabled", "true")
			}

			if blobUrl != nil {
				// cleanup old url
				jsx.URL.Call("revokeObjectURL", blobUrl)
				blobUrl = nil
			}
			alertText.SetTextContent("Working...")

			w := jsx.Worker.New("web.js")
			w.Set("onmessage", func(e *js.Object) {
				defer func() {
					// disable the button to avoid users double clicking
					elements := form.QuerySelectorAll("*")
					for _, el := range elements {
						el.RemoveAttribute("disabled")
					}
				}()
				data := e.Get("data")
				if err := data.Get("Error").String(); err != "" {
					alertText.SetTextContent(err)
					return
				}
				alertText.SetTextContent("Done")
				// dstImage.Set("src", data.Get("Url"))
				dstImage.Src = data.Get("Url").String()
				scaleInput.Value = strconv.FormatFloat(data.Get("ScaleUsed").Float(), 'f', -1, 64)
				dstImage.SetTitle(files[0].Get("name").String())
			})
			o := options{Object: js.Global.Get("Object").New()}
			o.Text = getInputValueAsString(textInput)
			var err error
			o.Scale, err = getInputValueAsFloat64(scaleInput)
			if err != nil {
				alertText.SetTextContent(err.Error())
				return
			}
			o.Color = getInputValueAsString(colorInput)
			opacity, err := getInputValueAsFloat64(opacityInput)
			if err != nil {
				alertText.SetTextContent(err.Error())
				return
			}
			o.Color += fmt.Sprintf("%02x", int(255*opacity/100))
			w.Call("postMessage", map[string]interface{}{
				"options": o,
				"file":    files[0],
			})
		})
	})
}

func worker() {
	printToPanic := func(s string) {
		panic(s)
	}
	self := mustGet("self", printToPanic)
	initJSX(printToPanic)

	self.Set("onmessage", func(e *js.Object) {
		go func() {
			var sendMsg struct {
				Error string
				ScaleUsed float64
				Url   string
			}
			defer func() {
				self.Call("postMessage", sendMsg)
			}()
			data := e.Get("data")

			file := &dom.File{data.Get("file")}
			o := &options{Object: data.Get("options")}
			opts := watermark.Options{
				Text:  o.Text,
				Scale: o.Scale,
				Color: watermark.ParseColor(o.Color),
			}

			var err error
			sendMsg.Url, err = Mark(file, &opts)
			if err != nil {
				sendMsg.Error = err.Error()
				return
			}
			sendMsg.ScaleUsed = opts.Scale
		}()
	})
}
