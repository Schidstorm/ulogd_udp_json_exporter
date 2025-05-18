package monitor

import "github.com/schidstorm/ulogd_monitor/pkg/nflog"

func packetToProto(packet nflog.NFLogPacket) *NFLogPacket {
	prefix := ""
	if packet.Prefix != nil {
		prefix = *packet.Prefix
	}

	var network *NFLogPacket_Network
	if packet.Network != nil {
		var transport *NFLogPacket_Network_Transport
		if packet.Network.Transport != nil {
			transport = &NFLogPacket_Network_Transport{
				SrcPort:  int32(packet.Network.Transport.SrcPort),
				DestPort: int32(packet.Network.Transport.DestPort),
			}
		}

		network = &NFLogPacket_Network{
			SrcIp:     packet.Network.SrcIp,
			DestIp:    packet.Network.DestIp,
			Protocol:  int32(packet.Network.Protocol),
			Transport: transport,
		}
	}

	return &NFLogPacket{
		Family:     uint32(packet.Family),
		Protocol:   packet.Protocol,
		PayloadLen: int32(packet.PayloadLen),
		Prefix:     prefix,
		Indev:      packet.Indev,
		Outdev:     packet.Outdev,
		Network:    network,
	}
}

func packetFromProto(packet *NFLogPacket) nflog.NFLogPacket {
	prefix := ""
	if packet.Prefix != "" {
		prefix = packet.Prefix
	}

	var network *nflog.NfLogPacketNetwork
	if packet.Network != nil {
		var transport *nflog.NFLogTransportPacket
		if packet.Network.Transport != nil {
			transport = &nflog.NFLogTransportPacket{
				SrcPort:  int(packet.Network.Transport.SrcPort),
				DestPort: int(packet.Network.Transport.DestPort),
			}
		}
		network = &nflog.NfLogPacketNetwork{
			SrcIp:     packet.Network.SrcIp,
			DestIp:    packet.Network.DestIp,
			Protocol:  int(packet.Network.Protocol),
			Transport: transport,
		}
	}

	return nflog.NFLogPacket{
		Family:     uint8(packet.Family),
		Protocol:   packet.Protocol,
		PayloadLen: int(packet.PayloadLen),
		Prefix:     &prefix,
		Indev:      packet.Indev,
		Outdev:     packet.Outdev,
		Network:    network,
	}
}
