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
	"text/template"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
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

	scale := mapboxgl.Get("ScaleControl").New()
	c.mapboxMap.Call("addControl", scale)

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

	if c.legendDiv != js.Undefined() {
		c.legendDiv.Set("innerHTML", "")
	}
	if c.summaryDiv != js.Undefined() {
		c.summaryDiv.Set("innerHTML", "")
	}
	go func() {
		c.summary(sel) // Update summary statistics.
	}()

	if c.dataLayer != js.Undefined() {
		c.mapboxMap.Call("removeLayer", "data")
		c.mapboxMap.Call("removeLayer", "city")
		c.mapboxMap.Call("removeSource", "data")
		c.mapboxMap.Call("removeSource", "city")
		c.dataLayer = js.Undefined()
		c.cityLayer = js.Undefined()
	}
	if c.egugridLayer != js.Undefined() {
		c.mapboxMap.Call("removeLayer", "egugrid")
		c.mapboxMap.Call("removeSource", "egugrid")
		c.egugridLayer = js.Undefined()
	}

	b, err := c.EmissionsGridBounds(ctx, &rpc.EmissionsGridBoundsRequest{
		CityName:   sel.cityName,
		SourceType: sel.sourceType,
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

	if strings.HasSuffix(sel.sourceType, "_egugrid") {
		c.egugridLayer = c.mapboxMap.Call("addLayer", map[string]interface{}{
			"id":           "egugrid",
			"source":       source,
			"source-layer": sel.cityName + "_egugrid",
			"type":         "line",
			"paint": js.ValueOf(map[string]interface{}{
				"line-width": 3,
				"line-color": "#958de0",
			}),
		})
	}

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
	cm.SetMin(scale.Min)
	cm.SetMax(scale.Max)
	go func() {
		c.setMapLegend(cm, sel.impactType)
	}()

	cutpts := make([]float64, 10)
	r := scale.Max - scale.Min
	floats.Span(cutpts, scale.Min, scale.Max)
	colors := make([]interface{}, len(cutpts)*2)
	for i := 0; i < len(colors); i += 2 {
		v := cutpts[i/2]
		color, err := cm.At(v)
		if err != nil {
			color, err = cm.At(v - 1e-10*r)
			if err != nil {
				color, err = cm.At(v + 1e-10*r)
				if err != nil {
					return nil, fmt.Errorf("creating mapbox color scale: %w", err)
				}
			}
		}
		r, g, b, _ := color.RGBA()
		r, g, b = r/0x101, g/0x101, b/0x101
		hex := fmt.Sprintf("#%02X%02X%02X", r, g, b)
		colors[i] = v
		colors[i+1] = hex
	}
	return colors, nil
}

func (c *CityAQ) setMapLegend(cm palette.ColorMap, it rpc.ImpactType) {
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

	img := vgimg.NewWith(vgimg.UseWH(150, 30), vgimg.UseDPI(300))
	dc := draw.New(img)
	dc = draw.Crop(dc, 0, 0, vg.Points(2), 0)
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
	var title string
	switch it {
	case rpc.ImpactType_Emissions:
		title = "<p class=\"small text-center\">Emissions (kg / kilotonne)</p>"
	case rpc.ImpactType_Concentrations:
		title = "<p class=\"small text-center\">PM<sub>2.5</sub> concentrations (μg m<sup>-3</sup> / kilotonne emissions)</p>"
	}
	c.legendDiv.Set("innerHTML", title+`<img id="legendimg" class="img-fluid" alt="Legend" src="data:image/png;base64,`+legendStr+`" />`)
}

// template for statistics summary modal
var statsModal *template.Template

// initialize statistics summary template modal.
func init() {
	statsModal = template.Must(template.New("statsModal").Parse(`
	<button type="button" class="btn btn-light" data-toggle="modal" data-target="#statsModal">
	  View impact statistics
	</button>

	<div class="modal fade" id="statsModal" tabindex="-1" role="dialog" aria-labelledby="statsModalLabel" aria-hidden="true">
	  <div class="modal-dialog" role="document">
	    <div class="modal-content">
	      <div class="modal-header">
	        <h5 class="modal-title" id="statsModalLabel">Statistics</h5>
	        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
	          <span aria-hidden="true">&times;</span>
	        </button>
	      </div>
	      <div class="modal-body">
					<table class="table">
						<thead>
							<tr>
								<th scope="col"></th>
								<th scope="col">Within City</th>
								<th scope="col">Domain Total</th>
							</tr>
						</thead>
						<tbody>
							<tr>
								<td>Population</td>
								<td>{{printf "%.2g" .CityPopulation}}</td>
								<td>{{printf "%.2g" .Population}}</td>
							</tr>
							<tr>
								<td><a href="#" data-toggle="tooltip" title="Population-weighted average concentration">Exposure</a> (μg m<sup>-3</sup>)</td>
								<td>{{printf "%.2g" .CityExposure}}</td>
								<td>{{printf "%.2g" .TotalExposure}}</td>
							</tr>
							<tr>
								<td><a href="#" data-toggle="tooltip" title="Intake fraction">iF</a> (ppm)</td>
								<td>{{printf "%.2g" .CityIF}}</td>
								<td>{{printf "%.2g" .TotalIF}}</td>
							</tr>
						</tbody>
					</table>
	      </div>
	    </div>
	  </div>
	</div>`))
}

func (c *CityAQ) summary(sel *selections) error {
	if sel.impactType != rpc.ImpactType_Concentrations {
		return nil
	}
	if c.summaryDiv == js.Undefined() {
		c.summaryDiv = c.doc.Call("getElementById", "summaryDiv")
	}
	impacts, err := c.ImpactSummary(context.TODO(), &rpc.ImpactSummaryRequest{
		CityName:   sel.cityName,
		Emission:   sel.emission,
		SourceType: sel.sourceType,
	})
	if err != nil {
		return err
	}
	w := new(strings.Builder)
	if err := statsModal.Execute(w, impacts); err != nil {
		return err
	}
	c.summaryDiv.Set("innerHTML", w.String())
	return nil
}

// Move the map window to a new location.
func (c *CityAQ) MoveMap(ctx context.Context, cityName, sourceType string) {
	b, err := c.EmissionsGridBounds(ctx, &rpc.EmissionsGridBoundsRequest{
		CityName:   cityName,
		SourceType: sourceType,
	})
	if err != nil {
		c.logError(err)
		return
	}

	min := js.ValueOf([]interface{}{b.Min.X, b.Min.Y})
	max := js.ValueOf([]interface{}{b.Max.X, b.Max.Y})

	c.mapboxMap.Call("fitBounds", []interface{}{min, max})
}
