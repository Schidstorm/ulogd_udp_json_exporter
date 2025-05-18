package monitor

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/rs/zerolog/log"

	"embed"

	"github.com/gorilla/websocket"
	"github.com/schidstorm/ulogd_monitor/pkg/nflog"
)

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
	PacketQueue PacketQueue
	readMany    *QueueReadMany
	mux         *http.ServeMux
}

type Packet struct {
	NflogPacket nflog.NFLogPacket
	Hostname    string
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

	go func() {
		if cfg.IsDev {
			runNflogMock(s.PacketQueue)
		} else {
			err := RunAgentReceiver(cfg.GrpcListenAddr, s.PacketQueue)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to start agent receiver")
			}
		}
	}()

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
	readQueue, id := s.readMany.Attach()
	defer s.readMany.Detach(id)

	for {
		packet := readQueue.Dequeue()
		message, err := json.Marshal(&packet)
		if err != nil {
			log.Err(err).Msg("json marshal")
			break
		}

		err = c.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			log.Info().Err(err).Msg("failed to write to websocket")
			break
		}
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
