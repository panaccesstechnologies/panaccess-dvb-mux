package transport

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

func freeUDPPort(t *testing.T) int {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

func TestRuntimeUDPIngestToEgress(t *testing.T) {
	sourcePort := freeUDPPort(t)
	sinkPort := freeUDPPort(t)

	receiver, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: sinkPort})
	if err != nil {
		t.Fatal(err)
	}
	defer receiver.Close()

	r, err := NewRuntime(RuntimeConfig{
		StreamID:      "phase2-test",
		Protocol:      "udp",
		BindAddr:      "127.0.0.1",
		Port:          sourcePort,
		OutputAddr:    net.JoinHostPort("127.0.0.1", formatPort(sinkPort)),
		QueueCapacity: 16,
		BatchPackets:  7,
		Bitrate:       8_000_000,
		ReadBuffer:    2048,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	sender, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: sourcePort})
	if err != nil {
		cancel()
		<-done
		t.Fatal(err)
	}
	defer sender.Close()

	var packet mpegts.Packet
	packet.Data[0] = mpegts.SyncByte
	packet.Data[3] = 0x10
	if _, err := sender.Write(packet.Data[:]); err != nil {
		cancel()
		<-done
		t.Fatal(err)
	}

	if err := receiver.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 2048)
	n, _, err := receiver.ReadFromUDP(buf)
	if err != nil {
		cancel()
		<-done
		t.Fatal(err)
	}
	if n != mpegts.PacketSize {
		t.Fatalf("output datagram length=%d, want %d", n, mpegts.PacketSize)
	}
	if buf[0] != mpegts.SyncByte {
		t.Fatalf("output sync byte=0x%02x, want 0x47", buf[0])
	}

	stats := r.Stream.Stats.Snapshot()
	monitor := r.Stream.Monitor.Snapshot()
	if stats.PacketsReceived != 1 {
		t.Fatalf("PacketsReceived=%d, want 1", stats.PacketsReceived)
	}
	if monitor.CCErrors != 0 {
		t.Fatalf("CCErrors=%d, want 0", monitor.CCErrors)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("runtime did not stop after cancellation")
	}
}

func formatPort(port int) string {
	return net.JoinHostPort("127.0.0.1", "")[:0] + itoa(port)
}

func itoa(port int) string {
	if port == 0 {
		return "0"
	}
	var b [6]byte
	i := len(b)
	for port > 0 {
		i--
		b[i] = byte('0' + port%10)
		port /= 10
	}
	return string(b[i:])
}
