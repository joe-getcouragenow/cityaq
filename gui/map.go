// +build js

package gui

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"strings"
	"sync"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

var mapboxgl js.Value

func (c *CityAQ) loadMap() {
	if c.mapDiv == js.Undefined() {
		c.mapDiv = c.doc.Call("getElementById", "mapDiv")
		c.mapDiv.Get("style").Set("background-color", "black")
	}
	c.setMapHeight()

	// Load mapbox CSS.
	link := c.doc.Call("createElement", "link")
	link.Set("href", "https://api.tiles.mapbox.com/mapbox-gl-js/v1.4.1/mapbox-gl.css")
	link.Set("type", "text/css")
	link.Set("rel", "stylesheet")
	c.doc.Get("head").Call("appendChild", link)

	// Load mapbox javascript.
	script := c.doc.Call("createElement", "script")
	script.Set("src", "https://api.tiles.mapbox.com/mapbox-gl-js/v1.4.1/mapbox-gl.js")
	c.doc.Get("head").Call("appendChild", script)

	var wg sync.WaitGroup
	wg.Add(1)
	var callback js.Func
	callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		mapboxgl = js.Global().Get("mapboxgl")
		callback.Release()
		wg.Done()
		return nil
	})
	script.Set("onreadystatechange", callback)
	script.Set("onload", callback)
	wg.Wait()

	mapboxgl.Set("accessToken", "pk.eyJ1IjoiY3Rlc3N1bSIsImEiOiJjaXh1dnZxYjAwMDRjMzNxcWczZ3JqZDd4In0.972k4y-Xc-PpYTdeUTbufA")

	// Create map.
	c.mapboxMap = mapboxgl.Get("Map").New(map[string]interface{}{
		"container": c.mapDiv,
		"style":     "mapbox://styles/ctessum/cixuwgf55003e2roe7z5ouk2w",
		"zoom":      0,
	})

	// Add listener to resize map when window resizes.
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.setMapHeight()
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "resize", cb)
}

// setMapHeight sets the height of the map to the height of the window.
func (c *CityAQ) setMapHeight() {
	const mapMargin = 0 // This is the height of the nav bar.
	height := js.Global().Get("window").Get("innerHeight")
	c.mapDiv.Get("style").Set("height", fmt.Sprintf("%dpx", height.Int()-mapMargin))
}

func (c *CityAQ) updateMap(ctx context.Context, sel *selections) {
	c.startLoading()

	if c.dataLayer != js.Undefined() {
		c.mapboxMap.Call("removeLayer", "data")
		c.mapboxMap.Call("removeLayer", "city")
		c.mapboxMap.Call("removeSource", "data")
		c.mapboxMap.Call("removeSource", "city")
		c.dataLayer = js.Undefined()
		c.cityLayer = js.Undefined()
	}

	b, err := c.EmissionsGridBounds(ctx, &rpc.EmissionsGridBoundsRequest{
		CityName:   sel.cityName,
		SourceType: sel.sourceType,
		Dx:         float32(mapResolution(sel.sourceType)),
	})
	if err != nil {
		c.logError(err)
		c.stopLoading()
		return
	}

	colors, err := c.legend(sel)
	if err != nil {
		c.logError(err)
		c.stopLoading()
		return
	}

	// Find the index of the first layer in the map style
	layer0ID := c.mapboxMap.Call("getStyle").Get("layers").Index(0).Get("id").String()

	source := js.ValueOf(map[string]interface{}{
		"type": "vector",
		"tiles": js.ValueOf([]interface{}{
			fmt.Sprintf("%smaptile?x={x}&y={y}&z={z}&c=%s&it=%d&em=%d&st=%s",
				c.doc.Get("baseURI").String(), html.EscapeString(sel.cityName),
				sel.impactType, sel.emission, sel.sourceType),
		}),
		"bounds":      js.ValueOf([]interface{}{b.Min.X, b.Min.Y, b.Max.X, b.Max.Y}),
		"attribution": "© CityAQ authors",
	})

	c.dataLayer = c.mapboxMap.Call("addLayer", map[string]interface{}{
		"id":     "data",
		"source": source,
		"source-layer": fmt.Sprintf(fmt.Sprintf("%s_%d_%d_%s",
			sel.cityName, sel.impactType, sel.emission, sel.sourceType)),
		"type": "fill",
		"paint": js.ValueOf(map[string]interface{}{
			"fill-color": js.ValueOf(append([]interface{}{
				"interpolate-lab",
				js.ValueOf([]interface{}{"linear"}),   // Interpolation settings
				js.ValueOf([]interface{}{"get", "v"}), // Which property to use.
			}, colors...), // The colors and cutpoints.
			),
		}),
	}, layer0ID)

	c.cityLayer = c.mapboxMap.Call("addLayer", map[string]interface{}{
		"id":           "city",
		"source":       source,
		"source-layer": sel.cityName,
		"type":         "line",
		"paint": js.ValueOf(map[string]interface{}{
			"line-width": 3,
			"line-color": "#8dbbe0",
		}),
	})

	// Turn off the loading symbol when the map becomes idle.
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.stopLoading()
		c.mapboxMap.Call("off", "idle", cb)
		cb.Release()
		return nil
	})
	c.mapboxMap.Call("on", "idle", cb)
}

func (c *CityAQ) legend(sel *selections) ([]interface{}, error) {
	scale, err := c.MapScale(context.TODO(), &rpc.MapScaleRequest{
		CityName:   sel.cityName,
		Emission:   sel.emission,
		SourceType: sel.sourceType,
		ImpactType: sel.impactType,
	})
	if err != nil {
		return nil, err
	}
	cm := moreland.ExtendedBlackBody()
	cm.SetMin(float64(scale.Min))
	cm.SetMax(float64(scale.Max))
	go func() {
		c.setMapLegend(cm)
	}()

	cutpts := make([]float64, 10)
	floats.Span(cutpts, float64(scale.Min), float64(scale.Max))
	colors := make([]interface{}, len(cutpts)*2)
	for i := 0; i < len(colors); i += 2 {
		v := cutpts[i/2]
		color, err := cm.At(v)
		if err != nil {
			return nil, err
		}
		r, g, b, _ := color.RGBA()
		r, g, b = r/0x101, g/0x101, b/0x101
		hex := fmt.Sprintf("#%02X%02X%02X", r, g, b)
		colors[i] = v
		colors[i+1] = hex
	}
	return colors, nil
}

func (c *CityAQ) setMapLegend(cm palette.ColorMap) {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	l := &plotter.ColorBar{
		ColorMap: cm,
	}
	p.Add(l)
	p.HideY()
	p.X.Padding = 0

	img := vgimg.New(300, 40)
	dc := draw.New(img)
	p.Draw(dc)
	b := new(bytes.Buffer)
	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(b); err != nil {
		panic(err)
	}
	legendStr := base64.StdEncoding.EncodeToString(b.Bytes())

	if c.legendDiv == js.Undefined() {
		c.legendDiv = c.doc.Call("getElementById", "legendDiv")
	}
	c.legendDiv.Set("innerHTML", `<img id="legendimg" alt="Embedded Image" src="data:image/png;base64,`+legendStr+`" />`)
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

// nationalEmissions returns whether the given sourceType should
// be allocated to the a country rather than a city.
func nationalEmissions(sourceType string) bool {
	return strings.HasSuffix(sourceType, "_national")
}

func mapResolution(sourceType string) float64 {
	if nationalEmissions(sourceType) {
		return 0.01
	}
	return 0.002
}

// Move the map window to a new location.
func (c *CityAQ) MoveMap(ctx context.Context, cityName, sourceType string) {
	b, err := c.EmissionsGridBounds(ctx, &rpc.EmissionsGridBoundsRequest{
		CityName:   cityName,
		SourceType: sourceType,
		Dx:         float32(mapResolution(sourceType)),
	})
	if err != nil {
		c.logError(err)
		return
	}

	min := js.ValueOf([]interface{}{b.Min.X, b.Min.Y})
	max := js.ValueOf([]interface{}{b.Max.X, b.Max.Y})

	c.mapboxMap.Call("fitBounds", []interface{}{min, max})
}
