package monitor

import "github.com/schidstorm/ulogd_monitor/pkg/packet"

const packetQueueSize = 128

type PacketQueue struct {
	packetQueue chan *packet.Packet
}

func CreatePacketQueue() PacketQueue {
	return PacketQueue{
		packetQueue: make(chan *packet.Packet, packetQueueSize),
	}
}

func (pq PacketQueue) Enqueue(packet *packet.Packet) {
	select {
	case pq.packetQueue <- packet:
	default:
	}
}

func (pq PacketQueue) Dequeue() *packet.Packet {
	return <-pq.packetQueue
}
