package nflog

import "sync"

type UserCallback struct {
	id       uint32
	callback NfLogCallback
}

var userCallbacks []UserCallback
var mutex sync.Mutex
var id uint32

func registerUserCallback(callback NfLogCallback) uint32 {
	mutex.Lock()
	defer mutex.Unlock()

	id += 1
	userCallbacks = append(userCallbacks, UserCallback{id: id, callback: callback})
	return id
}

func callCallback(id uint32, packet NFLogPacket) {
	mutex.Lock()
	defer mutex.Unlock()

	for _, cb := range userCallbacks {
		if cb.id == id {
			cb.callback(packet)
			return
		}
	}
}
