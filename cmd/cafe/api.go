package main

import (
	"net/http"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/temporalio/temporal-cafe/api"
	"go.temporal.io/sdk/client"
)

// apiCmd represents the api command
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Run API Server",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Dial(client.Options{})
		if err != nil {
			return err
		}
		defer c.Close()

		srv := &http.Server{
			Handler: api.Router(c),
			Addr:    "0.0.0.0:8084",
		}

		errCh := make(chan error, 1)
		go func() { errCh <- srv.ListenAndServe() }()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)

		select {
		case <-sigCh:
			srv.Close()
		case err = <-errCh:
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)
}
