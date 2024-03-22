package grpcprobe

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/torrefatto/grpc-probe/svc"
)

type ProbeService struct {
	name     string
	peers    sync.Map
	peerSend chan<- string
	logger   zerolog.Logger
	svc.UnimplementedProbeServer
}

func NewProbeService(name string, peerSend chan<- string, logger zerolog.Logger) *ProbeService {
	return &ProbeService{
		name:     name,
		peerSend: peerSend,
		logger:   logger,
	}
}

func (s *ProbeService) PingIt(ctx context.Context, req *svc.Ping) (*svc.Pong, error) {
	s.logger.Info().
		Str("sender", req.Sender).
		Msg("Received a ping")

	now := time.Now()

	_, present := s.peers.LoadOrStore(req.Sender, struct{}{})
	if !present {
		s.logger.Warn().
			Str("peer", req.Sender).
			Msg("New peer detected")
		s.peerSend <- req.Sender
	}

	s.logger.Debug().
		Time("now", now).
		Msg("Sending a pong")

	return &svc.Pong{
		Receiver:   s.name,
		ReceivedAt: timestamppb.New(now),
	}, nil
}
