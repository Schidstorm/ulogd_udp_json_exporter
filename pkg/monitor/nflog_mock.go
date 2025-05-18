package monitor

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/schidstorm/ulogd_monitor/pkg/nflog"
)

func runNflogMock(packetQueue PacketQueue) {
	log.Info().Msg("running nflog mock")

	var counter uint
	for {
		packet := nflog.NFLogPacket{
			Family:     2,
			Protocol:   6,
			PayloadLen: 100,
			Prefix:     nil,
			Indev:      "eth0",
			Outdev:     "eth1",
			Network: &nflog.NfLogPacketNetwork{
				SrcIp:    []byte{192, 168, 1, 1},
				DestIp:   []byte{192, 168, 1, 2},
				Protocol: 6,
				Transport: &nflog.NFLogTransportPacket{
					SrcPort:  80,
					DestPort: int(counter % 65535),
				},
			},
		}

		counter++
		packetQueue.Enqueue(Packet{
			NflogPacket: packet,
			Hostname:    "localhost",
		})

		time.Sleep(1 * time.Second)
	}
}
