package service

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/simulcrypt"
)

func TestECMGToTSIntegration(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer ln.Close()

	ecmSection := []byte{0x80, 0xb0, 0x06, 0x00, 0x65, 0xc1, 0x00, 0x00, 0x11, 0x22, 0x33}
	done := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil { done <- err; return }
		defer conn.Close()

		read := func() (simulcrypt.Message, error) {
			var h [5]byte
			if _, err := io.ReadFull(conn, h[:]); err != nil { return simulcrypt.Message{}, err }
			n := int(h[3])<<8 | int(h[4])
			b := make([]byte, n)
			if _, err := io.ReadFull(conn, b); err != nil { return simulcrypt.Message{}, err }
			return simulcrypt.Decode(append(h[:], b...))
		}
		write := func(m simulcrypt.Message) error {
			b, err := simulcrypt.Encode(m); if err != nil { return err }
			_, err = conn.Write(b); return err
		}

		m, err := read(); if err != nil { done <- err; return }
		if m.Type != simulcrypt.MsgChannelSetup { done <- fmt.Errorf("want channel_setup, got 0x%04x", m.Type); return }
		if err := write(simulcrypt.NewMessage(simulcrypt.MsgChannelStatus,
			simulcrypt.Uint16Parameter(simulcrypt.ParamECMChannelID, 1),
			simulcrypt.Uint32Parameter(simulcrypt.ParamSuperCASID, 0x4AFC0001))); err != nil { done <- err; return }

		m, err = read(); if err != nil { done <- err; return }
		if m.Type != simulcrypt.MsgStreamSetup { done <- fmt.Errorf("want stream_setup, got 0x%04x", m.Type); return }
		if err := write(simulcrypt.NewMessage(simulcrypt.MsgStreamStatus,
			simulcrypt.Uint16Parameter(simulcrypt.ParamECMChannelID, 1),
			simulcrypt.Uint16Parameter(simulcrypt.ParamECMStreamID, 1),
			simulcrypt.Uint16Parameter(simulcrypt.ParamECMID, 101))); err != nil { done <- err; return }

		m, err = read(); if err != nil { done <- err; return }
		if m.Type != simulcrypt.MsgCWProvision { done <- fmt.Errorf("want CW_provision, got 0x%04x", m.Type); return }
		if err := write(simulcrypt.NewMessage(simulcrypt.MsgECMResponse,
			simulcrypt.Uint16Parameter(simulcrypt.ParamECMChannelID, 1),
			simulcrypt.Uint16Parameter(simulcrypt.ParamECMStreamID, 1),
			simulcrypt.Uint16Parameter(simulcrypt.ParamCPNumber, 10),
			simulcrypt.BytesParameter(simulcrypt.ParamECMDatagram, ecmSection),
		)); err != nil { done <- err; return }
		done <- nil
	}()

	client := simulcrypt.NewECMGClient(simulcrypt.ECMGClientConfig{
		Address: ln.Addr().String(), DialTimeout: time.Second, IOTimeout: time.Second,
		ECMChannelID: 1, ECMStreamID: 1, ECMID: 101, SuperCASID: 0x4AFC0001,
	})
	if err := client.Connect(); err != nil { t.Fatal(err) }
	defer client.Close()
	if _, err := client.OpenStream(); err != nil { t.Fatal(err) }

	resp, err := client.ProvisionCW(10, []simulcrypt.CPControlWord{{CP: 10, CW: []byte{1,2,3,4,5,6,7,8}}}, nil, nil)
	if err != nil { t.Fatal(err) }

	s := Service{ServiceID: 101, ECM: []CAEndpoint{{CASystemID: 0x4AFC, PID: 0x1FFE}}}
	inserter, err := NewECMInserter(s, 0x4AFC, 3)
	if err != nil { t.Fatal(err) }
	pkts, err := inserter.InjectResponse(resp)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 1 { t.Fatalf("got %d packets, want 1", len(pkts)) }
	p := pkts[0]
	if p.Data[0] != 0x47 || p.Data[1]&0x40 == 0 { t.Fatal("bad ECM TS header") }
	if got := uint16(p.Data[1]&0x1f)<<8 | uint16(p.Data[2]); got != 0x1FFE { t.Fatalf("PID 0x%04x", got) }
	if p.Data[3]&0x0f != 3 { t.Fatalf("CC=%d", p.Data[3]&0x0f) }
	if p.Data[4] != 0 { t.Fatalf("pointer=%d", p.Data[4]) }
	if got := p.Data[5:5+len(ecmSection)]; string(got) != string(ecmSection) { t.Fatalf("ECM mismatch: %x", got) }

	if err := <-done; err != nil { t.Fatal(err) }
	_ = mpegts.PacketSize
}
