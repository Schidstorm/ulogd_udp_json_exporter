package nflog

import (
	"sync"

	"github.com/schidstorm/ulogd_monitor/pkg/pb"
)

type CallbackFunc func(packet *pb.Packet)

type UserCallback struct {
	id       uint32
	callback CallbackFunc
}

var userCallbacks []UserCallback
var mutex sync.Mutex
var id uint32

func registerUserCallback(callback CallbackFunc) uint32 {
	mutex.Lock()
	defer mutex.Unlock()

	id += 1
	userCallbacks = append(userCallbacks, UserCallback{id: id, callback: callback})
	return id
}

func callCallback(id uint32, packet *pb.Packet) {
	mutex.Lock()
	defer mutex.Unlock()

	for _, cb := range userCallbacks {
		if cb.id == id {
			cb.callback(packet)
			return
		}
	}
}
