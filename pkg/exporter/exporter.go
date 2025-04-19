package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/schidstorm/ulogd_udp_json_exporter/pkg/nflog"
)

var (
	PacketTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_total",
			Help: "Total number of blocked packets",
		},
		[]string{"prefix"},
	)
	PacketByProtocol = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_protocol_total",
			Help: "Total number of packets grouped by IP protocol",
		},
		[]string{"prefix", "protocol"},
	)
	PacketsByInterface = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_interface_total",
			Help: "Total packets per input network interface",
		},
		[]string{"prefix", "iif", "oif"},
	)
	PacketsByDestPort = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_dest_port_total",
			Help: "Total packets per destination port",
		},
		[]string{"prefix", "port"},
	)
	PacketsBySrcIP = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_src_ip_total",
			Help: "Total packets grouped by source IP address",
		},
		[]string{"prefix", "src_ip"},
	)
	PacketsByDestIP = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_dest_ip_total",
			Help: "Total packets grouped by destination IP address",
		},
		[]string{"prefix", "dest_ip"},
	)
	PacketSizeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ulogd_packet_size_bytes",
			Help:    "Histogram of packet sizes",
			Buckets: prometheus.ExponentialBuckets(64, 2, 10), // 64B to ~32KB
		},
		[]string{"prefix"},
	)
	JsonParseErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ulogd_json_parse_errors_total",
			Help: "Number of times JSON parsing failed",
		},
	)
	PacketReadErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ulogd_packet_read_errors_total",
			Help: "Number of times reading from UDP socket failed",
		},
	)
)

type UlogdMessage struct {
	IpProtocol int32  `json:"ip.protocol"`
	SrcPort    int    `json:"src_port"`
	DestPort   int32  `json:"dest_port"`
	OobIn      string `json:"oob.in"`
	OobOut     string `json:"oob.out"`
	SrcIp      string `json:"src_ip"`
	DestIp     string `json:"dest_ip"`
	Prefix     string `json:"oob.prefix"`
}

func RunExporter(group int) error {
	prometheus.MustRegister(PacketTotal)
	prometheus.MustRegister(PacketByProtocol)
	prometheus.MustRegister(PacketsByInterface)
	prometheus.MustRegister(PacketsByDestPort)
	prometheus.MustRegister(PacketsBySrcIP)
	prometheus.MustRegister(PacketsByDestIP)
	prometheus.MustRegister(PacketSizeHistogram)
	prometheus.MustRegister(JsonParseErrors)
	prometheus.MustRegister(PacketReadErrors)

	return nflog.NewNfLog(group).Start(nfLogCallback)
}

func nfLogCallback(packet nflog.NFLogPacket) {

	// Update the metrics
	prefix := ""
	if packet.Prefix != nil {
		prefix = *packet.Prefix
	}

	PacketTotal.WithLabelValues(prefix).Inc()
	PacketsByInterface.WithLabelValues(prefix, packet.Indev, packet.Outdev).Inc()
	PacketSizeHistogram.WithLabelValues(prefix).Observe(float64(packet.PayloadLen))

	if packet.Network != nil {
		PacketsBySrcIP.WithLabelValues(prefix, packet.Network.SrcIp.String()).Inc()
		PacketsByDestIP.WithLabelValues(prefix, packet.Network.DestIp.String()).Inc()

		if packet.Network.Transport != nil {
			protoName, serviceName := GetProtoAndService(int32(packet.Network.Transport.DestPort), int32(packet.Network.Protocol))

			PacketByProtocol.WithLabelValues(prefix, protoName).Inc()
			PacketsByDestPort.WithLabelValues(prefix, serviceName).Inc()
		}
	}
}
