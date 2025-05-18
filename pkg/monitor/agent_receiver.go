package monitor

import (
	context "context"
	"net"

	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"
)

func RunAgentReceiver(addr string, packetQueue PacketQueue) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	RegisterMonitorServer(s, &monitorServer{
		PacketQueue: packetQueue,
	})
	log.Info().Msgf("Starting gRPC server on %s", addr)

	return s.Serve(lis)
}

type monitorServer struct {
	UnimplementedMonitorServer

	PacketQueue PacketQueue
}

func (s *monitorServer) SendPacket(ctx context.Context, stream *SendPacketRequest) (*SendPacketResponse, error) {
	s.PacketQueue.Enqueue(Packet{
		NflogPacket: packetFromProto(stream.Packet),
		Hostname:    stream.Metadata.Hostname,
	})

	return &SendPacketResponse{}, nil
}
