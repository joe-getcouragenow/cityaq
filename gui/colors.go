// +build js

package gui

import (
	"context"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
)

// loadEmissionColors returns emissions map data.
func (c *CityAQ) loadEmissionColors(ctx context.Context, sel *selections) (colors [][]byte, legend string, err error) {
	resp, err := c.EmissionsMap(ctx, &rpc.EmissionsMapRequest{
		CityName:   sel.cityName,
		CityPath:   sel.cityPath,
		Emission:   sel.emission,
		SourceType: sel.sourceType,
	})
	if err != nil {
		return nil, "", err
	}
	return resp.RGB, resp.Legend, nil
}
