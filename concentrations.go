package cityaq

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	rpc "github.com/ctessum/cityaq/cityaqrpc"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/requestcache/v3"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spatialmodel/inmap/inmaputil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GriddedConcentrations returns PM2.5 concentrations calculated by the InMAP
// air quality model.
func (c *CityAQ) GriddedConcentrations(ctx context.Context, req *rpc.GriddedConcentrationsRequest) (*rpc.GriddedConcentrationsResponse, error) {
	var err error
	c.cloudSetupOnce.Do(func() {
		err = c.cloudSetup()
	})
	if err != nil {
		return nil, err
	}
	c.setupCache()

	job := &concentrationJob{
		c:          c,
		CityName:   req.CityName,
		SourceType: req.SourceType,
	}

	inmapReq := c.cache.NewRequest(ctx, job)
	var result inmapResult
	if err := inmapReq.Result(&result); err != nil {
		return nil, err
	}

	o := &rpc.GriddedConcentrationsResponse{
		Polygons: polygonsToRPC(result.Grid),
	}
	switch req.Emission {
	case rpc.Emission_PM2_5:
		o.Concentrations = result.PrimaryPM25
	case rpc.Emission_NH3:
		o.Concentrations = result.PNH4
	case rpc.Emission_NOx:
		o.Concentrations = result.PNO3
	case rpc.Emission_SOx:
		o.Concentrations = result.PSO4
	case rpc.Emission_VOC:
		o.Concentrations = result.SOA
	default:
		return nil, fmt.Errorf("cityaq: invalid emission type %s", req.Emission)
	}
	return o, nil
}

func (c *CityAQ) cloudSetup() error {
	cfg := inmaputil.InitializeConfig()

	if os.ExpandEnv("${KUBERNETES_SERVICE_HOST}") == "" {
		log.Println("NOT IN KUBERNETES CLUSTER")
		var err error
		c.inmapClient, err = cloud.NewFakeClient(nil, func(b []byte, err error) {
			fmt.Println(string(b))
			if err != nil {
				fmt.Println("ERROR", err)
			}
		}, c.CacheLoc, cfg.Root, cfg.Viper, cfg.InputFiles(), cfg.OutputFiles())
		if err != nil {
			return fmt.Errorf("failed to initialize fake InMAP server: %w", err)
		}
		return nil
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to load in-cluster Kubernetes configuration: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to initialize Kubernetes: %w", err)
	}

	c.inmapClient, err = cloud.NewClient(clientset, cfg.Root, cfg.Viper, c.CacheLoc, cfg.InputFiles(), cfg.OutputFiles())
	if err != nil {
		return fmt.Errorf("failed to initialize InMAP server: %w", err)
	}

	return nil
}

type concentrationJob struct {
	c          *CityAQ
	CityName   string
	SourceType string
}

func (j *concentrationJob) Key() string {
	return fmt.Sprintf("concentration_%s_%s", j.CityName, j.SourceType)
}

func (j *concentrationJob) Run(ctx context.Context, result requestcache.Result) error {
	shpFile, err := j.emisToShp(ctx)
	if err != nil {
		return err
	}

	cfg := inmaputil.InitializeConfig()
	cfg.SetConfigFile(j.c.InMAPConfigFile)
	if err := cfg.ReadInConfig(); err != nil {
		return fmt.Errorf("cityaq: problem reading InMAP configuration file: %v", err)
	}

	ctx = context.WithValue(ctx, "user", "cityaq_user")

	cfg.Set("EmissionsShapefiles", []string{shpFile})
	cfg.Set("job_name", j.Key())
	cfg.Set("cmds", []string{"run", "steady"})

	bnds, err := j.c.EmissionsGridBounds(ctx, &rpc.EmissionsGridBoundsRequest{
		CityName:   j.CityName,
		SourceType: j.SourceType,
	})
	if err != nil {
		return err
	}
	center := geom.Point{
		X: (bnds.Max.X + bnds.Min.X) / 2,
		Y: (bnds.Max.Y + bnds.Min.Y) / 2,
	}

	// Set lower-left corner of grid so that the
	// city is in its center, while still overlapping
	// the underlying CTM grid.
	vgc, err := inmaputil.VarGridConfig(cfg.Viper)
	if err != nil {
		return err
	}
	xo := vgc.VariableGridXo
	yo := vgc.VariableGridYo
	nx := vgc.Xnests[0]
	ny := vgc.Ynests[0]
	dx := vgc.VariableGridDx
	dy := vgc.VariableGridDy
	xo = math.Max(xo, roundUnit(center.X-float64(nx)*dx/2, dx))
	yo = math.Max(yo, roundUnit(center.Y-float64(ny)*dy/2, dy))
	cfg.Set("VarGrid.VariableGridXo", xo)
	cfg.Set("VarGrid.VariableGridYo", yo)
	if xo+dx*float64(nx) > 180 {
		nx = int((180 - xo) / dx)
	}
	if yo+dy*float64(ny) > 89.5 {
		ny = int((89.5 - yo) / dy)
	}
	vgc.Xnests[0] = nx
	vgc.Ynests[0] = ny
	cfg.Set("VarGrid.Xnests", vgc.Xnests)
	cfg.Set("VarGrid.Ynests", vgc.Ynests)

	in, err := cloud.JobSpec(
		cfg.Root, cfg.Viper,
		cfg.GetString("job_name"),
		cfg.GetStringSlice("cmds"),
		cfg.InputFiles(),
		int32(cfg.GetInt("memory_gb")),
	)
	if err != nil {
		return err
	}
	_, err = j.c.inmapClient.RunJob(ctx, in)
	if err != nil {
		return err
	}

	for {
		status, err := j.c.inmapClient.Status(ctx, &cloudrpc.JobName{
			Version: inmap.Version,
			Name:    j.Key(),
		})
		if err != nil {
			return err
		}
		if status.Status == cloudrpc.Status_Failed || status.Status == cloudrpc.Status_Missing {
			return fmt.Errorf("job %s error: %s, %s", j.Key(), status.Status, status.Message)
		} else if status.Status == cloudrpc.Status_Complete {
			break
		}
		time.Sleep(30 * time.Second)
	}

	output, err := j.c.inmapClient.Output(ctx, &cloudrpc.JobName{
		Version: inmap.Version,
		Name:    j.Key(),
	})
	if err != nil {
		return err
	}

	if err := inmapOutputToResult(output, result); err != nil {
		return err
	}
	if _, err := j.c.inmapClient.Delete(ctx, &cloudrpc.JobName{
		Version: inmap.Version,
		Name:    j.Key(),
	}); err != nil {
		return err
	}

	return nil
}

// roundUnit rounds a float to the nearest inverval of the
// given unit.
func roundUnit(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}

// emisToShp calculates the emissions associated with this job and
// saves them to a temporary shapefile.
func (j *concentrationJob) emisToShp(ctx context.Context) (string, error) {
	eReq := &rpc.GriddedEmissionsRequest{
		CityName:   j.CityName,
		SourceType: j.SourceType,
		Emission:   rpc.Emission_PM2_5,
	}
	emis, err := j.c.GriddedEmissions(ctx, eReq)
	if err != nil {
		return "", err
	}

	dir, err := ioutil.TempDir("", "cityaq_emissions")
	if err != nil {
		return "", err
	}
	file := filepath.Join(dir, "emissions.shp")
	type emisRecord struct {
		geom.Polygon
		PM2_5, VOC, NH3, NOx, SOx float64
	}
	e, err := shp.NewEncoder(file, emisRecord{})
	if err != nil {
		return "", err
	}
	for i, p := range emis.Polygons {
		v := emis.Emissions[i]
		err := e.Encode(&emisRecord{
			Polygon: rpcToGeom(p),
			PM2_5:   v,
			VOC:     v,
			NH3:     v,
			NOx:     v,
			SOx:     v,
		})
		if err != nil {
			return "", err
		}
	}
	e.Close()
	prjFile, err := os.Create(filepath.Join(dir, "emissions.prj"))
	if err != nil {
		return "", err
	}
	defer prjFile.Close()
	if _, err := fmt.Fprint(prjFile, `GEOGCS["GCS_WGS_1984",DATUM["D_WGS_1984",SPHEROID["WGS_1984",6378137,298.257223563]],PRIMEM["Greenwich",0],UNIT["Degree",0.017453292519943295]]`); err != nil {
		return "", err
	}
	return file, nil
}

type inmapResult struct {
	Grid          []geom.Polygon
	Population    []float64
	MortalityRate []float64
	PrimaryPM25   []float64
	SOA           []float64
	PNH4          []float64
	PNO3          []float64
	PSO4          []float64
}

type wrapInmapResult struct {
	Grid          []geom.Polygon
	Population    []float64
	MortalityRate []float64
	PrimaryPM25   []float64
	SOA           []float64
	PNH4          []float64
	PNO3          []float64
	PSO4          []float64
}

func (r *inmapResult) MarshalBinary() ([]byte, error) {
	w := wrapInmapResult{Grid: r.Grid, Population: r.Population,
		MortalityRate: r.MortalityRate, PrimaryPM25: r.PrimaryPM25,
		SOA: r.SOA, PNH4: r.PNH4, PNO3: r.PNO3, PSO4: r.PSO4}
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(w); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (r *inmapResult) UnmarshalBinary(b []byte) error {
	w := &wrapInmapResult{}
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(w); err != nil {
		return err
	}
	r.Grid = w.Grid
	r.Population = w.Population
	r.MortalityRate = w.MortalityRate
	r.PrimaryPM25 = w.PrimaryPM25
	r.SOA = w.SOA
	r.PNH4 = w.PNH4
	r.PNO3 = w.PNO3
	r.PSO4 = w.PSO4
	return nil
}

type inmapTempResult struct {
	geom.Polygon
	Population    float64 `shp:"pop"`
	MortalityRate float64 `shp:"mort"`
	PrimaryPM25   float64 `shp:"PrimPM25"`
	SOA           float64 `shp:"SOA"`
	PNH4          float64 `shp:"pNH4"`
	PNO3          float64 `shp:"pNO3"`
	PSO4          float64 `shp:"pSO4"`
}

func inmapOutputToResult(out *cloudrpc.JobOutput, result requestcache.Result) error {
	dir, err := ioutil.TempDir("", "cityaq_output")
	if err != nil {
		return err
	}
	for n, d := range out.Files {
		file := filepath.Join(dir, n)
		w, err := os.Create(file)
		if err != nil {
			return err
		}
		if _, err := w.Write(d); err != nil {
			return err
		}
		w.Close()
	}
	d, err := shp.NewDecoder(filepath.Join(dir, "OutputFile.shp"))
	if err != nil {
		return err
	}
	defer d.Close()
	o := result.(*inmapResult)
	o.Grid = make([]geom.Polygon, d.AttributeCount())
	o.Population = make([]float64, d.AttributeCount())
	o.MortalityRate = make([]float64, d.AttributeCount())
	o.PrimaryPM25 = make([]float64, d.AttributeCount())
	o.SOA = make([]float64, d.AttributeCount())
	o.PNH4 = make([]float64, d.AttributeCount())
	o.PNO3 = make([]float64, d.AttributeCount())
	o.PSO4 = make([]float64, d.AttributeCount())
	var i int
	for {
		var rec inmapTempResult
		if more := d.DecodeRow(&rec); !more {
			break
		}
		o.Grid[i] = rec.Polygon
		o.Population[i] = rec.Population
		o.MortalityRate[i] = rec.MortalityRate
		o.PrimaryPM25[i] = rec.PrimaryPM25
		o.SOA[i] = rec.SOA
		o.PNH4[i] = rec.PNH4
		o.PNO3[i] = rec.PNO3
		o.PSO4[i] = rec.PSO4
		i++
	}
	if err := d.Error(); err != nil {
		return err
	}
	return nil
}
