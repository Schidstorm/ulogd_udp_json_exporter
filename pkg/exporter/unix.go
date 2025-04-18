package exporter

// #cgo CFLAGS: -g -Wall
// #include <netdb.h>
// #include <stdlib.h>
// #include <unistd.h>
/*

typedef struct servent servent_t;
*/
import "C"
import (
	"strconv"
	"unsafe"
)

var serverventBUfferSize = C.sysconf(C._SC_GETPW_R_SIZE_MAX)
var cachedServices = make(map[protoPort]string)
var cachedProtos = make(map[int32]string)

type protoPort int32

func (pp protoPort) Proto() int32 {
	return int32(pp) >> 16
}
func (pp protoPort) Port() int32 {
	return int32(pp) & 0xFFFF
}
func ProtoPort(proto, port int32) protoPort {
	return protoPort((proto << 16) | port)
}

func GetProtoAndService(port int32, proto int32) (protoName string, serviceName string) {
	if name, ok := cachedProtos[proto]; ok {
		protoName = name
	} else {
		protoName = getProtoByNumberUncached(int(proto))
		cachedProtos[proto] = protoName
	}

	pp := ProtoPort(proto, port)
	if name, ok := cachedServices[pp]; ok {
		serviceName = name
	} else {
		serviceName = getServiceByPortUncached(int(port), protoName)
		cachedServices[pp] = serviceName
	}

	return protoName, serviceName
}

func getServiceByPortUncached(port int, proto string) string {
	cProto := C.CString(proto)
	defer C.free(unsafe.Pointer(cProto))

	var servent C.struct_servent
	var result *C.struct_servent
	bufSize := C.size_t(4096)
	buf := C.malloc(bufSize)
	defer C.free(buf)

	errno := C.getservbyport_r(
		C.int(C.htons(C.ushort(port))),
		cProto,
		&servent,
		(*C.char)(buf),
		bufSize,
		&result,
	)

	if errno != 0 || result == nil {
		return strconv.Itoa(port)
	}

	return C.GoString(result.s_name)
}

func getProtoByNumberUncached(proto int) string {
	protoent := C.getprotobynumber(C.int(proto))
	if protoent == nil {
		return strconv.Itoa(proto)
	}

	if protoent.p_name == nil {
		return strconv.Itoa(proto)
	}

	name := C.GoString(protoent.p_name)
	if name == "" {
		return strconv.Itoa(proto)
	}

	return name
}
