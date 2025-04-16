package main

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type UlogdMessage struct {
	IpProtocol int32  `json:"ip.protocol"`
	SrcPort    int    `json:"src_port"`
	DestPort   int32  `json:"dest_port"`
	OobIn      string `json:"oob.in"`
	OobOut     string `json:"oob.out"`
	SrcIp      string `json:"src_ip"`
	DestIp     string `json:"dest_ip"`
	Message    string `json:"message"`
}

var (
	PacketTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_total",
			Help: "Total number of blocked packets",
		},
		[]string{"message"},
	)
	PacketByProtocol = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_protocol_total",
			Help: "Total number of packets grouped by IP protocol",
		},
		[]string{"message", "protocol"},
	)
	PacketsByInterface = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_interface_total",
			Help: "Total packets per input network interface",
		},
		[]string{"message", "interface"},
	)
	PacketsByDestPort = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_dest_port_total",
			Help: "Total packets per destination port",
		},
		[]string{"message", "port"},
	)
	PacketsBySrcIP = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_src_ip_total",
			Help: "Total packets grouped by source IP address",
		},
		[]string{"message", "src_ip"},
	)
	PacketSizeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ulogd_packet_size_bytes",
			Help:    "Histogram of packet sizes",
			Buckets: prometheus.ExponentialBuckets(64, 2, 10), // 64B to ~32KB
		},
		[]string{"message"},
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

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var listenAddr string
	var metricsAddr string

	root := cobra.Command{
		Use: "ulogd_udp_json_exporter",
	}

	root.Flags().StringVarP(&listenAddr, "listen", "l", ":9999", "UDP address to listen on")
	root.Flags().StringVarP(&metricsAddr, "metrics", "m", ":8080", "HTTP address to expose metrics on")

	err := root.Execute()
	if err != nil {
		log.Fatal().Err(err).Msg("Error executing command")
	}

	prometheus.MustRegister(PacketTotal)
	prometheus.MustRegister(PacketByProtocol)
	prometheus.MustRegister(PacketsByInterface)
	prometheus.MustRegister(PacketsByDestPort)
	prometheus.MustRegister(PacketsBySrcIP)
	prometheus.MustRegister(PacketSizeHistogram)
	prometheus.MustRegister(JsonParseErrors)
	prometheus.MustRegister(PacketReadErrors)

	// Start the Prometheus metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info().Str("address", metricsAddr).Msg("Starting metrics server")
		if err := http.ListenAndServe(metricsAddr, nil); err != nil {
			log.Fatal().Err(err).Msg("Error starting metrics server")
		}
	}()

	if err := listen(listenAddr); err != nil {
		log.Fatal().Err(err).Msg("Error listening on UDP address")
	}
}

func listen(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("Error listening on UDP address")
	}

	log.Info().Str("address", addr).Msg("Listening on UDP address")

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
		PacketTotal.WithLabelValues(data.Message).Inc()
		PacketByProtocol.WithLabelValues(data.Message, protoName).Inc()
		PacketsByInterface.WithLabelValues(data.Message, data.OobIn).Inc()
		PacketsByDestPort.WithLabelValues(data.Message, serviceName).Inc()
		PacketsBySrcIP.WithLabelValues(data.Message, data.SrcIp).Inc()
		PacketSizeHistogram.WithLabelValues(data.Message).Observe(float64(len))
	}
}
