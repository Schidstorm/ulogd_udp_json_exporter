package main

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schidstorm/ulogd_monitor/pkg/monitor"
	"github.com/spf13/cobra"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	cmd := rootCommand()
	if err := cmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Error executing command")
	}
}

func rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ulogd_monitor",
		Short: "Run the ulogd UDP JSON exporter",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logLevelStr, _ := cmd.Flags().GetString("log-level")
			logLevel, err := zerolog.ParseLevel(logLevelStr)
			if err != nil {
				log.Fatal().Err(err).Msg("Invalid log level")
			}

			zerolog.SetGlobalLevel(logLevel)
		},
	}

	cmd.PersistentFlags().StringP("log-level", "l", "error", "loglevel")

	cmd.AddCommand(agentCommand())

	return cmd
}

func agentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Run the ulogd monitor agent",
		Run: func(cmd *cobra.Command, args []string) {
			url, _ := cmd.Flags().GetString("metrics.url")
			group, _ := cmd.Flags().GetInt("group")
			runAgent(group, url)
		},
	}

	cmd.Flags().StringP("metrics.url", "a", "https://metrics.schidlow.ski/api/v1/write", "URL of the remote write endpoint")
	cmd.Flags().IntP("group", "g", 0, "nflog group to listen on")
	return cmd
}

func runAgent(group int, remoteWriteUrl string) {
	err := monitor.RunAgent(monitor.AgentConfig{
		GroupId:        group,
		RemoteWriteUrl: remoteWriteUrl,
	})
	fmt.Println(err)
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting agent")
	}
}
