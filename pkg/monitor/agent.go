package monitor

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/schidstorm/ulogd_monitor/pkg/nflog"
	"github.com/schidstorm/ulogd_monitor/pkg/packet"
)

var hostname, _ = os.Hostname()

type AgentConfig struct {
	GroupId        int
	RemoteWriteUrl string
}

func RunAgent(cfg AgentConfig) error {
	agent := NewAgent(cfg)
	return agent.Start()
}

type Agent struct {
	Config      AgentConfig
	PacketQueue PacketQueue
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
	errChan := createErrChannel()
	defer errChan.close()

	go func() {
		err := a.send()
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

func (a *Agent) send() error {
	packetBuffer := make([]packet.Metricer, 128)

	for {
		n := takeAtMost(&a.PacketQueue, packetBuffer)
		if n == 0 {
			continue
		}

		RemoteWrite(a.Config.RemoteWriteUrl, packetBuffer[:n], hostname)
	}
}

func (a *Agent) attachNflog() error {
	nf := nflog.NewNfLog(a.Config.GroupId)
	return nf.Start(func(packet *packet.Packet) {
		a.PacketQueue.Enqueue(packet)
	})
}

func takeAtMost(queue *PacketQueue, out []packet.Metricer) int {
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
