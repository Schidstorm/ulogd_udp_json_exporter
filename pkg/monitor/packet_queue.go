package monitor

const packetQueueSize = 128

type PacketQueue struct {
	packetQueue chan Packet
}

func CreatePacketQueue() PacketQueue {
	return PacketQueue{
		packetQueue: make(chan Packet, packetQueueSize),
	}
}

func (pq PacketQueue) Enqueue(packet Packet) {
	select {
	case pq.packetQueue <- packet:
	default:
	}
}

func (pq PacketQueue) Dequeue() Packet {
	return <-pq.packetQueue
}
