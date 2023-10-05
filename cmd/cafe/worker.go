package main

import (
	"log"
	"time"

	"github.com/spf13/cobra"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"go.temporal.io/sdk/worker"

	"github.com/temporalio/temporal-cafe/workflows"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run worker",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := client.Dial(client.Options{
			MetricsHandler: sdktally.NewMetricsHandler(newPrometheusScope(prometheus.Configuration{
				ListenAddress: "0.0.0.0:9092",
				TimerType:     "histogram",
			})),
		})
		if err != nil {
			log.Fatalf("client error: %v", err)
		}
		defer c.Close()

		w := worker.New(c, "cafe", worker.Options{})

		w.RegisterWorkflow(workflows.Order)
		w.RegisterWorkflow(workflows.KitchenOrder)
		w.RegisterWorkflow(workflows.BaristaOrder)

		err = w.Run(worker.InterruptCh())
		if err != nil {
			log.Fatalf("worker exited: %v", err)
		}
	},
}

func newPrometheusScope(c prometheus.Configuration) tally.Scope {
	reporter, err := c.NewReporter(
		prometheus.ConfigurationOptions{
			Registry: prom.NewRegistry(),
			OnError: func(err error) {
				log.Println("error in prometheus reporter", err)
			},
		},
	)
	if err != nil {
		log.Fatalln("error creating prometheus reporter", err)
	}

	scopeOpts := tally.ScopeOptions{
		CachedReporter:  reporter,
		Separator:       prometheus.DefaultSeparator,
		SanitizeOptions: &sdktally.PrometheusSanitizeOptions,
	}
	scope, _ := tally.NewRootScope(scopeOpts, time.Second)
	scope = sdktally.NewPrometheusNamingScope(scope)

	return scope
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
