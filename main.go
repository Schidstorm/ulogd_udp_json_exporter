package main

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schidstorm/ulogd_udp_json_exporter/pkg/exporter"
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
			metricsAddr, _ := cmd.Flags().GetString("metrics")
			group, _ := cmd.Flags().GetInt("group")
			go runMetricsServer(metricsAddr)

			log.Info().Msg("Starting Exporter")
			if err := exporter.RunExporter(group); err != nil {
				log.Fatal().Err(err).Msg("Error running exporter")
			}
		},
	}

	cmd.Flags().StringP("metrics", "m", ":8080", "HTTP address to expose metrics on")
	cmd.Flags().IntP("group", "g", 0, "nflog group to listen on")
	return cmd
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
