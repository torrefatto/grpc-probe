package grpcprobe

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/torrefatto/grpc-probe/svc"
)

type Pinger struct {
	name        string
	peers       map[string]svc.ProbeClient
	peerReceive <-chan string
	frequency   time.Duration
	logger      zerolog.Logger
}

func NewPinger(name string, peerReceive <-chan string, frequency time.Duration, logger zerolog.Logger, initialPeers []string) *Pinger {
	peers := make(map[string]svc.ProbeClient)

	for _, peer := range initialPeers {
		client := addPeer(peer, logger)
		if client != nil {
			peers[peer] = client
		}
	}

	return &Pinger{
		name:        name,
		peers:       peers,
		peerReceive: peerReceive,
		frequency:   frequency,
		logger:      logger,
	}
}

func (p *Pinger) Run(ctx context.Context) {
	p.logger.Info().Msg("Pinger started")

	ticker := time.NewTicker(p.frequency)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info().Msg("Pinger stopped")
			return
		case peer := <-p.peerReceive:
			p.logger.Warn().Str("peer", peer).Msg("Received a peer")
			client := addPeer(peer, p.logger)
			if client != nil {
				p.peers[peer] = client
			}
		case <-ticker.C:
			for peer := range p.peers {
				go p.pingPong(ctx, peer)
			}
		}
	}
}

func (p *Pinger) pingPong(ctx context.Context, target string) {
	p.logger.Info().
		Str("target", target).
		Msg("Sending a ping")
	client, ok := p.peers[target]
	if !ok {
		p.logger.Error().Str("target", target).Msg("Failed to get a client")
		return
	}
	pong, err := client.PingIt(ctx, &svc.Ping{
		Sender: p.name,
		SentAt: timestamppb.Now(),
	})
	if err != nil {
		p.logger.Error().Err(err).Msg("Failed to send a ping")
		return
	}
	p.logger.Debug().
		Str("receiver", pong.Receiver).
		Time("received_at", pong.ReceivedAt.AsTime()).
		Msg("Received a pong")
}

func addPeer(peer string, logger zerolog.Logger) svc.ProbeClient {
	conn, err := grpc.Dial(peer+":12345", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to peer")
		return nil
	}
	return svc.NewProbeClient(conn)
}
