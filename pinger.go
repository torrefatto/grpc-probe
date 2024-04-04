package grpcprobe

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/torrefatto/grpc-probe/svc"
)

type Pinger struct {
	name        string
	peers       map[string]svc.ProbeClient
	port        int
	peerReceive <-chan string
	frequency   time.Duration
	logger      zerolog.Logger
	successes   *prometheus.CounterVec
	failures    *prometheus.CounterVec
}

func NewPinger(
	name string,
	peerReceive <-chan string,
	frequency time.Duration,
	logger zerolog.Logger,
	initialPeers []string,
	port int,
	successes *prometheus.CounterVec,
	failures *prometheus.CounterVec,
) *Pinger {
	pinger := &Pinger{
		name:        name,
		peers:       make(map[string]svc.ProbeClient),
		port:        port,
		peerReceive: peerReceive,
		frequency:   frequency,
		logger:      logger,
		successes:   successes,
		failures:    failures,
	}

	for _, peer := range initialPeers {
		_, err := pinger.addPeer(peer)
		if err != nil {
			pinger.logger.Error().Err(err).Str("peer", peer).Msg("Failed to add peer")
			pinger.peers[peer] = nil
		}
	}

	return pinger
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
			p.addPeer(peer)
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
	if !ok || client == nil {
		var err error
		client, err = p.addPeer(target)
		if err != nil {
			p.logger.Error().Err(err).Msg("Cannot ping")
			return
		}
	}
	pong, err := client.PingIt(ctx, &svc.Ping{
		Sender: p.name,
		SentAt: timestamppb.Now(),
	})
	if err != nil {
		p.logger.Error().Err(err).Msg("Failed to send a ping")
		p.failures.WithLabelValues(p.name, target, "ping").Inc()
		p.peers[target] = nil
		return
	}
	p.logger.Debug().
		Str("receiver", pong.Receiver).
		Time("received_at", pong.ReceivedAt.AsTime()).
		Msg("Received a pong")
	p.successes.WithLabelValues(p.name, target).Inc()
}

func (p *Pinger) addPeer(peer string) (svc.ProbeClient, error) {
	p.logger.Debug().Str("peer", peer).Msg("Adding a peer")
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", peer, p.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		p.logger.Error().Err(err).Msg("Failed to connect to peer")
		p.failures.WithLabelValues(p.name, peer, "conn").Inc()
		return nil, fmt.Errorf("Failed to connect")
	}
	client := svc.NewProbeClient(conn)
	if client == nil {
		p.logger.Error().Err(err).Msg("Failed to instantiate gRPC client")
		p.failures.WithLabelValues(p.name, peer, "client").Inc()
		return nil, fmt.Errorf("Failed to instantiate client")
	}
	p.logger.Info().Str("peer", peer).Msg("Added a peer")
	p.peers[peer] = client
	return client, nil
}
