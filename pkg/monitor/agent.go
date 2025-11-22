package monitor

import (
	context "context"
	"io"
	"os"
	"sync"
	"time"

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

	go func() {
		for err := a.startStreamPackets(); err != nil; err = a.startStreamPackets() {
			log.Error().Err(err).Msg("Error in packet stream, reconnecting...")
			time.Sleep(time.Second)
		}
	}()

	if err := a.attachNflog(); err != nil {
		log.Fatal().Err(err).Msg("Failed to attach to nflog")
	}

	select {}
}

func (a *Agent) startStreamPackets() error {
	conn, err := grpc.NewClient(a.Config.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to server")
	}
	defer conn.Close()

	c := pb.NewMonitorClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := c.StreamPackets(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start packet stream")
	}
	defer client.CloseSend()

	errChan := createErrChannel()
	defer errChan.close()

	go func() {
		err := a.receive(client)
		if err == io.EOF {
			err = nil
		}

		errChan.send(err)
	}()

	go func() {
		err := a.send(client)
		if err == io.EOF {
			err = nil
		}

		errChan.send(err)
	}()

	return errChan.wait()
}

func createErrChannel() errChannel {
	return errChannel{
		ch: make(chan error),
	}
}

type errChannel struct {
	mutex  sync.Mutex
	closed bool
	ch     chan error
}

func (ec *errChannel) send(err error) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()

	if !ec.closed {
		select {
		case ec.ch <- err:
		default:
		}
	}
}

func (ec *errChannel) close() {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()

	if !ec.closed {
		close(ec.ch)
		ec.closed = true
	}
}

func (ec *errChannel) wait() error {
	return <-ec.ch
}

func (a *Agent) receive(client grpc.BidiStreamingClient[pb.StreamPacketsRequest, pb.StreamPacketsResponse]) error {
	for {
		resp, err := client.Recv()
		if err == io.EOF {
			return err
		}
		if err != nil {
			log.Error().Err(err).Msg("failed to receive response from agent_receiver")
			return err
		}

		if activateCommand := resp.GetActivateCommand(); activateCommand != nil {
			log.Info().Bool("active", activateCommand.Activate).Msg("Received activate command from server")
			a.active = activateCommand.Activate
		}
	}
}

func (a *Agent) send(client grpc.BidiStreamingClient[pb.StreamPacketsRequest, pb.StreamPacketsResponse]) error {
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
			return err
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
	if len(out) == 0 {
		panic("packetQueue size is 0")
	}

	// wait for at least one packet
	firstPacket := <-queue.packetQueue
	out[0] = firstPacket

	for i := 1; i < len(out); i++ {
		select {
		case packet := <-queue.packetQueue:
			out[i] = packet
		default:
			return i
		}
	}
	return len(out)
}
