package main

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type UlogdMessage struct {
	IpProtocol int    `json:"ip.protocol"`
	SrcPort    int    `json:"src_port"`
	DestPort   int    `json:"dest_port"`
	OobIn      string `json:"oob.in"`
	OobOut     string `json:"oob.out"`
	SrcIp      string `json:"src_ip"`
	DestIp     string `json:"dest_ip"`
}

var (
	PacketTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ulogd_packets_total",
		Help: "Total number of blocked packets",
	})
	PacketByProtocol = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_protocol_total",
			Help: "Total number of packets grouped by IP protocol",
		},
		[]string{"protocol"},
	)
	PacketsByInterface = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_interface_total",
			Help: "Total packets per input network interface",
		},
		[]string{"interface"},
	)
	PacketsByDestPort = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_dest_port_total",
			Help: "Total packets per destination port",
		},
		[]string{"port"},
	)
	PacketsBySrcIP = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ulogd_packets_by_src_ip_total",
			Help: "Total packets grouped by source IP address",
		},
		[]string{"src_ip"},
	)
	PacketSizeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ulogd_packet_size_bytes",
			Help:    "Histogram of packet sizes",
			Buckets: prometheus.ExponentialBuckets(64, 2, 10), // 64B to ~32KB
		},
	)
	JsonParseErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ulogd_json_parse_errors_total",
			Help: "Number of times JSON parsing failed",
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
			return err
		}

		// Parse the JSON data
		var data UlogdMessage
		err = json.Unmarshal(buf[:len], &data)
		if err != nil {
			JsonParseErrors.Inc()
			continue
		}

		// Update the metrics
		PacketTotal.Inc()
		PacketByProtocol.WithLabelValues(strconv.Itoa(data.IpProtocol)).Inc()
		PacketsByInterface.WithLabelValues(data.OobIn).Inc()
		PacketsByDestPort.WithLabelValues(strconv.Itoa(data.DestPort)).Inc()
		PacketsBySrcIP.WithLabelValues(data.SrcIp).Inc()
		PacketSizeHistogram.Observe(float64(len))
	}
}
