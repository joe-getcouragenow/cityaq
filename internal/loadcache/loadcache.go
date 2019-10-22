package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	rpc "github.com/ctessum/cityaq/cityaqrpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// Set up a client to connect to https://inmap.run.
	ctx := context.Background()
	conn, err := grpc.Dial("inmap.run:443", grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	check(err)
	client := rpc.NewCityAQClient(conn)

	sourceTypes := []string{"electric_gen_egugrid", "population", "residential",
		"commercial", "industrial", "builtup", "roadways", "railways", "waterways",
		"bus_routes", "airports", "agricultural"}

	cities := []string{
		"Guadalajara", "Autonomous City of Buenos Aires", "City of Johannesburg Metropolitan Municipality",
		"Accra Metropolitan", "Chennai", "Addis Ababa", "Seattle", "New York", "Bengaluru", "Washington",
	}

	allCities, err := client.Cities(ctx, &rpc.CitiesRequest{})
	check(err)
	for _, n := range allCities.Names {
		cities = append(cities, n)
	}

	c := make(chan query)
	var wg sync.WaitGroup
	const nprocs = 2
	wg.Add(nprocs)
	for i := 0; i < nprocs; i++ {
		go func() {
			runQuery(c, &wg)
		}()
	}

	var i int
	total := len(cities) * len(sourceTypes)
	for _, name := range cities {
		for _, sourceType := range sourceTypes {
			c <- query{
				name:       name,
				sourceType: sourceType,
				i:          i,
				total:      total,
			}
			i++
		}
	}
	close(c)
	wg.Wait()
}

type query struct {
	i, total   int
	name       string
	sourceType string
}

func runQuery(c chan query, wg *sync.WaitGroup) {
	ctx := context.Background()
	conn, err := grpc.Dial("inmap.run:443", grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	check(err)
	client := rpc.NewCityAQClient(conn)
	for q := range c {
		log.Printf("%d/%d; %s; %s", q.i, q.total, q.name, q.sourceType)
		bkf := backoff.NewConstantBackOff(30 * time.Second)
		check(backoff.RetryNotify(
			func() error {
				_, err := client.EmissionsMap(ctx, &rpc.EmissionsMapRequest{
					CityName:   q.name,
					SourceType: q.sourceType,
					Emission:   rpc.Emission_PM2_5,
				})
				if err != nil && (strings.Contains(err.Error(), "no emissions") || strings.Contains(err.Error(), "larger than max")) {
					fmt.Println(err)
					return nil
				}
				return err
			},
			bkf,
			func(err error, d time.Duration) {
				log.Printf("%v: retrying in %v", err, d)
			},
		))
	}
	wg.Done()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
