package gui

import (
	"syscall/js"
	"testing"
)

func TestLoadMap(t *testing.T) {

	c := &CityAQ{
		//		CityAQClient: client,
		doc: js.Global().Get("document"),
	}
	c.mapDiv = c.doc.Call("createElement", "div")

	t.Run("loadMap", func(t *testing.T) {
		c.loadMap()
		height := c.mapDiv.Get("style").Get("height").String()
		wantHeight := "600px"
		if height != wantHeight {
			t.Errorf("%s != %s", height, wantHeight)
		}
	})

	// TODO: Figure out why WebGL doesn't work in test environment
	/*t.Run("updateMap", func(t *testing.T) {
		c.legendDiv = c.doc.Call("createElement", "div")

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		client := caqmock.NewMockCityAQClient(mockCtrl)

		client.EXPECT().EmissionsMap(
			gomock.Any(),
			gomock.AssignableToTypeOf(&rpc.EmissionsMapRequest{}),
		).Return(&rpc.EmissionsMapResponse{
			RGB: [][]byte{
				{0, 1, 2},
				{1, 2, 3},
				{2, 3, 4},
			},
			Legend: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVQYV2NgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=",
		}, nil)

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

		c.CityAQClient = client

		c.updateMap(context.Background(), &selections{
			cityName:   "city1",
			cityPath:   "city1path",
			impactType: emission,
			emission:   1,
			sourceType: "roads"},
		)

	})*/

}
