package simulcrypt

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestEMMGClientLifecycleAndProvision(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer ln.Close()

	done := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil { done <- err; return }
		defer conn.Close()

		msg, err := readEMMGTestMessage(conn)
		if err != nil { done <- err; return }
		if msg.Type != EMMGMsgChannelSetup { done <- fmt.Errorf("got 0x%04x, want channel_setup", msg.Type); return }
		if v, err := Uint32Value(emmgParam(msg, EMMGParamClientID)); err != nil || v != 0x4AFC0001 { done <- fmt.Errorf("bad client_id"); return }
		if v, err := Uint16Value(emmgParam(msg, EMMGParamDataChannelID)); err != nil || v != 1 { done <- fmt.Errorf("bad channel"); return }
		if v := emmgParam(msg, EMMGParamSectionTSPktFlag).Value; len(v) != 1 || v[0] != 0 { done <- fmt.Errorf("bad section flag"); return }

		if err := writeTestMessage(conn, NewMessage(EMMGMsgChannelStatus,
			Uint32Parameter(EMMGParamClientID, 0x4AFC0001),
			Uint16Parameter(EMMGParamDataChannelID, 1),
			Uint8Parameter(EMMGParamSectionTSPktFlag, 0),
		)); err != nil { done <- err; return }

		msg, err = readEMMGTestMessage(conn)
		if err != nil { done <- err; return }
		if msg.Type != EMMGMsgStreamSetup { done <- fmt.Errorf("got 0x%04x, want stream_setup", msg.Type); return }
		if v, err := Uint16Value(emmgParam(msg, EMMGParamDataStreamID)); err != nil || v != 1 { done <- fmt.Errorf("bad stream"); return }
		if v, err := Uint16Value(emmgParam(msg, EMMGParamDataID)); err != nil || v != 100 { done <- fmt.Errorf("bad data id"); return }
		if v := emmgParam(msg, EMMGParamDataType).Value; len(v) != 1 || v[0] != EMMGDataTypeEMM { done <- fmt.Errorf("bad data type"); return }

		if err := writeTestMessage(conn, NewMessage(EMMGMsgStreamStatus,
			Uint32Parameter(EMMGParamClientID, 0x4AFC0001),
			Uint16Parameter(EMMGParamDataChannelID, 1),
			Uint16Parameter(EMMGParamDataStreamID, 1),
			Uint16Parameter(EMMGParamDataID, 100),
			Uint8Parameter(EMMGParamDataType, EMMGDataTypeEMM),
		)); err != nil { done <- err; return }

		msg, err = readEMMGTestMessage(conn)
		if err != nil { done <- err; return }
		if msg.Type != EMMGMsgDataProvision { done <- fmt.Errorf("got 0x%04x, want data_provision", msg.Type); return }
		if len(msg.Parameters) != 6 { done <- fmt.Errorf("got %d parameters, want 6", len(msg.Parameters)); return }
		if got := emmgParam(msg, EMMGParamDatagram).Value; string(got) != string([]byte{0x80, 0x01, 0x02}) { done <- fmt.Errorf("bad first datagram: %x", got); return }
		var datagrams [][]byte
		for _, p := range msg.Parameters { if p.Type == EMMGParamDatagram { datagrams = append(datagrams, p.Value) } }
		if len(datagrams) != 2 || string(datagrams[1]) != string([]byte{0x80, 0x04}) { done <- fmt.Errorf("bad datagrams"); return }

		msg, err = readEMMGTestMessage(conn)
		if err != nil { done <- err; return }
		if msg.Type != EMMGMsgStreamCloseReq { done <- fmt.Errorf("got 0x%04x, want stream_close_request", msg.Type); return }
		if err := writeTestMessage(conn, NewMessage(EMMGMsgStreamCloseResp,
			Uint32Parameter(EMMGParamClientID, 0x4AFC0001),
			Uint16Parameter(EMMGParamDataChannelID, 1),
			Uint16Parameter(EMMGParamDataStreamID, 1),
		)); err != nil { done <- err; return }
		done <- nil
	}()

	client := NewEMMGClient(EMMGClientConfig{
		Address: "127.0.0.1:" + fmt.Sprint(ln.Addr().(*net.TCPAddr).Port),
		DialTimeout: time.Second, IOTimeout: time.Second,
		ClientID: 0x4AFC0001, DataChannelID: 1, DataStreamID: 1, DataID: 100,
		SectionTSPktFlag: 0, DataType: EMMGDataTypeEMM,
	})
	if err := client.Connect(); err != nil { t.Fatal(err) }
	defer client.Close()
	if err := client.OpenStream(); err != nil { t.Fatal(err) }
	if err := client.Provision([]byte{0x80, 0x01, 0x02}, []byte{0x80, 0x04}); err != nil { t.Fatal(err) }
	if err := client.CloseStream(); err != nil { t.Fatal(err) }
	if err := <-done; err != nil { t.Fatal(err) }
}

func emmgParam(m Message, typ uint16) Parameter {
	raw, ok := m.First(typ)
	if !ok { return Parameter{Type: typ} }
	return Parameter{Type: typ, Value: raw}
}

func readEMMGTestMessage(conn net.Conn) (Message, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil { return Message{}, err }
	n := int(hdr[3])<<8 | int(hdr[4])
	body := make([]byte, n)
	if _, err := io.ReadFull(conn, body); err != nil { return Message{}, err }
	return Decode(append(hdr[:], body...))
}
