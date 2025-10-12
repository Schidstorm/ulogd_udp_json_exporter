package monitor

import "github.com/schidstorm/ulogd_monitor/pkg/pb"

const packetQueueSize = 128

type PacketQueue struct {
	packetQueue chan *pb.Packet
}

func CreatePacketQueue() PacketQueue {
	return PacketQueue{
		packetQueue: make(chan *pb.Packet, packetQueueSize),
	}
}

func (pq PacketQueue) Enqueue(packet *pb.Packet) {
	select {
	case pq.packetQueue <- packet:
	default:
	}
}

func (pq PacketQueue) Dequeue() *pb.Packet {
	return <-pq.packetQueue
}
