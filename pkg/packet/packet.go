package packet

import "time"

type Packet struct {
	Metadata *PacketMetadata `json:"metadata,omitempty"`
	Layers   []*Layer        `json:"layers,omitempty"`
}

func (p *Packet) ToMetric() Metric {
	labels := make(map[string]string)

	if p.Metadata != nil {
		if p.Metadata.InterfaceName != "" {
			labels["interface"] = p.Metadata.InterfaceName
		}
		if p.Metadata.Prefix != "" {
			labels["prefix"] = p.Metadata.Prefix
		}
		if p.Metadata.Hook != "" {
			labels["hook"] = p.Metadata.Hook
		}
	}

	// Extract protocol information from layers
	for _, layer := range p.Layers {
		if layer.Ipv4 != nil {
			labels["protocol"] = layer.Ipv4.Protocol
			labels["src_ip"] = layer.Ipv4.SrcIp
			labels["dest_ip"] = layer.Ipv4.DestIp
		}
		if layer.Ipv6 != nil {
			labels["protocol"] = layer.Ipv6.NextHeader
			labels["src_ip"] = layer.Ipv6.SrcIp
			labels["dest_ip"] = layer.Ipv6.DestIp
		}
		if layer.Tcp != nil {
			labels["transport"] = "tcp"
		}
		if layer.Udp != nil {
			labels["transport"] = "udp"
		}
		if layer.Icmpv4 != nil {
			labels["transport"] = "icmpv4"
		}
		if layer.Icmpv6 != nil {
			labels["transport"] = "icmpv6"
		}
	}

	var packetSize float64
	if p.Metadata != nil {
		packetSize = float64(p.Metadata.Length)
	}

	return Metric{
		Name:   "packet_bytes",
		Labels: labels,
		Value:  packetSize,
		Time:   p.Metadata.Timestamp,
	}
}

type PacketMetadata struct {
	Hostname      string    `json:"hostname,omitempty"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
	CaptureLength uint32    `json:"capture_length,omitempty"`
	Length        uint32    `json:"length,omitempty"`
	InterfaceName string    `json:"interface_name,omitempty"`
	Prefix        string    `json:"prefix,omitempty"`
	Hook          string    `json:"hook,omitempty"`
}

type Layer struct {
	Ethernet *LayerEthernet `json:"ethernet,omitempty"`
	Ipv4     *LayerIPv4     `json:"ipv4,omitempty"`
	Ipv6     *LayerIPv6     `json:"ipv6,omitempty"`
	Tcp      *LayerTCP      `json:"tcp,omitempty"`
	Udp      *LayerUDP      `json:"udp,omitempty"`
	Icmpv4   *LayerICMPV4   `json:"icmpv4,omitempty"`
	Icmpv6   *LayerICMPV6   `json:"icmpv6,omitempty"`
}

type LayerEthernet struct {
	SrcMac    string `json:"src_mac,omitempty"`
	DestMac   string `json:"dest_mac,omitempty"`
	Ethertype string `json:"ethertype,omitempty"`
}

type LayerIPv4 struct {
	SrcIp    string `json:"src_ip,omitempty"`
	DestIp   string `json:"dest_ip,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Ttl      uint32 `json:"ttl,omitempty"`
}

type LayerIPv6 struct {
	SrcIp      string `json:"src_ip,omitempty"`
	DestIp     string `json:"dest_ip,omitempty"`
	NextHeader string `json:"next_header,omitempty"`
	HopLimit   uint32 `json:"hop_limit,omitempty"`
}

type LayerTCP struct {
	SrcPort       uint32 `json:"src_port,omitempty"`
	DestPort      uint32 `json:"dest_port,omitempty"`
	Seq           uint32 `json:"seq,omitempty"`
	Ack           uint32 `json:"ack,omitempty"`
	DataOffset    uint32 `json:"data_offset,omitempty"`
	Window        uint32 `json:"window,omitempty"`
	Checksum      uint32 `json:"checksum,omitempty"`
	UrgentPointer uint32 `json:"urgent_pointer,omitempty"`
}

type LayerUDP struct {
	SrcPort  uint32 `json:"src_port,omitempty"`
	DestPort uint32 `json:"dest_port,omitempty"`
	Length   uint32 `json:"length,omitempty"`
	Checksum uint32 `json:"checksum,omitempty"`
}

type LayerICMPV4 struct {
	TypeCode string `json:"typeCode,omitempty"`
	Checksum uint32 `json:"checksum,omitempty"`
	Id       uint32 `json:"id,omitempty"`
	Seq      uint32 `json:"seq,omitempty"`
}

type LayerICMPV6 struct {
	TypeCode string `json:"typeCode,omitempty"`
	Checksum uint32 `json:"checksum,omitempty"`
}
