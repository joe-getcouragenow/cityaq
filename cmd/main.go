package main

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/ctessum/cityaq"
	"github.com/sirupsen/logrus"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"google.golang.org/grpc/grpclog"
)

var logger *logrus.Logger

func init() {
	logger = logrus.StandardLogger()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
		DisableSorting:  true,
	})
	// Should only be done from init functions
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(logger.Out, logger.Out, logger.Out))
}

func main() {
	c := &cityaq.CityAQ{
		CityGeomDir: "../testdata/cities",
		SpatialConfig: aeputil.SpatialConfig{
			SrgSpec:       "srgspec_osm.json",
			SrgSpecType:   "OSM",
			SCCExactMatch: true,
			GridRef:       []string{"../testdata/gridref_osm.txt"},
			OutputSR:      "+proj=longlat",
			InputSR:       "+proj=longlat",
		},
	}

	srv := cityaq.NewGRPCServer(c, "")
	srv.Log = logger

	addr := "localhost:10000"
	httpsSrv := &http.Server{
		Addr:    addr,
		Handler: srv,
		// Some security settings
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		TLSConfig: &tls.Config{
			PreferServerCipherSuites: true,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
		},
	}

	logger.Info("Serving on https://" + addr)
	logger.Fatal(httpsSrv.ListenAndServeTLS("./insecure/cert.pem", "./insecure/key.pem"))
}
