package main

import (
	"errors"
	"io"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

// FileReader wraps a File to allow for reading
type FileReader struct {
	File         *dom.File
	fr           *js.Object
	offset, size int64
}

// NewFileReader opens a File for reading
func NewFileReader(f *dom.File) *FileReader {
	return &FileReader{
		f,
		js.Global.Get("FileReader").New(),
		0,
		int64(f.Get("size").Int()),
	}
}

// Close implements the io.Closer interface
func (fr *FileReader) Close() error {
	fr.fr = nil
	return nil
}

// Read implements the io.Reader interface
func (fr *FileReader) Read(b []byte) (int, error) {
	n, err := fr.ReadAt(b, int64(fr.offset))
	fr.offset += int64(n)
	return n, err
}

//ReadAt implements the io.ReaderAt interface
func (fr *FileReader) ReadAt(b []byte, off int64) (int, error) {
	if fr.fr == nil {
		return 0, ErrClosed
	}
	if off >= fr.size {
		return 0, io.EOF
	}

	type readResult struct {
		size int
		err  error
	}

	c := make(chan readResult)
	fr.fr.Set("onloadend", func(*js.Object) {
		arr := js.Global.Get("Uint8Array").New(fr.fr.Get("result"))
		buf := arr.Interface().([]byte)
		go func() {
			if len(buf) == 0 {
				c <- readResult{0, io.EOF}
			} else {
				copy(b, buf)
				c <- readResult{len(buf), nil}
			}
		}()
	})
	e := off + int64(len(b))
	if e > fr.size {
		e = fr.size
	}
	blob := fr.File.Call("slice", fr.offset, e)
	fr.fr.Call("readAsArrayBuffer", blob)
	r := <-c
	return r.size, r.err
}

// Seek implements the io.Seeker interface
func (fr *FileReader) Seek(offset int64, whence int) (int64, error) {
	if fr.fr == nil {
		return 0, ErrClosed
	}
	switch whence {
	case 0:
		fr.offset = offset
	case 1:
		fr.offset += offset
	case 2:
		fr.offset = fr.size + offset
	}
	if fr.offset < 0 {
		fr.offset = 0
	}
	return fr.offset, nil
}

// Errors
var (
	ErrClosed = errors.New("file closed")
)
