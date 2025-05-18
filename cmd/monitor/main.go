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

	cmd.PersistentFlags().BoolP("dev", "d", false, "run in dev mode")
	cmd.PersistentFlags().StringP("log-level", "l", "error", "loglevel")

	cmd.AddCommand(serverCommand())
	cmd.AddCommand(agentCommand())

	return cmd
}

func serverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the ulogd monitor server",
		Run: func(cmd *cobra.Command, args []string) {
			httpAddr, _ := cmd.Flags().GetString("http.addr")
			grpcAddr, _ := cmd.Flags().GetString("grpc.addr")
			devMode, _ := cmd.Flags().GetBool("dev")
			runServer(httpAddr, grpcAddr, devMode)
		},
	}

	cmd.Flags().StringP("http.addr", "a", ":8080", "HTTP address of web server")
	cmd.Flags().StringP("grpc.addr", "g", ":8081", "GRPC address of agent receiver")

	return cmd
}

func runServer(httpAddr, grpcAddr string, devMode bool) {
	server := monitor.NewServer()
	server.Start(monitor.ServerConfig{
		HttpListenAddr: httpAddr,
		GrpcListenAddr: grpcAddr,
		IsDev:          devMode,
	})
}

func agentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Run the ulogd monitor agent",
		Run: func(cmd *cobra.Command, args []string) {
			addr, _ := cmd.Flags().GetString("grpc.addr")
			group, _ := cmd.Flags().GetInt("group")
			devMode, _ := cmd.Flags().GetBool("dev")
			runAgent(group, addr, devMode)
		},
	}

	cmd.Flags().StringP("grpc.addr", "a", "localhost:8081", "GRPC address of agent receiver")
	cmd.Flags().IntP("group", "g", 0, "nflog group to listen on")
	return cmd
}

func runAgent(group int, addr string, devMode bool) {
	err := monitor.RunAgent(monitor.AgentConfig{
		GroupId:    group,
		ServerAddr: addr,
		IsDevMode:  devMode,
	})
	fmt.Println(err)
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting agent")
	}
}
