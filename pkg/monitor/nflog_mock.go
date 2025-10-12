package monitor

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/schidstorm/ulogd_monitor/pkg/pb"
)

func runNflogMock(packetQueue PacketQueue) {
	log.Info().Msg("running nflog mock")

	var counter uint
	for {
		p := &pb.Packet{
			Metadata: &pb.PacketMetadata{
				Timestamp: uint64(time.Now().Unix()),
				Hostname:  "mocked-host",
			},
			Layers: []*pb.Layer{
				{
					Layer: &pb.Layer_Ethernet{
						Ethernet: &pb.LayerEthernet{
							SrcMac:    "00:11:22:33:44:55",
							DestMac:   "66:77:88:99:AA:BB",
							Ethertype: "IPv4",
						},
					},
				},
				{
					Layer: &pb.Layer_Ipv4{
						Ipv4: &pb.LayerIPv4{
							SrcIp:    "192.168.1.1",
							DestIp:   "192.168.1.2",
							Protocol: "TCP",
						},
					},
				},
				{
					Layer: &pb.Layer_Tcp{
						Tcp: &pb.LayerTCP{
							SrcPort:  80,
							DestPort: uint32(counter % 65535),
						},
					},
				},
			},
		}

		counter++
		packetQueue.Enqueue(p)

		time.Sleep(1 * time.Second)
	}
}
