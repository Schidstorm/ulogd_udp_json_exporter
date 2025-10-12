package monitor

import (
	"io"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/schidstorm/ulogd_monitor/pkg/pb"

	"google.golang.org/grpc"
)

func NewAgentReceiver(packetQueue PacketQueue) *AgentReceiver {
	receiver := &AgentReceiver{
		PacketQueue: packetQueue,
		grpcServer:  grpc.NewServer(),
		streams:     make(map[int]grpc.BidiStreamingServer[pb.StreamPacketsRequest, pb.StreamPacketsResponse]),
	}
	pb.RegisterMonitorServer(receiver.grpcServer, receiver)

	return receiver
}

type AgentReceiver struct {
	pb.UnimplementedMonitorServer

	PacketQueue PacketQueue
	streams     map[int]grpc.BidiStreamingServer[pb.StreamPacketsRequest, pb.StreamPacketsResponse]
	streamsLock sync.Mutex
	active      bool
	grpcServer  *grpc.Server
}

func (s *AgentReceiver) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Info().Msgf("Starting gRPC server on %s", addr)
	return s.grpcServer.Serve(lis)
}

func (s *AgentReceiver) StreamPackets(stream grpc.BidiStreamingServer[pb.StreamPacketsRequest, pb.StreamPacketsResponse]) error {
	deregister := s.addStream(stream)
	defer deregister.Close()

	if s.active {
		stream.Send(activeCommand(true))
	}

	for {
		if packetRequest, err := stream.Recv(); err != nil {
			if err == io.EOF {
				return nil
			}
			log.Error().Err(err).Msg("Error receiving packet from stream")
			return err
		} else {
			if packetRequest.Packets == nil {
				continue
			}

			for _, packet := range packetRequest.Packets {
				if packet == nil {
					continue
				}
				s.PacketQueue.Enqueue(packet)
			}
		}
	}
}

type streamHandle func()

func (sh streamHandle) Close() error {
	sh()
	return nil
}

func (s *AgentReceiver) addStream(stream grpc.BidiStreamingServer[pb.StreamPacketsRequest, pb.StreamPacketsResponse]) io.Closer {
	s.streamsLock.Lock()
	defer s.streamsLock.Unlock()

	var maxHandle int = 0
	for handle := range s.streams {
		if handle >= maxHandle {
			maxHandle = handle
		}
	}
	s.streams[int(maxHandle)+1] = stream

	return streamHandle(func() {
		s.streamsLock.Lock()
		defer s.streamsLock.Unlock()
		delete(s.streams, int(maxHandle)+1)
	})
}

func (s *AgentReceiver) SetActive(active bool) {
	s.streamsLock.Lock()
	defer s.streamsLock.Unlock()

	s.active = active
	for _, stream := range s.streams {
		if err := stream.Send(activeCommand(active)); err != nil {
			log.Error().Err(err).Msg("Error sending active command to stream")
		}
	}
}

func activeCommand(active bool) *pb.StreamPacketsResponse {
	return &pb.StreamPacketsResponse{
		Response: &pb.StreamPacketsResponse_ActivateCommand{
			ActivateCommand: &pb.ActivateCommand{
				Activate: active,
			},
		},
	}
}
