package simulcrypt

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestECMGClientChannelAndStreamLifecycle(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		msg, err := readTestMessage(conn)
		if err != nil {
			serverDone <- err
			return
		}
		if msg.Type != MsgChannelSetup {
			serverDone <- fmt.Errorf("got message 0x%04x, want channel_setup", msg.Type)
			return
		}
		ch, err := requireUint16(msg, ParamECMChannelID)
		if err != nil || ch != 1 {
			serverDone <- fmt.Errorf("bad channel id")
			return
		}
		if v, err := Uint32Value(paramFromMessage(msg, ParamSuperCASID)); err != nil || v != 0x4AFC0001 {
			serverDone <- fmt.Errorf("bad Super_CAS_id")
			return
		}
		if err := writeTestMessage(conn, NewMessage(MsgChannelStatus,
			Uint16Parameter(ParamECMChannelID, 1),
			Uint8Parameter(ParamSectionTSPktFlag, 0),
			Uint32Parameter(ParamDelayStart, 100),
			Uint32Parameter(ParamDelayStop, 100),
			Uint32Parameter(ParamECMRepPeriod, 1000),
			Uint16Parameter(ParamMaxStreams, 5),
			Uint32Parameter(ParamMinCPDuration, 2000),
			Uint8Parameter(ParamLeadCW, 0),
			Uint8Parameter(ParamCWPerMsg, 1),
			Uint32Parameter(ParamMaxCompTime, 100),
		)); err != nil {
			serverDone <- err
			return
		}

		msg, err = readTestMessage(conn)
		if err != nil {
			serverDone <- err
			return
		}
		if msg.Type != MsgStreamSetup {
			serverDone <- fmt.Errorf("got message 0x%04x, want stream_setup", msg.Type)
			return
		}
		if v, err := Uint16Value(paramFromMessage(msg, ParamECMChannelID)); err != nil || v != 1 {
			serverDone <- fmt.Errorf("bad stream channel")
			return
		}
		if v, err := Uint16Value(paramFromMessage(msg, ParamECMStreamID)); err != nil || v != 1 {
			serverDone <- fmt.Errorf("bad stream id")
			return
		}
		if v, err := Uint16Value(paramFromMessage(msg, ParamECMID)); err != nil || v != 101 {
			serverDone <- fmt.Errorf("bad ECM id")
			return
		}
		if v, err := Uint32Value(paramFromMessage(msg, ParamNominalCPDuration)); err != nil || v != 2000 {
			serverDone <- fmt.Errorf("bad CP duration")
			return
		}
		if err := writeTestMessage(conn, NewMessage(MsgStreamStatus,
			Uint16Parameter(ParamECMChannelID, 1),
			Uint16Parameter(ParamECMStreamID, 1),
			Uint16Parameter(ParamECMID, 101),
			Uint8Parameter(ParamAccessCriteriaTransferMode, 0),
		)); err != nil {
			serverDone <- err
			return
		}

		msg, err = readTestMessage(conn)
		if err != nil {
			serverDone <- err
			return
		}
		if msg.Type != MsgChannelTest {
			serverDone <- fmt.Errorf("got message 0x%04x, want channel_test", msg.Type)
			return
		}
		if err := writeTestMessage(conn, NewMessage(MsgChannelStatus,
			Uint16Parameter(ParamECMChannelID, 1),
			Uint8Parameter(ParamSectionTSPktFlag, 0),
		)); err != nil {
			serverDone <- err
			return
		}

		msg, err = readTestMessage(conn)
		if err != nil {
			serverDone <- err
			return
		}
		if msg.Type != MsgStreamTest {
			serverDone <- fmt.Errorf("got message 0x%04x, want stream_test", msg.Type)
			return
		}
		if err := writeTestMessage(conn, NewMessage(MsgStreamStatus,
			Uint16Parameter(ParamECMChannelID, 1),
			Uint16Parameter(ParamECMStreamID, 1),
			Uint16Parameter(ParamECMID, 101),
			Uint8Parameter(ParamAccessCriteriaTransferMode, 0),
		)); err != nil {
			serverDone <- err
			return
		}

		msg, err = readTestMessage(conn)
		if err != nil {
			serverDone <- err
			return
		}
		if msg.Type != MsgCWProvision {
			serverDone <- fmt.Errorf("got message 0x%04x, want CW_provision", msg.Type)
			return
		}
		if v, err := Uint16Value(paramFromMessage(msg, ParamCPNumber)); err != nil || v != 10 {
			serverDone <- fmt.Errorf("bad CP number")
			return
		}
		combo := paramFromMessage(msg, ParamCPCWCombination)
		if len(combo.Value) != 10 || combo.Value[0] != 0 || combo.Value[1] != 10 {
			serverDone <- fmt.Errorf("bad CP/CW combination")
			return
		}
		if v, err := Uint16Value(paramFromMessage(msg, ParamCPDuration)); err != nil || v != 2000 {
			serverDone <- fmt.Errorf("bad CP duration in CW provision")
			return
		}
		if got := string(paramFromMessage(msg, ParamAccessCriteria).Value); got != "test-ac" {
			serverDone <- fmt.Errorf("bad access criteria: %q", got)
			return
		}
		if err := writeTestMessage(conn, NewMessage(MsgECMResponse,
			Uint16Parameter(ParamECMChannelID, 1),
			Uint16Parameter(ParamECMStreamID, 1),
			Uint16Parameter(ParamCPNumber, 10),
			BytesParameter(ParamECMDatagram, []byte{0x80, 0x01, 0x02, 0x03, 0x04}),
		)); err != nil {
			serverDone <- err
			return
		}

		msg, err = readTestMessage(conn)
		if err != nil {
			serverDone <- err
			return
		}
		if msg.Type != MsgStreamCloseReq {
			serverDone <- fmt.Errorf("got message 0x%04x, want stream_close_request", msg.Type)
			return
		}
		if err := writeTestMessage(conn, NewMessage(MsgStreamCloseResp,
			Uint16Parameter(ParamECMChannelID, 1),
			Uint16Parameter(ParamECMStreamID, 1),
		)); err != nil {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	client := NewECMGClient(ECMGClientConfig{
		Address: "127.0.0.1:" + fmt.Sprint(ln.Addr().(*net.TCPAddr).Port),
		DialTimeout: time.Second,
		IOTimeout: time.Second,
		ECMChannelID: 1,
		SuperCASID: 0x4AFC0001,
		ECMStreamID: 1,
		ECMID: 101,
		NominalCPDuration: 2000,
	})
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	status, err := client.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	if status.ChannelID != 1 || status.StreamID != 1 || status.ECMID != 101 {
		t.Fatalf("unexpected stream status: %+v", status)
	}
	if err := client.TestChannel(); err != nil {
		t.Fatal(err)
	}
	if err := client.TestStream(); err != nil {
		t.Fatal(err)
	}
	duration := uint16(2000)
	ecm, err := client.ProvisionCW(10, []CPControlWord{{
		CP: 10,
		CW: []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}}, &duration, []byte("test-ac"))
	if err != nil {
		t.Fatal(err)
	}
	if ecm.ChannelID != 1 || ecm.StreamID != 1 || ecm.CPNumber != 10 {
		t.Fatalf("unexpected ECM response identity: %+v", ecm)
	}
	if got, want := ecm.ECMDatagram, []byte{0x80, 0x01, 0x02, 0x03, 0x04}; string(got) != string(want) {
		t.Fatalf("unexpected ECM datagram: %x", got)
	}
	if err := client.CloseStream(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

func paramFromMessage(m Message, typ uint16) Parameter {
	raw, ok := m.First(typ)
	if !ok {
		return Parameter{Type: typ}
	}
	return Parameter{Type: typ, Value: raw}
}

func readTestMessage(conn net.Conn) (Message, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return Message{}, err
	}
	body := make([]byte, int(hdr[3])<<8|int(hdr[4]))
	if len(body) > 0 {
		if _, err := io.ReadFull(conn, body); err != nil {
			return Message{}, err
		}
	}
	data := append(hdr[:], body...)
	return Decode(data)
}

func writeTestMessage(conn net.Conn, msg Message) error {
	data, err := Encode(msg)
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}
