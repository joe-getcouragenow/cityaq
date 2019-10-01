package gui

import (
	"context"
	"reflect"
	"syscall/js"
	"testing"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	caqmock "github.com/ctessum/cityaq/cityaqrpc/mock_cityaqrpc"
	"github.com/golang/mock/gomock"
	"github.com/norunners/vert"
)

func TestLoadEmissionsGrid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	client := caqmock.NewMockCityAQClient(mockCtrl)

	client.EXPECT().EmissionsGrid(
		gomock.Any(),
		gomock.AssignableToTypeOf(&rpc.EmissionsGridRequest{}),
	).Return(&rpc.EmissionsGridResponse{
		Polygons: []*rpc.Polygon{
			{
				Paths: []*rpc.Path{
					{
						Points: []*rpc.Point{
							{X: 0, Y: 0},
							{X: 1, Y: 0},
							{X: 1, Y: 1},
						},
					},
				},
			},
		},
	}, nil)

	c := &CityAQ{
		CityAQClient: client,
		doc:          js.Global().Get("document"),
	}

	c.loadEmissionsGrid(context.Background(), &selections{})

	var g geojson
	vert.Value{c.grid.geometry}.AssignTo(&g)

	want := geojson{
		Features: []geojsonGeom{
			geojsonGeom{Geometry: struct {
				Coordinates [][][]float32 "json:\"coordinates\",js:\"coordinates\""
			}{
				Coordinates: [][][]float32{[][]float32{[]float32{0, 0}, []float32{1, 0}, []float32{1, 1}}}},
			},
		},
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("%v != %v", g, want)
	}
}
