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

func TestEMMGToTSIntegration(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer ln.Close()

	emmSection := []byte{0x82, 0xb0, 0x06, 0x00, 0x65, 0xc1, 0x00, 0x00, 0xaa, 0xbb, 0xcc}
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
			b, err := simulcrypt.Encode(m)
			if err != nil { return err }
			_, err = conn.Write(b)
			return err
		}

		m, err := read()
		if err != nil { done <- err; return }
		if m.Type != simulcrypt.EMMGMsgChannelSetup {
			done <- fmt.Errorf("want channel_setup, got 0x%04x", m.Type); return
		}
		if err := write(simulcrypt.NewMessage(simulcrypt.EMMGMsgChannelStatus,
			simulcrypt.Uint32Parameter(simulcrypt.EMMGParamClientID, 7),
			simulcrypt.Uint16Parameter(simulcrypt.EMMGParamDataChannelID, 2))); err != nil { done <- err; return }

		m, err = read()
		if err != nil { done <- err; return }
		if m.Type != simulcrypt.EMMGMsgStreamSetup {
			done <- fmt.Errorf("want stream_setup, got 0x%04x", m.Type); return
		}
		if err := write(simulcrypt.NewMessage(simulcrypt.EMMGMsgStreamStatus,
			simulcrypt.Uint32Parameter(simulcrypt.EMMGParamClientID, 7),
			simulcrypt.Uint16Parameter(simulcrypt.EMMGParamDataChannelID, 2),
			simulcrypt.Uint16Parameter(simulcrypt.EMMGParamDataStreamID, 3),
			simulcrypt.Uint16Parameter(simulcrypt.EMMGParamDataID, 101),
			simulcrypt.Uint8Parameter(simulcrypt.EMMGParamDataType, simulcrypt.EMMGDataTypeEMM),
		)); err != nil { done <- err; return }

		m, err = read()
		if err != nil { done <- err; return }
		if m.Type != simulcrypt.EMMGMsgDataProvision {
			done <- fmt.Errorf("want data_provision, got 0x%04x", m.Type); return
		}
		p, ok := m.First(simulcrypt.EMMGParamDatagram)
		if !ok || string(p) != string(emmSection) {
			done <- fmt.Errorf("EMM datagram mismatch"); return
		}
		done <- nil
	}()

	client := simulcrypt.NewEMMGClient(simulcrypt.EMMGClientConfig{
		Address: ln.Addr().String(), DialTimeout: time.Second, IOTimeout: time.Second,
		ClientID: 7, DataChannelID: 2, DataStreamID: 3, DataID: 101,
		SectionTSPktFlag: 0, DataType: simulcrypt.EMMGDataTypeEMM,
	})
	if err := client.Connect(); err != nil { t.Fatal(err) }
	defer client.Close()
	if err := client.OpenStream(); err != nil { t.Fatal(err) }
	if err := client.Provision(emmSection); err != nil { t.Fatal(err) }

	s := Service{ServiceID: 101, EMM: []CAEndpoint{{CASystemID: 0x4afc, PID: 0x1ffe}}}
	inserter, err := NewEMMInserter(s, 0x4afc, 6)
	if err != nil { t.Fatal(err) }
	pkts, err := inserter.InjectDatagram(emmSection)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 1 { t.Fatalf("got %d packets, want 1", len(pkts)) }
	p := pkts[0]
	if p.Data[0] != 0x47 || p.Data[1]&0x40 == 0 { t.Fatal("bad EMM TS header") }
	if got := uint16(p.Data[1]&0x1f)<<8 | uint16(p.Data[2]); got != 0x1ffe { t.Fatalf("PID 0x%04x", got) }
	if p.Data[3]&0x0f != 6 { t.Fatalf("CC=%d", p.Data[3]&0x0f) }
	if p.Data[4] != 0 { t.Fatalf("pointer=%d", p.Data[4]) }
	if got := p.Data[5:5+len(emmSection)]; string(got) != string(emmSection) { t.Fatalf("EMM mismatch: %x", got) }

	if err := <-done; err != nil { t.Fatal(err) }
}

func TestEMMInserterTSPacketFormat(t *testing.T) {
	s := Service{ServiceID: 101, EMM: []CAEndpoint{{CASystemID: 0x4afc, PID: 0x1ffe}}}
	inserter, err := NewEMMInserterWithFormat(s, 0x4afc, 0, EMMTSPacketFormat)
	if err != nil { t.Fatal(err) }

	input := make([]byte, mpegts.PacketSize)
	input[0] = mpegts.SyncByte
	input[1] = 0x40 | 0x02 // PUSI + input PID 0x0002; MUX must replace PID.
	input[2] = 0x02
	input[3] = 0x12
	for i := 4; i < len(input); i++ { input[i] = byte(i) }

	pkts, err := inserter.InjectDatagram(input)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 1 { t.Fatalf("got %d packets, want 1", len(pkts)) }
	p := pkts[0]
	if p.PID() != 0x1ffe { t.Fatalf("PID=0x%04x", p.PID()) }
	if p.ContinuityCounter() != 2 { t.Fatalf("CC=%d", p.ContinuityCounter()) }
	for i := 3; i < len(input); i++ {
		if p.Data[i] != input[i] { t.Fatalf("byte %d changed", i) }
	}
}

func TestEMMInserterTSPacketFormatRejectsInvalidDatagrams(t *testing.T) {
	s := Service{ServiceID: 101, EMM: []CAEndpoint{{CASystemID: 0x4afc, PID: 0x1ffe}}}
	inserter, err := NewEMMInserterWithFormat(s, 0x4afc, 0, EMMTSPacketFormat)
	if err != nil { t.Fatal(err) }

	if _, err := inserter.InjectDatagram(make([]byte, mpegts.PacketSize-1)); err == nil {
		t.Fatal("expected short TS datagram to be rejected")
	}
	bad := make([]byte, mpegts.PacketSize)
	bad[0] = 0x46
	if _, err := inserter.InjectDatagram(bad); err == nil {
		t.Fatal("expected invalid sync byte to be rejected")
	}
}
