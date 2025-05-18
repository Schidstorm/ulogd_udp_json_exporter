package monitor

import (
	context "context"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/schidstorm/ulogd_monitor/pkg/nflog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var hostname, _ = os.Hostname()

type AgentConfig struct {
	GroupId    int
	ServerAddr string
	IsDevMode  bool
}

func RunAgent(cfg AgentConfig) error {
	conn, err := grpc.NewClient(cfg.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to server")
	}
	defer conn.Close()
	c := NewMonitorClient(conn)
	packetQueue := CreatePacketQueue()

	go func() {
		for {
			packet := packetQueue.Dequeue()
			// Contact the server and print out its response.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_, err := c.SendPacket(ctx, &SendPacketRequest{
				Packet: packetToProto(packet.NflogPacket),
				Metadata: &PacketMetadata{
					Hostname: packet.Hostname,
				},
			})
			cancel()

			if err != nil {
				log.Info().Err(err).Msg("wailed to send aocket to agent_receiver")
				continue
			}
		}
	}()

	if cfg.IsDevMode {
		runNflogMock(packetQueue)
		return nil
	} else {
		// Create a new NfLog instance
		nf := nflog.NewNfLog(cfg.GroupId)
		return nf.Start(func(packet nflog.NFLogPacket) {
			packetQueue.Enqueue(Packet{
				NflogPacket: packet,
				Hostname:    hostname,
			})
		})
	}
}
