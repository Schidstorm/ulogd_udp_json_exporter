package exporter

import (
	"encoding/json"
	"net"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
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

func RunExporter(listenAddr string) error {
	prometheus.MustRegister(PacketTotal)
	prometheus.MustRegister(PacketByProtocol)
	prometheus.MustRegister(PacketsByInterface)
	prometheus.MustRegister(PacketsByDestPort)
	prometheus.MustRegister(PacketsBySrcIP)
	prometheus.MustRegister(PacketsByDestIP)
	prometheus.MustRegister(PacketSizeHistogram)
	prometheus.MustRegister(JsonParseErrors)
	prometheus.MustRegister(PacketReadErrors)

	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("Error listening on UDP address")
	}

	log.Info().Str("address", listenAddr).Msg("Listening on UDP address")

	// Read from UDP listener in endless loop
	var buf [65536]byte
	for {
		len, _, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			PacketReadErrors.Inc()
			log.Debug().Err(err).Msg("Error reading from UDP socket")
			continue
		}

		// Parse the JSON data
		var data UlogdMessage
		err = json.Unmarshal(buf[:len], &data)
		if err != nil {
			JsonParseErrors.Inc()
			continue
		}

		protoName, serviceName := GetProtoAndService(data.DestPort, data.IpProtocol)

		// Update the metrics
		PacketTotal.WithLabelValues(data.Prefix).Inc()
		PacketByProtocol.WithLabelValues(data.Prefix, protoName).Inc()
		PacketsByInterface.WithLabelValues(data.Prefix, data.OobIn, data.OobOut).Inc()
		PacketsByDestPort.WithLabelValues(data.Prefix, serviceName).Inc()
		PacketsBySrcIP.WithLabelValues(data.Prefix, data.SrcIp).Inc()
		PacketsByDestIP.WithLabelValues(data.Prefix, data.DestIp).Inc()
		PacketSizeHistogram.WithLabelValues(data.Prefix).Observe(float64(len))
	}
}
