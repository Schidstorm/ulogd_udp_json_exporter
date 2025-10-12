package monitor

import (
	context "context"
	"io"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/schidstorm/ulogd_monitor/pkg/nflog"
	"github.com/schidstorm/ulogd_monitor/pkg/pb"
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
	agent := NewAgent(cfg)
	return agent.Start()
}

type Agent struct {
	Config      AgentConfig
	PacketQueue PacketQueue
	active      bool
}

func NewAgent(cfg AgentConfig) *Agent {
	return &Agent{
		Config:      cfg,
		PacketQueue: CreatePacketQueue(),
	}
}

func (a *Agent) Start() error {
	conn, err := grpc.NewClient(a.Config.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to server")
	}
	defer conn.Close()

	c := pb.NewMonitorClient(conn)

	client, err := c.StreamPackets(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start packet stream")
	}

	go a.receive(client)
	go a.send(client)

	if err := a.attachNflog(); err != nil {
		log.Fatal().Err(err).Msg("Failed to attach to nflog")
	}

	select {}
}

func (a *Agent) receive(client grpc.BidiStreamingClient[pb.StreamPacketsRequest, pb.StreamPacketsResponse]) {
	for {
		resp, err := client.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Error().Err(err).Msg("failed to receive response from agent_receiver")
			continue
		}

		if activateCommand := resp.GetActivateCommand(); activateCommand != nil {
			log.Info().Bool("active", activateCommand.Activate).Msg("Received activate command from server")
			a.active = activateCommand.Activate
		}
	}
}

func (a *Agent) send(client grpc.BidiStreamingClient[pb.StreamPacketsRequest, pb.StreamPacketsResponse]) {
	packetBuffer := make([]*pb.Packet, 32)

	for {
		n := takeAtMost(&a.PacketQueue, packetBuffer)
		if n == 0 || !a.active {
			continue
		}

		for i := range n {
			packetBuffer[i].Metadata.Hostname = hostname
		}

		err := client.Send(&pb.StreamPacketsRequest{
			Packets: packetBuffer[:n],
		})

		if err != nil {
			log.Info().Err(err).Msg("failed to send packet to agent_receiver")
			continue
		}
	}
}

func (a *Agent) attachNflog() error {
	if a.Config.IsDevMode {
		runNflogMock(a.PacketQueue)
		return nil
	} else {
		nf := nflog.NewNfLog(a.Config.GroupId)
		return nf.Start(func(packet *pb.Packet) {
			a.PacketQueue.Enqueue(packet)
		})
	}
}

func takeAtMost(queue *PacketQueue, out []*pb.Packet) int {
	for i := range len(out) {
		select {
		case packet := <-queue.packetQueue:
			out[i] = packet
		default:
			return i
		}
	}
	return len(out)
}
