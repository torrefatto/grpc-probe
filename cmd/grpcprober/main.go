package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/torrefatto/grpc-probe"
	"github.com/torrefatto/grpc-probe/svc"
)

var (
	defaultFrequency = 3 * time.Second
	defaultSvcPort   = 12345
	metricsPort      = 9090

	successes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_probe_success",
		Help: "The total number of failures",
	}, []string{"src", "dst"})
	failures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_probe_failure",
		Help: "The total number of failures",
	}, []string{"src", "dst", "type"})
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
	logger.Debug().Msg("Starting")

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

	var svcPort int
	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			logger.Fatal().Err(err).Msg("Invalid PORT")
		}
		svcPort = port
	} else {
		svcPort = defaultSvcPort
	}

	go setupPrometheus(metricsPort)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", svcPort))
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to listen")
	}
	s := grpc.NewServer()
	peerCh := make(chan string)

	pinger := grpcprobe.NewPinger(name, peerCh, defaultFrequency, logger, peers, svcPort, successes, failures)
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

func setupPrometheus(port int) {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
