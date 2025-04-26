package nflog

/*
#cgo LDFLAGS: -lnetfilter_log

#include <netdb.h>
#include <string.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/socket.h>
#include <errno.h>

#include <libnfnetlink/libnfnetlink.h>
#include <libnetfilter_log/libnetfilter_log.h>

static int getErrno(void) {
	return errno;
}

static char* getStrerror(int err) {
	return strerror(err);
}

extern int goCallback(struct nflog_g_handle*, struct nfgenmsg*, struct nflog_data*, void*);

*/
import "C"
import (
	"errors"
	"fmt"
	"net"
	"unsafe"

	"github.com/rs/zerolog/log"
)

var nfLogMessageBufferSize = C.size_t(65536)

const PROTOCOL_TCP = 6
const PROTOCOL_UDP = 17
const PROTOCOL_ICMP = 1
const PROTOCOL_ICMPV6 = 58
const AF_INET = 2
const AF_INET6 = 10

type NfLogCallback func(packet NFLogPacket)

type NfLog struct {
	group       int
	handle      *C.struct_nflog_handle
	groupHandle *C.struct_nflog_g_handle
	fd          C.int
	callbackId  uint32
}

type NFLogPacket struct {
	Family     uint8
	Protocol   int32
	PayloadLen int
	Prefix     *string
	Indev      string
	Outdev     string
	Network    *NfLogPacketNetwork
}

type NfLogPacketNetwork struct {
	SrcIp     net.IP
	DestIp    net.IP
	Protocol  int
	Transport *NFLogTransportPacket
}

type NFLogTransportPacket struct {
	SrcPort  int
	DestPort int
}

func NewNfLog(group int) *NfLog {
	return &NfLog{
		group: group,
	}
}

func (n *NfLog) Start(callback NfLogCallback) error {
	n.callbackId = registerUserCallback(callback)

	defer n.close()

	log.Info().Msgf("Starting nfLog on group %d", n.group)
	n.handle = C.nflog_open()
	if n.handle == nil {
		return errors.Join(getErrnoError(), fmt.Errorf("failed to open nfLog"))
	}

	log.Info().Msgf("Binding group %d", n.group)
	n.groupHandle = C.nflog_bind_group(n.handle, C.uint16_t(n.group))
	if n.groupHandle == nil {
		return errors.Join(getErrnoError(), fmt.Errorf("failed to bind group %d", n.group))
	}

	log.Info().Msgf("Setting mode for group %d", n.group)
	if C.nflog_set_mode(n.groupHandle, C.NFULNL_COPY_PACKET, 0xffff) < 0 {
		return errors.Join(getErrnoError(), fmt.Errorf("failed to set mode for group %d", n.group))
	}

	C.nflog_callback_register(n.groupHandle, (*C.nflog_callback)(C.goCallback), unsafe.Pointer(n))

	fd := C.nflog_fd(n.handle)

	buf := C.malloc(nfLogMessageBufferSize)
	if buf == nil {
		return errors.Join(getErrnoError(), fmt.Errorf("failed to allocate buffer"))
	}
	defer C.free(buf)

	for {
		sz := C.recv(fd, buf, nfLogMessageBufferSize, 0)
		errno := C.getErrno()
		if sz < 0 && errno == C.EINTR {
			continue
		} else if sz < 0 {
			break
		}

		C.nflog_handle_packet(n.handle, (*C.char)(buf), C.int(sz))
	}

	return nil
}

//export goCallback
func goCallback(_ *C.struct_nflog_g_handle, nfmsg *C.struct_nfgenmsg, nfad *C.struct_nflog_data, data unsafe.Pointer) C.int {
	if nflog := (*NfLog)(data); nflog != nil {
		return C.int(nflog.callback(nfmsg, nfad))
	}

	return 0
}

func (n *NfLog) callback(nfmsg *C.struct_nfgenmsg, nfad *C.struct_nflog_data) int {
	if nfmsg == nil || nfad == nil {
		return 0
	}

	packet := interpretPacket(nfad, uint8(nfmsg.nfgen_family))
	callCallback(n.callbackId, packet)

	return 0
}

func interpretPacket(ldata *C.struct_nflog_data, pfFamily uint8) NFLogPacket {
	packet := NFLogPacket{
		Family: pfFamily,
	}

	// Header
	ph := C.nflog_get_msg_packet_hdr(ldata)
	if ph != nil {
		proto := uint16(C.ntohs(ph.hw_protocol))
		packet.Protocol = int32(proto)
	}

	// Payload
	var payload *C.char
	payloadLen := C.nflog_get_payload(ldata, &payload)
	if payloadLen >= 0 {
		packet.Network = interpretNetwork(C.GoBytes(unsafe.Pointer(payload), payloadLen), packet.Family)
		packet.PayloadLen = int(payloadLen)
	}

	// Prefix
	prefix := C.nflog_get_prefix(ldata)
	if prefix != nil {
		prefixStr := C.GoString(prefix)
		packet.Prefix = &prefixStr
	}

	// interfaces
	indev := uint32(C.nflog_get_indev(ldata))
	packet.Indev = getInterfaceName(indev)
	outdev := uint32(C.nflog_get_outdev(ldata))
	packet.Outdev = getInterfaceName(outdev)

	return packet
}

func interpretNetwork(payload []byte, family uint8) *NfLogPacketNetwork {
	switch family {
	case AF_INET: // AF_INET
		return interpretIPv4(payload)
	case AF_INET6: // AF_INET6
		return interpretIPv6(payload)
	default:
		return nil
	}
}

func interpretIPv4(payload []byte) *NfLogPacketNetwork {
	if len(payload) < 20 {
		return nil
	}

	network := &NfLogPacketNetwork{
		SrcIp:    net.IP(payload[12:16]),
		DestIp:   net.IP(payload[16:20]),
		Protocol: int(payload[9]),
	}

	switch network.Protocol {
	case PROTOCOL_TCP:
		network.Transport = interpretTCP(payload[20:])
	case PROTOCOL_UDP:
		network.Transport = interpretUDP(payload[20:])
	}

	return network
}

func interpretIPv6(payload []byte) *NfLogPacketNetwork {
	if len(payload) < 40 {
		return nil
	}
	network := &NfLogPacketNetwork{
		SrcIp:    net.IP(payload[8:24]),
		DestIp:   net.IP(payload[24:40]),
		Protocol: int(payload[6]),
	}

	switch network.Protocol {
	case PROTOCOL_TCP:
		network.Transport = interpretTCP(payload[40:])
	case PROTOCOL_UDP:
		network.Transport = interpretUDP(payload[40:])
	}

	return network
}
func interpretTCP(payload []byte) *NFLogTransportPacket {
	if len(payload) < 20 {
		return nil
	}

	tcpHeader := payload[:20]
	srcPort := (int(tcpHeader[0]) << 8) + int(tcpHeader[1])
	destPort := (int(tcpHeader[2]) << 8) + int(tcpHeader[3])

	return &NFLogTransportPacket{
		SrcPort:  srcPort,
		DestPort: destPort,
	}
}
func interpretUDP(payload []byte) *NFLogTransportPacket {
	if len(payload) < 8 {
		return nil
	}

	udpHeader := payload[:8]
	srcPort := (int(udpHeader[0]) << 8) + int(udpHeader[1])
	destPort := (int(udpHeader[2]) << 8) + int(udpHeader[3])

	return &NFLogTransportPacket{
		SrcPort:  srcPort,
		DestPort: destPort,
	}
}

var interfaceCache = make(map[uint32]string)

func getInterfaceName(index uint32) string {
	if index == 0 {
		return "unknown"
	}

	if name, ok := interfaceCache[index]; ok {
		return name
	}

	iface, err := net.InterfaceByIndex(int(index))
	if err != nil {
		return fmt.Sprintf("unknown-%d", index)
	}

	interfaceCache[index] = iface.Name
	return iface.Name
}

func getErrnoError() error {
	errno := C.getErrno()
	if errno != 0 {
		strconvErr := C.GoString(C.getStrerror(errno))
		return fmt.Errorf("errno %d: %s", errno, strconvErr)
	}
	return fmt.Errorf("unknown error")
}

func (n *NfLog) close() error {
	if n.handle != nil {
		C.nflog_close(n.handle)
		n.handle = nil
	}
	if n.groupHandle != nil {
		C.nflog_unbind_group(n.groupHandle)
		n.groupHandle = nil
	}
	return nil
}
