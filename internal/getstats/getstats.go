package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	rpc "github.com/ctessum/cityaq/cityaqrpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	sourceTypes := []string{
		"railways", "electric_gen_egugrid", "population_gpw", "residential",
		"commercial", "industrial", "builtup",
		"roadways_motorway", "roadways_trunk", "roadways_primary",
		"roadways_secondary", "roadways_tertiary",
		"roadways", "waterways",
		"bus_routes", "airports", "agricultural",
	}

	cities := []string{
		"Guadalajara",
		"Autonomous City of Buenos Aires",
		"City of Johannesburg Metropolitan Municipality",
		//"Accra Metropolitan",
		"Chennai",
		"Addis Ababa",
		"Seattle",
		"New York",
		"Bengaluru",
		"Washington",
		"Fuzhou City",
		"Kolkata",
		"Qingdao City",
		"Medellín",
		"Quito",
		"Lima",
		"Lagos",
		"Ho Chi Minh City",
		"Quezon City",
	}

	ctx := context.Background()
	conn, err := grpc.Dial("inmap.run:443", grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	check(err)
	client := rpc.NewCityAQClient(conn)

	o, err := os.Create("cityaq_stats.csv")
	check(err)
	w := csv.NewWriter(o)
	check(w.Write([]string{"city", "source", "emission", "metric", "domain", "value"}))

	for _, city := range cities {
		for _, sourceType := range sourceTypes {
			for emission := 1; emission <= 5; emission++ {
				log.Printf("%s; %s; %s", city, sourceType, rpc.Emission(emission))
				bkf := backoff.NewConstantBackOff(30 * time.Second)
				check(backoff.RetryNotify(
					func() error {
						impacts, err := client.ImpactSummary(ctx, &rpc.ImpactSummaryRequest{
							CityName:   city,
							SourceType: sourceType,
							Emission:   rpc.Emission(emission),
						})
						if err != nil {
							return err
						}
						check(w.Write([]string{
							city,
							sourceType,
							rpc.Emission(emission).String(),
							"population",
							"city",
							fmt.Sprint(impacts.CityPopulation),
						}))
						check(w.Write([]string{
							city,
							sourceType,
							rpc.Emission(emission).String(),
							"population",
							"total",
							fmt.Sprint(impacts.Population),
						}))
						check(w.Write([]string{
							city,
							sourceType,
							rpc.Emission(emission).String(),
							"exposure",
							"city",
							fmt.Sprint(impacts.CityExposure),
						}))
						check(w.Write([]string{
							city,
							sourceType,
							rpc.Emission(emission).String(),
							"exposure",
							"total",
							fmt.Sprint(impacts.TotalExposure),
						}))
						check(w.Write([]string{
							city,
							sourceType,
							rpc.Emission(emission).String(),
							"iF",
							"city",
							fmt.Sprint(impacts.CityIF),
						}))
						check(w.Write([]string{
							city,
							sourceType,
							rpc.Emission(emission).String(),
							"iF",
							"total",
							fmt.Sprint(impacts.TotalIF),
						}))
						fmt.Println(impacts)
						return nil
					},
					bkf,
					func(err error, d time.Duration) {
						log.Printf("%v: retrying in %v", err, d)
					},
				))
			}
		}
	}
	w.Flush()
	o.Close()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
