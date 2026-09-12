package transport

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

const defaultReadDeadline = 250 * time.Millisecond

// RuntimeConfig describes one Phase 2 live transport path. Processing is a
// deliberate passthrough for now; later phases insert service routing, PSI/SI,
// SimulCrypt and scrambling between the queue and egress.
type RuntimeConfig struct {
	StreamID       string
	Protocol       string // "udp" or "rtp"
	BindAddr       string
	Port           int
	MulticastGroup string
	Interface      *net.Interface

	OutputLocalAddr string
	OutputAddr      string
	OutputInterface string

	QueueCapacity int
	BatchPackets  int
	Bitrate       uint64
	ReadBuffer    int
}

func (c RuntimeConfig) normalize() RuntimeConfig {
	if c.BindAddr == "" {
		c.BindAddr = "0.0.0.0"
	}
	if c.QueueCapacity == 0 {
		c.QueueCapacity = 4096
	}
	if c.BatchPackets == 0 {
		c.BatchPackets = DefaultBatchPackets
	}
	if c.ReadBuffer == 0 {
		c.ReadBuffer = DefaultDatagramSize
	}
	return c
}

func (c RuntimeConfig) validate() error {
	if c.StreamID == "" {
		return fmt.Errorf("stream ID is required")
	}
	if c.Protocol != "udp" && c.Protocol != "rtp" {
		return fmt.Errorf("unsupported protocol %q", c.Protocol)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid UDP port %d", c.Port)
	}
	if c.OutputAddr == "" {
		return fmt.Errorf("output address is required")
	}
	if c.QueueCapacity < 1 {
		return fmt.Errorf("queue capacity must be positive")
	}
	if c.BatchPackets < 1 {
		return fmt.Errorf("batch packets must be positive")
	}
	if c.Bitrate == 0 {
		return fmt.Errorf("bitrate must be positive")
	}
	if c.ReadBuffer < mpegts.PacketSize {
		return fmt.Errorf("read buffer must be at least 188 bytes")
	}
	return nil
}

// Runtime is the live Phase 2 ingest-to-egress worker.
type Runtime struct {
	Config RuntimeConfig
	Stream *Stream
	Source *UDPSource
	Sink   *UDPSink
}

func NewRuntime(config RuntimeConfig) (*Runtime, error) {
	config = config.normalize()
	if err := config.validate(); err != nil {
		return nil, err
	}

	stream, err := NewStream(config.StreamID, config.BindAddr, config.Protocol, config.QueueCapacity, config.Bitrate)
	if err != nil {
		return nil, err
	}
	source, err := ListenUDP(config.BindAddr, config.Port, config.MulticastGroup, config.Interface)
	if err != nil {
		return nil, err
	}

	var sink *UDPSink
	if config.OutputInterface != "" {
		sink, err = NewUDPSinkWithInterface(config.OutputLocalAddr, config.OutputAddr, config.OutputInterface)
	} else {
		sink, err = NewUDPSink(config.OutputLocalAddr, config.OutputAddr)
	}
	if err != nil {
		_ = source.Close()
		return nil, err
	}
	return &Runtime{Config: config, Stream: stream, Source: source, Sink: sink}, nil
}

func (r *Runtime) Close() error {
	var first error
	if r.Source != nil {
		if err := r.Source.Close(); err != nil && first == nil {
			first = err
		}
	}
	if r.Sink != nil {
		if err := r.Sink.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Run starts ingest and consumes queued TS packets in the current goroutine.
// The queue is the explicit back-pressure boundary: if processing/egress
// cannot keep up, ingest blocks rather than growing an unbounded backlog.
func (r *Runtime) Run(ctx context.Context) error {
	if r == nil || r.Stream == nil || r.Source == nil || r.Sink == nil {
		return fmt.Errorf("runtime is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	defer r.Close()

	r.Stream.ResetPacing(time.Now())
	ingestDone := make(chan error, 1)
	go func() {
		ingestDone <- r.ingestLoop(ctx)
	}()

	var cumulativeBytes int
	for {
		batch := make([]mpegts.Packet, 0, r.Config.BatchPackets)
		select {
		case <-ctx.Done():
			return nil
		case err := <-ingestDone:
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				return err
			}
			return nil
		case p := <-r.Stream.Queue.C:
			batch = append(batch, p)
		}

		fill:
		for len(batch) < r.Config.BatchPackets {
			select {
			case p := <-r.Stream.Queue.C:
				batch = append(batch, p)
			default:
				break fill
			}
		}

		for _, p := range batch {
			r.Stream.Monitor.Observe(p)
		}
		written, err := r.Sink.WritePackets(batch, r.Config.BatchPackets*mpegts.PacketSize)
		if err != nil {
			return err
		}
		cumulativeBytes += written * mpegts.PacketSize
		r.Stream.Pacer.Wait(cumulativeBytes)
	}
}

func (r *Runtime) ingestLoop(ctx context.Context) error {
	buf := make([]byte, r.Config.ReadBuffer)
	for {
		if err := r.Source.SetReadDeadline(time.Now().Add(defaultReadDeadline)); err != nil {
			return err
		}
		datagram, _, err := r.Source.ReadDatagram(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					return context.Canceled
				default:
					continue
				}
			}
			select {
			case <-ctx.Done():
				return context.Canceled
			default:
			}
			r.Stream.Stats.AddPacketErrors(1)
			continue
		}
		if err := r.Stream.EnqueueDatagramContext(ctx, datagram); err != nil {
			if ctx.Err() != nil {
				return context.Canceled
			}
			// EnqueueDatagramContext already increments packet errors. Keep the
			// source alive so one malformed datagram cannot terminate a service.
			continue
		}
	}
}
