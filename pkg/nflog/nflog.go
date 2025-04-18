package nflog

/*
#cgo LDFLAGS: -lnetfilter_log
#include <netdb.h>
#include <stdlib.h>
#include <unistd.h>

#include <libnfnetlink/libnfnetlink.h>
#include <libnetfilter_log/libnetfilter_log.h>
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

type NfLog struct {
	group       int
	handle      *C.struct_nflog_handle
	groupHandle *C.struct_nflog_g_handle
	fd          C.int
}

type NFLogPacket struct {
	Family       uint8
	Hook         *uint8
	Protocol     *uint16
	MAC          []byte
	MACLen       *uint16
	MACType      *uint16
	MACSource    []byte
	MACSourceLen *uint16
	Payload      []byte
	PayloadLen   int
	Prefix       *string
	Timestamp    time.Time
	Mark         uint32
	Indev        *uint32
	Outdev       *uint32
	UID          *uint32
	GID          *uint32
	SeqLocal     *uint32
	SeqGlobal    *uint32
}

func NewNfLog(group int) *NfLog {
	return &NfLog{
		group: group,
	}
}

func (n *NfLog) Start() error {
	n.handle = C.nflog_open()
	if n.handle == nil {
		return fmt.Errorf("failed to open nflog handle")
	}

	n.groupHandle = C.nflog_bind_group(n.handle, C.uint16_t(n.group))
	if n.groupHandle == nil {
		defer n.Close()
		return fmt.Errorf("failed to bind group %d", n.group)
	}

	// nflog_set_mode(ui->nful_gh, NFULNL_COPY_PACKET, 0xffff);
	if C.nflog_set_mode(n.groupHandle, C.NFULNL_COPY_PACKET, 0xffff) < 0 {
		defer n.Close()
		return fmt.Errorf("failed to set mode for group %d", n.group)
	}

	// nflog_set_callback(ui->nful_gh, callback, ui);
	C.nflog_callback_register(n.groupHandle, nil, unsafe.Pointer(n))

	n.fd = C.nflog_fd(n.handle)

	return nil
}

// struct nflog_g_handle *gh, struct nfgenmsg *nfmsg, struct nflog_data *nfad, void *data
func callback(gh *C.struct_nflog_g_handle, nfmsg *C.struct_nfgenmsg, nfad *C.struct_nflog_data, data unsafe.Pointer) {
	if nflog := (*NfLog)(data); nflog != nil {
		nflog.callback(nfmsg, nfad)
	}
}

func (n *NfLog) callback(nfmsg *C.struct_nfgenmsg, nfad *C.struct_nflog_data) int {
	if nfmsg == nil || nfad == nil {
		return 0
	}

	packet := interpretPacket(nfad, uint8(nfmsg.nfgen_family))
	if packet == nil {
		return 0
	}

	fmt.Println(packet.Indev, packet.Outdev)

	return 0
}

func interpretPacket(ldata *C.struct_nflog_data, pfFamily uint8) *NFLogPacket {
	packet := &NFLogPacket{
		Family: pfFamily,
	}

	// Header
	ph := C.nflog_get_msg_packet_hdr(ldata)
	if ph != nil {
		hook := uint8(ph.hook)
		proto := uint16(C.ntohs(ph.hw_protocol))
		packet.Hook = &hook
		packet.Protocol = &proto
	}

	// Hardware header
	hwhdrLen := C.nflog_get_msg_packet_hwhdrlen(ldata)
	if hwhdrLen > 0 {
		hdrPtr := C.nflog_get_msg_packet_hwhdr(ldata)
		packet.MAC = C.GoBytes(unsafe.Pointer(hdrPtr), C.int(hwhdrLen))
		hlen := uint16(hwhdrLen)
		packet.MACLen = &hlen

		hwtype := uint16(C.nflog_get_hwtype(ldata))
		packet.MACType = &hwtype
	}

	// HW address
	hw := C.nflog_get_packet_hw(ldata)
	if hw != nil {
		addrLen := uint16(C.ntohs(hw.hw_addrlen))
		packet.MACSource = C.GoBytes(unsafe.Pointer(&hw.hw_addr[0]), C.int(addrLen))
		packet.MACSourceLen = &addrLen
	}

	// Payload
	var payload *C.char
	payloadLen := C.nflog_get_payload(ldata, &payload)
	if payloadLen >= 0 {
		packet.Payload = C.GoBytes(unsafe.Pointer(payload), payloadLen)
		packet.PayloadLen = int(payloadLen)
	}

	// Prefix
	prefix := C.nflog_get_prefix(ldata)
	if prefix != nil {
		prefixStr := C.GoString(prefix)
		packet.Prefix = &prefixStr
	}

	// Timestamp
	var ts C.struct_timeval
	if !(C.nflog_get_timestamp(ldata, &ts) == 0 && ts.tv_sec != 0) {
		now := time.Now()
		packet.Timestamp = now
	} else {
		packet.Timestamp = time.Unix(int64(ts.tv_sec), int64(ts.tv_usec)*1000)
	}

	// Mark, interfaces
	packet.Mark = uint32(C.nflog_get_nfmark(ldata))

	indev := uint32(C.nflog_get_indev(ldata))
	if indev > 0 {
		packet.Indev = &indev
	}
	outdev := uint32(C.nflog_get_outdev(ldata))
	if outdev > 0 {
		packet.Outdev = &outdev
	}

	// UID / GID
	var uid, gid C.uint32_t
	if C.nflog_get_uid(ldata, &uid) == 0 {
		u := uint32(uid)
		packet.UID = &u
	}
	if C.nflog_get_gid(ldata, &gid) == 0 {
		g := uint32(gid)
		packet.GID = &g
	}

	// Sequences
	var seq C.uint32_t
	if C.nflog_get_seq(ldata, &seq) == 0 {
		s := uint32(seq)
		packet.SeqLocal = &s
	}
	if C.nflog_get_seq_global(ldata, &seq) == 0 {
		sg := uint32(seq)
		packet.SeqGlobal = &sg
	}

	return packet
}

func (n *NfLog) Close() error {
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
