package gui

import (
	"context"
	"reflect"
	"strings"
	"syscall/js"
	"testing"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	caqmock "github.com/ctessum/cityaq/cityaqrpc/mock_cityaqrpc"
	"github.com/golang/mock/gomock"
)

func TestDOM(t *testing.T) {
	doc := js.Global().Get("document")
	elem := doc.Call("createElement", "div")
	inputString := "hello world"
	elem.Set("innerText", inputString)
	out := elem.Get("innerText")

	// need Contains because a "\n" gets appended in the output
	if !strings.Contains(out.String(), inputString) {
		t.Errorf("unexpected output string. Expected %q to contain %q", out.String(), inputString)
	}
}

func TestCitySelector(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	client := caqmock.NewMockCityAQClient(mockCtrl)

	client.EXPECT().Cities(
		gomock.Any(),
		gomock.AssignableToTypeOf(&rpc.CitiesRequest{}),
	).Return(&rpc.CitiesResponse{Names: []string{"city1", "city2"}, Paths: []string{"city1path", "city2path"}}, nil)

	c := &CityAQ{
		CityAQClient: client,
		doc:          js.Global().Get("document"),
	}
	c.citySelector = c.doc.Call("createElement", "select")

	c.updateCitySelector(context.Background())
	html := c.citySelector.Get("innerHTML").String()
	want := `<option value="city1path">city1</option><option value="city2path">city2</option>`
	if html != want {
		t.Errorf("%v != %v", html, want)
	}

	// Call again to make sure contents get cleared every time.
	client.EXPECT().Cities(
		gomock.Any(), // expect any value for first parameter
		gomock.Any(), // expect any value for second parameter
	).Return(&rpc.CitiesResponse{Names: []string{"city3", "city4"}, Paths: []string{"city3path", "city4path"}}, nil)

	c.updateCitySelector(context.Background())
	html = c.citySelector.Get("innerHTML").String()
	want = `<option value="city3path">city3</option><option value="city4path">city4</option>`
	if html != want {
		t.Errorf("%v != %v", html, want)
	}
}

func TestImpactTypeSelector(t *testing.T) {
	c := &CityAQ{
		doc: js.Global().Get("document"),
	}
	c.impactTypeSelector = c.doc.Call("createElement", "select")

	c.updateImpactTypeSelector()
	html := c.impactTypeSelector.Get("innerHTML").String()
	want := `<option value="1">Emissions</option>`
	if html != want {
		t.Errorf("%v != %v", html, want)
	}
}

func TestSourceTypeSelector(t *testing.T) {
	c := &CityAQ{
		doc: js.Global().Get("document"),
	}
	c.sourceTypeSelector = c.doc.Call("createElement", "select")

	c.updateSourceTypeSelector()
	html := c.sourceTypeSelector.Get("innerHTML").String()
	want := `<option value="roads">roads</option>`
	if html != want {
		t.Errorf("%v != %v", html, want)
	}
}

func TestEmissionSelector(t *testing.T) {
	c := &CityAQ{
		doc: js.Global().Get("document"),
	}
	c.emissionSelector = c.doc.Call("createElement", "select")

	c.updateEmissionSelector()
	html := c.emissionSelector.Get("innerHTML").String()
	want := `<option value="1">PM2_5</option><option value="2">NH3</option><option value="3">NOx</option><option value="4">SOx</option><option value="5">VOC</option>`
	if html != want {
		t.Errorf("%v != %v", html, want)
	}
}

func changeSelector(selector js.Value, index int) {
	selector.Set("selectedIndex", index)
}

func TestSelectors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	client := caqmock.NewMockCityAQClient(mockCtrl)

	client.EXPECT().Cities(
		gomock.Any(), // expect any value for first parameter
		gomock.Any(), // expect any value for second parameter
	).Return(&rpc.CitiesResponse{Names: []string{"city1", "city2"}, Paths: []string{"city1path", "city2path"}}, nil)

	c := &CityAQ{
		CityAQClient: client,
		doc:          js.Global().Get("document"),
	}
	c.citySelector = c.doc.Call("createElement", "select")
	c.impactTypeSelector = c.doc.Call("createElement", "select")
	c.emissionSelector = c.doc.Call("createElement", "select")
	c.sourceTypeSelector = c.doc.Call("createElement", "select")

	c.updateSelectors(context.Background())

	changeSelector(c.citySelector, 0)
	changeSelector(c.impactTypeSelector, 0)
	changeSelector(c.emissionSelector, 0)
	changeSelector(c.sourceTypeSelector, 0)

	sel, err := c.selectorValues()
	if err != nil {
		t.Fatal(err)
	}
	want := &selections{cityName: "city1", cityPath: "city1path", impactType: emission, emission: 1, sourceType: "roads"}

	if !reflect.DeepEqual(want, sel) {
		t.Errorf("%v != %v", sel, want)
	}
}
