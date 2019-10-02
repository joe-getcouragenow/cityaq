// +build js

package gui

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/ctessum/go-leaflet"
	"github.com/ctessum/go-leaflet/plugin/glify"
	"google.golang.org/grpc/grpclog"
)

func (c *CityAQ) loadMap() {
	if c.mapDiv == js.Undefined() {
		c.mapDiv = c.doc.Call("getElementById", "mapDiv")
		c.mapDiv.Get("style").Set("background-color", "black")
	}
	c.setMapHeight()

	// Create map.
	c.leafletMap = leaflet.NewMap(c.mapDiv, map[string]interface{}{"preferCanvas": true})
	c.leafletMap.SetView(leaflet.NewLatLng(39.8282, -98.5795), 4)

	// Add listener to resize map when window resizes.
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.setMapHeight()
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "resize", cb)

	// Add background tiles.
	options := make(map[string]interface{})
	options["Attribution"] = `Map data &copy; <a href=\"http://openstreetmap.org">OpenStreetMap</a> contributors, <a href="http://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>, Imagery © <a href="http://mapbox.com">Mapbox</a>`
	c.baseMapLayer = leaflet.NewTileLayer("https://api.mapbox.com/styles/v1/ctessum/cixuwgf55003e2roe7z5ouk2w/tiles/256/{z}/{x}/{y}?access_token=pk.eyJ1IjoiY3Rlc3N1bSIsImEiOiJjaXh1dnZxYjAwMDRjMzNxcWczZ3JqZDd4In0.972k4y-Xc-PpYTdeUTbufA", options)
	c.baseMapLayer.AddTo(c.leafletMap)
}

// setMapHeight sets the height of the map to the height of the window.
func (c *CityAQ) setMapHeight() {
	const mapMargin = 0 // This is the height of the nav bar.
	height := js.Global().Get("window").Get("innerHeight")
	c.mapDiv.Get("style").Set("height", fmt.Sprintf("%dpx", height.Int()-mapMargin))
}

func (c *CityAQ) updateMap(ctx context.Context, sel *selections) {
	var colors [][]byte
	var legend string
	var err error
	switch sel.impactType {
	case emission:
		c.loadEmissionsGrid(ctx, sel)
		colors, legend, err = c.loadEmissionColors(ctx, sel)
	default:
		grpclog.Printf("invalid impact type %v", sel.impactType)
		panic(err)
		return
	}
	if err != nil {
		grpclog.Println(err)
		panic(err)
		return
	}

	go func() {
		c.setMapLegend(legend)
	}()

	colorF := func(i int) (r, g, b float64) {
		bt := colors[i]
		r = float64(uint8(bt[0])) / 255
		g = float64(uint8(bt[1])) / 255
		b = float64(uint8(bt[2])) / 255
		return
	}

	if c.mapColors != nil {
		c.mapColors.Remove()
	}

	opacity := 0.5
	c.mapColors = glify.NewShapes(c.leafletMap, c.grid.geometry, colorF, opacity)
}

func (c *CityAQ) setMapLegend(legend string) {
	if c.legendDiv == js.Undefined() {
		c.legendDiv = c.doc.Call("getElementById", "legendDiv")
	}
	c.legendDiv.Set("innerHTML", `<img id="legendimg" alt="Embedded Image" src="data:image/png;base64,`+legend+`" />`)
	c.setLegendWidth()

	// Add listener to resize legend when window resizes.
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.setLegendWidth()
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "resize", cb)
}

func (c *CityAQ) setLegendWidth() {
	if c.legendDiv != js.Undefined() {
		rect := c.legendDiv.Call("getBoundingClientRect")
		c.doc.Call("getElementById", "legendimg").Set("width", rect.Get("width").Int())
	}
}
