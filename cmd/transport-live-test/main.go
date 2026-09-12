package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/transport"
)

func main() {
	var (
		protocol        = flag.String("protocol", "udp", "input protocol: udp or rtp")
		bindAddr        = flag.String("bind", "0.0.0.0", "input bind address")
		port            = flag.Int("port", 6000, "input UDP port")
		multicastGroup  = flag.String("multicast-group", "", "input multicast group; empty for unicast")
		inputInterface  = flag.String("input-interface", "", "input multicast interface name")
		outputLocalAddr = flag.String("output-local", "", "output local/source address")
		outputAddr      = flag.String("output", "239.100.1.1:5000", "output UDP destination")
		outputInterface = flag.String("output-interface", "", "output interface name")
		bitrate         = flag.Uint64("bitrate", 4000000, "target bitrate in bits/s")
		queueCapacity   = flag.Int("queue", 4096, "bounded queue capacity in TS packets")
		batchPackets    = flag.Int("batch", 7, "TS packets per UDP output datagram")
		readBuffer      = flag.Int("read-buffer", 2048, "UDP read buffer size")
		duration        = flag.Duration("duration", 30*time.Second, "test duration; 0 runs until interrupted")
	)
	flag.Parse()

	var ifaceName string
	if *inputInterface != "" {
		ifaceName = *inputInterface
	}
	var iface interfaceResolver
	if ifaceName != "" {
		netIface, err := lookupInterface(ifaceName)
		if err != nil {
			fatal(err)
		}
		iface = netIface
	}

	cfg := transport.RuntimeConfig{
		StreamID:         "phase2.14-live",
		Protocol:         *protocol,
		BindAddr:         *bindAddr,
		Port:             *port,
		MulticastGroup:   *multicastGroup,
		Interface:        iface,
		OutputLocalAddr:  *outputLocalAddr,
		OutputAddr:       *outputAddr,
		OutputInterface:  *outputInterface,
		QueueCapacity:    *queueCapacity,
		BatchPackets:     *batchPackets,
		Bitrate:          *bitrate,
		ReadBuffer:       *readBuffer,
	}

	r, err := transport.NewRuntime(cfg)
	if err != nil {
		fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *duration > 0 {
		var stop context.CancelFunc
		ctx, stop = context.WithTimeout(ctx, *duration)
		defer stop()
	}

	fmt.Printf("Phase 2.14 live transport test: %s input %s:%d -> %s, output interface=%s\n", *protocol, *bindAddr, *port, *outputAddr, *outputInterface)
	if err := r.Run(ctx); err != nil {
		fatal(err)
	}
	fmt.Println("Phase 2.14 live transport test stopped cleanly")
}

// Keep the CLI independent of net.Interface in its flag parsing while passing
// the concrete interface expected by the transport package.
type interfaceResolver = transport.Interface

func lookupInterface(name string) (interfaceResolver, error) {
	return transport.LookupInterface(name)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "transport-live-test:", err)
	os.Exit(1)
}
