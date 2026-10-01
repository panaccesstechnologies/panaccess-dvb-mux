package simulcrypt

import (
	"fmt"
	"io"
	"net"
	"time"
)

type EMMGClientConfig struct {
	Address          string
	DialTimeout      time.Duration
	IOTimeout        time.Duration
	ClientID         uint32
	DataChannelID    uint16
	DataStreamID     uint16
	DataID           uint16
	SectionTSPktFlag byte
	DataType         byte
}

type EMMGClient struct {
	conn net.Conn
	cfg  EMMGClientConfig
}

func NewEMMGClient(cfg EMMGClientConfig) *EMMGClient {
	if cfg.DialTimeout <= 0 { cfg.DialTimeout = 5 * time.Second }
	if cfg.IOTimeout <= 0 { cfg.IOTimeout = 5 * time.Second }
	if cfg.DataType == 0 { cfg.DataType = EMMGDataTypeEMM }
	return &EMMGClient{cfg: cfg}
}

func (c *EMMGClient) Connect() error {
	if c.conn != nil { return fmt.Errorf("EMMG client already connected") }
	conn, err := net.DialTimeout("tcp", c.cfg.Address, c.cfg.DialTimeout)
	if err != nil { return fmt.Errorf("EMMG dial %s: %w", c.cfg.Address, err) }
	c.conn = conn
	if err := c.setupChannel(); err != nil { _ = c.Close(); return err }
	return nil
}

func (c *EMMGClient) setupChannel() error {
	msg := NewMessage(EMMGMsgChannelSetup,
		Uint32Parameter(EMMGParamClientID, c.cfg.ClientID),
		Uint16Parameter(EMMGParamDataChannelID, c.cfg.DataChannelID),
		Uint8Parameter(EMMGParamSectionTSPktFlag, c.cfg.SectionTSPktFlag),
	)
	resp, err := c.exchange(msg, EMMGMsgChannelStatus, EMMGMsgChannelError)
	if err != nil { return fmt.Errorf("EMMG channel setup: %w", err) }
	if v, err := Uint32Value(paramFromMessage(resp, EMMGParamClientID)); err != nil || v != c.cfg.ClientID {
		return fmt.Errorf("EMMG channel status client_id mismatch")
	}
	return nil
}

func (c *EMMGClient) OpenStream() error {
	if c.conn == nil { return fmt.Errorf("EMMG client not connected") }
	msg := NewMessage(EMMGMsgStreamSetup,
		Uint32Parameter(EMMGParamClientID, c.cfg.ClientID),
		Uint16Parameter(EMMGParamDataChannelID, c.cfg.DataChannelID),
		Uint16Parameter(EMMGParamDataStreamID, c.cfg.DataStreamID),
		Uint16Parameter(EMMGParamDataID, c.cfg.DataID),
		Uint8Parameter(EMMGParamDataType, c.cfg.DataType),
	)
	resp, err := c.exchange(msg, EMMGMsgStreamStatus, EMMGMsgStreamError)
	if err != nil { return fmt.Errorf("EMMG stream setup: %w", err) }
	for _, typ := range []uint16{EMMGParamClientID, EMMGParamDataChannelID, EMMGParamDataStreamID, EMMGParamDataID, EMMGParamDataType} {
		if _, ok := resp.First(typ); !ok { return fmt.Errorf("EMMG stream status missing parameter 0x%04x", typ) }
	}
	return nil
}

func (c *EMMGClient) Provision(datagrams ...[]byte) error {
	if c.conn == nil { return fmt.Errorf("EMMG client not connected") }
	if len(datagrams) == 0 { return fmt.Errorf("at least one EMM datagram is required") }
	msg := EMMGDataProvision{
		ClientID: c.cfg.ClientID, DataChannelID: c.cfg.DataChannelID,
		DataStreamID: c.cfg.DataStreamID, DataID: c.cfg.DataID, Datagrams: datagrams,
	}.Message()
	if _, err := c.exchange(msg, EMMGMsgStreamStatus, EMMGMsgStreamError); err == nil {
		return nil
	}
	// Data_provision has no protocol-level response. Write it directly.
	return c.writeMessage(msg)
}

func (c *EMMGClient) TestStream() error {
	if c.conn == nil { return fmt.Errorf("EMMG client not connected") }
	msg := NewMessage(EMMGMsgStreamTest,
		Uint32Parameter(EMMGParamClientID, c.cfg.ClientID),
		Uint16Parameter(EMMGParamDataChannelID, c.cfg.DataChannelID),
		Uint16Parameter(EMMGParamDataStreamID, c.cfg.DataStreamID),
	)
	return c.writeAndExpect(msg, EMMGMsgStreamStatus, EMMGMsgStreamError)
}

func (c *EMMGClient) CloseStream() error {
	if c.conn == nil { return nil }
	msg := NewMessage(EMMGMsgStreamCloseReq,
		Uint32Parameter(EMMGParamClientID, c.cfg.ClientID),
		Uint16Parameter(EMMGParamDataChannelID, c.cfg.DataChannelID),
		Uint16Parameter(EMMGParamDataStreamID, c.cfg.DataStreamID),
	)
	if err := c.writeAndExpect(msg, EMMGMsgStreamCloseResp, EMMGMsgStreamError); err != nil {
		return fmt.Errorf("EMMG stream close: %w", err)
	}
	return nil
}

func (c *EMMGClient) Close() error {
	if c.conn == nil { return nil }
	err := c.conn.Close(); c.conn = nil
	return err
}

func (c *EMMGClient) exchange(msg Message, okType, errType uint16) (Message, error) {
	if err := c.writeMessage(msg); err != nil { return Message{}, err }
	return c.readExpected(okType, errType)
}

func (c *EMMGClient) writeAndExpect(msg Message, okType, errType uint16) error {
	_, err := c.exchange(msg, okType, errType)
	return err
}

func (c *EMMGClient) readExpected(okType, errType uint16) (Message, error) {
	resp, err := c.readMessage()
	if err != nil { return Message{}, err }
	if resp.ProtocolVersion != ECMGProtocolVersion { return Message{}, fmt.Errorf("unsupported EMMG protocol version 0x%02x", resp.ProtocolVersion) }
	if resp.Type == errType { return Message{}, fmt.Errorf("EMMG returned error message 0x%04x", resp.Type) }
	if resp.Type != okType { return Message{}, fmt.Errorf("%w: got 0x%04x want 0x%04x", ErrUnexpectedMessage, resp.Type, okType) }
	return resp, nil
}

func (c *EMMGClient) writeMessage(msg Message) error {
	data, err := Encode(msg); if err != nil { return err }
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.cfg.IOTimeout)); err != nil { return err }
	if _, err := c.conn.Write(data); err != nil { return fmt.Errorf("EMMG write: %w", err) }
	return nil
}

func (c *EMMGClient) readMessage() (Message, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.cfg.IOTimeout)); err != nil { return Message{}, err }
	var hdr [5]byte
	if _, err := io.ReadFull(c.conn, hdr[:]); err != nil { return Message{}, fmt.Errorf("EMMG read header: %w", err) }
	bodyLen := int(hdr[3])<<8 | int(hdr[4])
	data := make([]byte, 5+bodyLen)
	copy(data, hdr[:])
	if bodyLen > 0 { if _, err := io.ReadFull(c.conn, data[5:]); err != nil { return Message{}, fmt.Errorf("EMMG read body: %w", err) } }
	return Decode(data)
}
