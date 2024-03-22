package main

import (
	"context"
	"net"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/torrefatto/grpc-probe"
	"github.com/torrefatto/grpc-probe/svc"
)

var (
	defaultFrequency = 3 * time.Second
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var debug bool
	if os.Getenv("DEBUG") != "" {
		debug = true
	}

	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	if debug {
		logger = logger.Level(zerolog.DebugLevel)
	}

	name := os.Getenv("NAME")
	if name == "" {
		logger.Fatal().Msg("NAME is required")
	}

	freqStr := os.Getenv("FREQUENCY")
	if freqStr != "" {
		freq, err := time.ParseDuration(freqStr)
		if err != nil {
			logger.Fatal().Err(err).Msg("Invalid FREQUENCY")
		}
		defaultFrequency = freq
	}

	var peers []string
	if peerStr := os.Getenv("PEERS"); peerStr != "" {
		peers = strings.Split(peerStr, ",")
	}

	lis, err := net.Listen("tcp", ":12345")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to listen")
	}
	s := grpc.NewServer()
	peerCh := make(chan string)

	pinger := grpcprobe.NewPinger(name, peerCh, defaultFrequency, logger, peers)
	server := grpcprobe.NewProbeService(name, peerCh, logger)

	svc.RegisterProbeServer(s, server)

	go func() {
		err := s.Serve(lis)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to serve")
			cancel()
		}
	}()

	time.Sleep(1 * time.Second)

	pinger.Run(ctx)
}
