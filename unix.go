package main

// #cgo CFLAGS: -g -Wall
// #include <stdlib.h>
// #include <netdb.h>
import "C"
import (
	"strconv"
	"unsafe"
)

func getServiceByPort(port int, proto string) string {
	cProto := C.CString(proto)
	defer C.free(unsafe.Pointer(cProto))

	servent := C.getservbyport(C.int(port), cProto)
	if servent == nil {
		return strconv.Itoa(port)
	}

	name := C.GoString(servent.s_name)
	if name == "" {
		return strconv.Itoa(port)
	}

	return name
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
