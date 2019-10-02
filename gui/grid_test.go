package gui

import (
	"context"
	"syscall/js"
	"testing"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	caqmock "github.com/ctessum/cityaq/cityaqrpc/mock_cityaqrpc"
	"github.com/golang/mock/gomock"
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

	nFeatures := c.grid.geometry.Get("features").Length()
	wantFeatures := 1
	if nFeatures != wantFeatures {
		t.Errorf("wrong number of features: %d != %d", nFeatures, wantFeatures)
	}
	points := c.grid.geometry.Get("features").Index(0).Get("geometry").Get("coordinates")
	pointsStr := js.Global().Get("JSON").Call("stringify", points).String()
	wantPointsStr := "[[[0,0],[1,0],[1,1]]]"
	if pointsStr != wantPointsStr {
		t.Errorf("wrong points: %s != %s", pointsStr, wantPointsStr)
	}
}
