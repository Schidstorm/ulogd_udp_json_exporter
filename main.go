package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schidstorm/ulogd_udp_json_exporter/pkg/exporter"
	"github.com/schidstorm/ulogd_udp_json_exporter/pkg/nflog"
	"github.com/spf13/cobra"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cmd := rootCommand()
	cmd.AddCommand(metricsCommand())
	if err := cmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Error executing command")
	}
}

func rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ulogd_udp_json_exporter",
		Short: "Run the ulogd UDP JSON exporter",
		Run: func(cmd *cobra.Command, args []string) {
			listenAddr, _ := cmd.Flags().GetString("listen")
			metricsAddr, _ := cmd.Flags().GetString("metrics")

			log.Info().Msg("Starting nflog")
			nf := initNflog()
			defer nf.Close()

			log.Info().Str("metrics", metricsAddr).Msg("Starting metrics server")
			go runMetricsServer(metricsAddr)

			log.Info().Str("address", listenAddr).Msg("Starting UDP listener")
			if err := exporter.RunExporter(listenAddr); err != nil {
				log.Fatal().Err(err).Msg("Error running exporter")
			}
		},
	}

	cmd.Flags().StringP("listen", "l", ":9999", "UDP address to listen on")
	cmd.Flags().StringP("metrics", "m", ":8080", "HTTP address to expose metrics on")
	return cmd
}

func initNflog() io.Closer {
	nf := nflog.NewNfLog(0)
	if err := nf.Start(); err != nil {
		log.Fatal().Err(err).Msg("Error starting nflog")
	}

	return nf
}

func runMetricsServer(metricsAddr string) {
	http.Handle("/metrics", promhttp.Handler())
	log.Info().Str("address", metricsAddr).Msg("Starting metrics server")
	if err := http.ListenAndServe(metricsAddr, nil); err != nil {
		log.Fatal().Err(err).Msg("Error starting metrics server")
	}
}

func metricsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Print all metrics",
		Run: func(cmd *cobra.Command, args []string) {
			if collector, ok := prometheus.DefaultGatherer.(prometheus.Collector); ok {
				descChannel := make(chan *prometheus.Desc)
				go func() {
					for desc := range descChannel {
						fmt.Println(desc.String())
					}
				}()

				collector.Describe(descChannel)
			}
		},
	}

	return cmd
}
