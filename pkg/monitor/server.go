package monitor

import (
	"crypto/rand"
	"encoding/json"
	"io/fs"
	"math/big"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"

	"embed"

	"github.com/gorilla/websocket"
)

var mockPackets = []string{
	`{"metadata":{"hostname":"scan","capture_length":60,"length":60,"prefix":"reject"},"layers":[{"Layer":{"Ethernet":{"ethertype":"IPv4"}}},{"Layer":{"Ipv4":{"src_ip":"192.168.1.65","dest_ip":"34.96.126.106","protocol":"TCP","ttl":64}}},{"Layer":{"Tcp":{"src_port":33132,"dest_port":443,"seq":65944144,"data_offset":10,"window":64240,"checksum":25314}}}]}`,
	`{"metadata":{"hostname":"scan","capture_length":66,"length":66,"prefix":"accept"},"layers":[{"Layer":{"Ethernet":{"ethertype":"IPv4"}}},{"Layer":{"Ipv4":{"src_ip":"192.168.1.65","dest_ip":"192.168.1.155","protocol":"UDP","ttl":64}}},{"Layer":{"Udp":{"src_port":37092,"dest_port":53,"length":46,"checksum":33900}}}]}`,
	`{"metadata":{"hostname":"scan","capture_length":655,"length":655,"prefix":"reject"},"layers":[{"Layer":{"Ethernet":{"ethertype":"IPv6"}}},{"Layer":{"Ipv6":{"src_ip":"fe80::24f1:caac:a217:c23d","dest_ip":"ff02::c","next_header":"UDP","hop_limit":1}}},{"Layer":{"Udp":{"src_port":51072,"dest_port":3702,"length":615,"checksum":21499}}}]}`,
	`{"metadata":{"hostname":"scan","capture_length":64,"length":64,"prefix":"reject"},"layers":[{"Layer":{"Ethernet":{"ethertype":"IPv6"}}},{"Layer":{"Ipv6":{"src_ip":"fe80::c093:bcff:fe40:96dc","dest_ip":"ff02::1","next_header":"UDP","hop_limit":1}}},{"Layer":{"Udp":{"src_port":8612,"dest_port":8612,"length":24,"checksum":4191}}}]}`,
	`{"metadata":{"hostname":"scan","capture_length":44,"length":44,"prefix":"accept"},"layers":[{"Layer":{"Ethernet":{"ethertype":"IPv4"}}},{"Layer":{"Ipv4":{"src_ip":"10.88.0.1","dest_ip":"10.88.255.255","protocol":"UDP","ttl":64}}},{"Layer":{"Udp":{"src_port":8612,"dest_port":8610,"length":24,"checksum":5338}}}]}`,
	`{"metadata":{"hostname":"scan","capture_length":76,"length":76,"prefix":"reject"},"layers":[{"Layer":{"Ethernet":{"ethertype":"IPv6"}}},{"Layer":{"Ipv6":{"src_ip":"fe80::68df:f9ff:fe83:b152","dest_ip":"ff02::16","next_header":"IPv6HopByHop","hop_limit":1}}},{"Layer":{"Icmpv6":{"typeCode":"143(0)","checksum":24137}}}]}`,
	`{"metadata":{"hostname":"scan","capture_length":60,"length":60,"prefix":"accept"},"layers":[{"Layer":{"Ethernet":{"ethertype":"IPv4"}}},{"Layer":{"Ipv4":{"src_ip":"192.168.1.65","dest_ip":"192.168.1.155","protocol":"TCP","ttl":64}}},{"Layer":{"Tcp":{"src_port":41128,"dest_port":5000,"seq":3429858000,"data_offset":10,"window":64240,"checksum":33883}}}]}`,
	`{"metadata":{"hostname":"wlan","capture_length":2852,"length":2852,"prefix":"reject"},"layers":[{"Layer":{"Ethernet":{"ethertype":"IPv4"}}},{"Layer":{"Ipv4":{"src_ip":"192.168.1.9","dest_ip":"34.96.126.106","protocol":"TCP","ttl":64}}},{"Layer":{"Tcp":{"src_port":54590,"dest_port":443,"seq":2481316014,"ack":2194214556,"data_offset":8,"window":590,"checksum":28050}}}]}`,
}

var upgrader = websocket.Upgrader{}

//go:embed public/html/index.html
var indexHTML []byte

//go:embed public/js
//go:embed public/js/*
var jsFiles embed.FS

//go:embed public/css
//go:embed public/css/*
var cssFiles embed.FS

type Server struct {
	PacketQueue      PacketQueue
	readMany         *QueueReadMany
	mux              *http.ServeMux
	isDev            bool
	websocketCounter atomic.Int64
	agentReceiver    *AgentReceiver
}

type ServerConfig struct {
	HttpListenAddr string
	GrpcListenAddr string

	IsDev bool
}

func NewServer() *Server {
	s := &Server{
		PacketQueue: CreatePacketQueue(),
		readMany:    &QueueReadMany{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/packets", s.packets)
	mux.HandleFunc("/", handleHtml)
	jsFs, _ := fs.Sub(jsFiles, "public/js")
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.FS(jsFs))))
	cssFs, _ := fs.Sub(cssFiles, "public/css")
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.FS(cssFs))))
	s.mux = mux

	return s
}

func (s *Server) Start(cfg ServerConfig) error {
	go func() {
		log.Info().Msg("Starting packet reader")
		if err := s.readMany.Start(s.PacketQueue); err != nil {
			panic(err)
		}
	}()

	if cfg.IsDev {
		s.isDev = true
	} else {
		s.agentReceiver = NewAgentReceiver(s.PacketQueue)
		go func() {
			err := s.agentReceiver.Serve(cfg.GrpcListenAddr)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to start agent receiver")
			}
		}()
	}

	srv := &http.Server{
		Addr:    cfg.HttpListenAddr,
		Handler: s.mux,
	}
	log.Info().Str("address", cfg.HttpListenAddr).Msg("Starting http server")
	if err := srv.ListenAndServe(); err != nil {
		log.Err(err).Msg("listen and serve")
	}

	return nil
}

func (s *Server) packets(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Err(err).Msg("upgrade")
		return
	}
	defer c.Close()
	s.incrementWebsocketCounter()
	readQueue, id := s.readMany.Attach()
	defer s.readMany.Detach(id)
	defer s.decrementWebsocketCounter()

	for {
		var message []byte

		if s.isDev {
			randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(mockPackets))))
			if err != nil {
				log.Err(err).Msg("rand int")
				break
			}
			message = []byte(mockPackets[randomIndex.Int64()])
			time.Sleep(500 * time.Millisecond)
		} else {
			packet := readQueue.Dequeue()
			message, err = json.Marshal(&packet)
			if err != nil {
				log.Err(err).Msg("json marshal")
				break
			}
		}

		err = c.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			log.Info().Err(err).Msg("failed to write to websocket")
			break
		}
	}
}

func (s *Server) incrementWebsocketCounter() {
	if s.websocketCounter.Add(1) == 1 {
		if s.agentReceiver != nil {
			s.agentReceiver.SetActive(true)
		}
		log.Info().Msg("First websocket connected, activating agent")
	}
}

func (s *Server) decrementWebsocketCounter() {
	if s.websocketCounter.Add(-1) == 0 {
		if s.agentReceiver != nil {
			s.agentReceiver.SetActive(false)
		}
		log.Info().Msg("Last websocket disconnected, deactivating agent")
	}
}

func handleHtml(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(indexHTML)
	if err != nil {
		log.Err(err).Msg("write index.html")
	}
}
