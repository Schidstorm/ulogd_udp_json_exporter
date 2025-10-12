package monitor

import (
	"sync"

	"github.com/schidstorm/ulogd_monitor/pkg/pb"
)

type QueueHandler func(packet *pb.Packet)
type QueueHandlerId int

type QueueReadMany struct {
	attached map[QueueHandlerId]PacketQueue
	mutex    sync.Mutex
	nextId   QueueHandlerId
}

func (q *QueueReadMany) Start(queue PacketQueue) error {
	q.attached = make(map[QueueHandlerId]PacketQueue)

	for {
		packet := queue.Dequeue()
		q.distribute(packet)
	}
}

func (q *QueueReadMany) distribute(packet *pb.Packet) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for _, handler := range q.attached {
		handler.Enqueue(packet)
	}
}

func (q *QueueReadMany) Attach() (PacketQueue, QueueHandlerId) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	id := q.nextId
	q.nextId++
	q.attached[id] = CreatePacketQueue()
	return q.attached[id], id
}

func (q *QueueReadMany) Detach(id QueueHandlerId) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	delete(q.attached, id)
}
