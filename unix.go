package main

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

func getServiceByPort(port int, proto string) string {
	cProto := C.CString(proto)
	defer C.free(unsafe.Pointer(cProto))

	var servent C.struct_servent
	var result *C.struct_servent
	bufSize := C.size_t(4096)
	buf := C.malloc(bufSize)
	defer C.free(buf)

	errno := C.getservbyport_r(
		C.int(C.htons(C.ushort(port))), // important: htons!
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

func getProtoByNumber(proto int) string {
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
