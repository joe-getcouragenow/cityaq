package cityaq

import (
	"bytes"
	"encoding/base64"
	"sort"

	"github.com/ctessum/sparse"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func newColormap(a *sparse.SparseArray) palette.ColorMap {
	cm1 := moreland.ExtendedBlackBody()
	/*cm2, err := moreland.NewLuminance([]color.Color{
		color.NRGBA{G: 176, A: 255},
		color.NRGBA{G: 255, A: 255},
	})
	if err != nil {
		panic(err)
	}*/
	//cm := &plotextra.BrokenColorMap{
	//	Base:     cm1,
	//	OverFlow: palette.Reverse(cm2),
	//}
	//minMaxCutpt := percentiles(a, 1, 0.99)
	max := percentiles(a, 1)
	cm1.SetMin(0)
	cm1.SetMax(max[0])
	//cm.SetHighCut(minMaxCutpt[1])
	return cm1
}

// percentiles returns percentiles p (range [0,1]) of the given data.
func percentiles(data *sparse.SparseArray, p ...float64) []float64 {
	tmp := make([]float64, 0, len(data.Elements))
	for _, v := range data.Elements {
		tmp = append(tmp, v)
	}
	sort.Float64s(tmp)
	o := make([]float64, len(p))
	for i, pp := range p {
		o[i] = tmp[roundInt(pp*float64(len(data.Elements)-1))]
	}
	return o
}

// roundInt rounds a float to an integer
func roundInt(x float64) int {
	return int(x + 0.5)
}

func legend(cm palette.ColorMap) string {
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
	return base64.StdEncoding.EncodeToString(b.Bytes())
}
