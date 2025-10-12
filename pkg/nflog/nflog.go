package nflog

/*
#cgo LDFLAGS: -lnetfilter_log

#include <netdb.h>
#include <string.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/socket.h>
#include <errno.h>
#include <linux/netfilter.h>

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

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/rs/zerolog/log"
	"github.com/schidstorm/ulogd_monitor/pkg/pb"
)

var nfLogMessageBufferSize = C.size_t(65536)

const PROTOCOL_TCP = 6
const PROTOCOL_UDP = 17
const PROTOCOL_ICMP = 1
const PROTOCOL_ICMPV6 = 58
const AF_INET = 2
const AF_INET6 = 10

type NfLog struct {
	group       int
	handle      *C.struct_nflog_handle
	groupHandle *C.struct_nflog_g_handle
	fd          C.int
	callbackId  uint32
}

func NewNfLog(group int) *NfLog {
	return &NfLog{
		group: group,
	}
}

func (n *NfLog) Start(callback CallbackFunc) error {
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

func interpretPacket(ldata *C.struct_nflog_data, pfFamily uint8) *pb.Packet {
	packet := pb.Packet{
		Metadata: &pb.PacketMetadata{},
		Layers:   []*pb.Layer{},
	}

	// Header
	ph := C.nflog_get_msg_packet_hdr(ldata)
	if ph != nil {
		proto := uint16(C.ntohs(ph.hw_protocol))
		gopacketProto := layers.EthernetType(proto)
		packet.Layers = append(packet.Layers, &pb.Layer{
			Layer: &pb.Layer_Ethernet{
				Ethernet: &pb.LayerEthernet{
					Ethertype: gopacketProto.String(),
				},
			},
		})

		var hook string
		switch ph.hook {
		case C.NF_INET_PRE_ROUTING:
			hook = "prerouting"
		case C.NF_INET_LOCAL_IN:
			hook = "input"
		case C.NF_INET_FORWARD:
			hook = "forward"
		case C.NF_INET_LOCAL_OUT:
			hook = "output"
		case C.NF_INET_POST_ROUTING:
			hook = "postrouting"
		default:
			hook = ""
		}
		packet.Metadata.Hook = hook
	}

	// Payload
	var payload *C.char
	payloadLen := C.nflog_get_payload(ldata, &payload)
	if payloadLen >= 0 {
		packet.Metadata.CaptureLength = uint32(payloadLen)
		packet.Metadata.Length = uint32(payloadLen)
		packetLayers, err := interpretNetwork(C.GoBytes(unsafe.Pointer(payload), payloadLen), pfFamily)
		if err != nil {
			log.Info().Err(err).Msg("failed to interpret network layer")
		} else {
			packet.Layers = append(packet.Layers, packetLayers...)
		}
	}

	// Prefix
	prefix := C.nflog_get_prefix(ldata)
	if prefix != nil {
		prefixStr := C.GoString(prefix)
		packet.Metadata.Prefix = prefixStr
	}

	// // interfaces
	// indev := uint32(C.nflog_get_indev(ldata))
	// packet.Metadata.InterfaceName = getInterfaceName(indev)
	// outdev := uint32(C.nflog_get_outdev(ldata))
	// packet.Outdev = getInterfaceName(outdev)

	return &packet
}

func interpretNetwork(payload []byte, family uint8) ([]*pb.Layer, error) {
	switch family {
	case AF_INET: // AF_INET
		return interpretIPv4(payload)
	case AF_INET6: // AF_INET6
		return interpretIPv6(payload)
	default:
		return nil, fmt.Errorf("unknown protocol family: %d", family)
	}
}

func interpretIPv4(payload []byte) ([]*pb.Layer, error) {
	packet := gopacket.NewPacket(payload, layers.LayerTypeIPv4, gopacket.Default)
	if err := packet.ErrorLayer(); err != nil {
		return nil, err.Error()
	}

	return gopacketTolayers(packet), nil
}

func interpretIPv6(payload []byte) ([]*pb.Layer, error) {
	packet := gopacket.NewPacket(payload, layers.LayerTypeIPv6, gopacket.Default)
	packet.Metadata()
	if err := packet.ErrorLayer(); err != nil {
		return nil, err.Error()
	}

	return gopacketTolayers(packet), nil
}

func gopacketTolayers(packet gopacket.Packet) []*pb.Layer {
	var layersList []*pb.Layer
	for _, layer := range packet.Layers() {
		switch l := layer.(type) {
		case *layers.Ethernet:
			layersList = append(layersList, &pb.Layer{
				Layer: &pb.Layer_Ethernet{
					Ethernet: &pb.LayerEthernet{
						SrcMac:    l.SrcMAC.String(),
						DestMac:   l.DstMAC.String(),
						Ethertype: l.EthernetType.String(),
					},
				},
			})
		case *layers.IPv4:
			layersList = append(layersList, &pb.Layer{
				Layer: &pb.Layer_Ipv4{
					Ipv4: &pb.LayerIPv4{
						SrcIp:    l.SrcIP.String(),
						DestIp:   l.DstIP.String(),
						Protocol: l.Protocol.String(),
						Ttl:      uint32(l.TTL),
					},
				},
			})
		case *layers.IPv6:
			layersList = append(layersList, &pb.Layer{
				Layer: &pb.Layer_Ipv6{
					Ipv6: &pb.LayerIPv6{
						SrcIp:      l.SrcIP.String(),
						DestIp:     l.DstIP.String(),
						NextHeader: l.NextHeader.String(),
						HopLimit:   uint32(l.HopLimit),
					},
				},
			})
		case *layers.TCP:
			layersList = append(layersList, &pb.Layer{
				Layer: &pb.Layer_Tcp{
					Tcp: &pb.LayerTCP{
						SrcPort:       uint32(l.SrcPort),
						DestPort:      uint32(l.DstPort),
						Seq:           l.Seq,
						Ack:           l.Ack,
						DataOffset:    uint32(l.DataOffset),
						Window:        uint32(l.Window),
						Checksum:      uint32(l.Checksum),
						UrgentPointer: uint32(l.Urgent),
					},
				},
			})
		case *layers.UDP:
			layersList = append(layersList, &pb.Layer{
				Layer: &pb.Layer_Udp{
					Udp: &pb.LayerUDP{
						SrcPort:  uint32(l.SrcPort),
						DestPort: uint32(l.DstPort),
						Length:   uint32(l.Length),
						Checksum: uint32(l.Checksum),
					},
				},
			})
		case *layers.ICMPv4:
			layersList = append(layersList, &pb.Layer{
				Layer: &pb.Layer_Icmpv4{
					Icmpv4: &pb.LayerICMPV4{
						TypeCode: l.TypeCode.String(),
						Id:       uint32(l.Id),
						Seq:      uint32(l.Seq),
					},
				},
			})
		case *layers.ICMPv6:
			layersList = append(layersList, &pb.Layer{
				Layer: &pb.Layer_Icmpv6{
					Icmpv6: &pb.LayerICMPV6{
						TypeCode: l.TypeCode.String(),
						Checksum: uint32(l.Checksum),
					},
				},
			})
		}
	}
	return layersList
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
