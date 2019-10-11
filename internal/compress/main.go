package main

import (
	"io"
	"os"

	"github.com/andybalholm/brotli"
)

func main() {
	w, err := os.Create("gui/html/cityaq.wasm.br")
	if err != nil {
		panic(err)
	}
	wb := brotli.NewWriterLevel(w, brotli.BestCompression)
	r, err := os.Open("gui/html/cityaq.wasm")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(wb, r); err != nil {
		panic(err)
	}
	if err := wb.Close(); err != nil {
		panic(err)
	}
	w.Close()
	r.Close()
}
