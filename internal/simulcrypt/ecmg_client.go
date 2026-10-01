package simulcrypt

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

var ErrUnexpectedMessage = errors.New("unexpected ECMG message")

type ECMGClientConfig struct {
	Address          string
	DialTimeout      time.Duration
	IOTimeout        time.Duration
	ECMChannelID     uint16
	SuperCASID       uint32
	ECMStreamID      uint16
	ECMID            uint16
	NominalCPDuration uint32
}

type ECMGChannelStatus struct {
	Message Message
	ChannelID uint16
}

type ECMGStreamStatus struct {
	Message Message
	ChannelID uint16
	StreamID uint16
	ECMID uint16
}

type ECMGClient struct {
	conn net.Conn
	cfg  ECMGClientConfig
}

func NewECMGClient(cfg ECMGClientConfig) *ECMGClient {
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.IOTimeout <= 0 {
		cfg.IOTimeout = 5 * time.Second
	}
	if cfg.ECMChannelID == 0 {
		cfg.ECMChannelID = 1
	}
	if cfg.ECMStreamID == 0 {
		cfg.ECMStreamID = 1
	}
	if cfg.ECMID == 0 {
		cfg.ECMID = cfg.ECMStreamID
	}
	if cfg.NominalCPDuration == 0 {
		cfg.NominalCPDuration = 1000
	}
	return &ECMGClient{cfg: cfg}
}

func (c *ECMGClient) Connect() error {
	if c.conn != nil {
		return fmt.Errorf("ECMG client already connected")
	}
	conn, err := net.DialTimeout("tcp", c.cfg.Address, c.cfg.DialTimeout)
	if err != nil {
		return fmt.Errorf("ECMG dial %s: %w", c.cfg.Address, err)
	}
	c.conn = conn
	if err := c.setupChannel(); err != nil {
		_ = c.Close()
		return err
	}
	return nil
}

func (c *ECMGClient) setupChannel() error {
	msg := NewMessage(MsgChannelSetup,
		Uint16Parameter(ParamECMChannelID, c.cfg.ECMChannelID),
		Uint32Parameter(ParamSuperCASID, c.cfg.SuperCASID),
	)
	if err := c.exchange(msg, MsgChannelStatus, MsgChannelError); err != nil {
		return fmt.Errorf("ECMG channel setup: %w", err)
	}
	return nil
}

func (c *ECMGClient) OpenStream() (ECMGStreamStatus, error) {
	if c.conn == nil {
		return ECMGStreamStatus{}, fmt.Errorf("ECMG client not connected")
	}
	msg := NewMessage(MsgStreamSetup,
		Uint16Parameter(ParamECMChannelID, c.cfg.ECMChannelID),
		Uint16Parameter(ParamECMStreamID, c.cfg.ECMStreamID),
		Uint16Parameter(ParamECMID, c.cfg.ECMID),
		Uint32Parameter(ParamNominalCPDuration, c.cfg.NominalCPDuration),
	)
	resp, err := c.exchangeMessage(msg, MsgStreamStatus, MsgStreamError)
	if err != nil {
		return ECMGStreamStatus{}, fmt.Errorf("ECMG stream setup: %w", err)
	}
	channelID, err := requireUint16(resp, ParamECMChannelID)
	if err != nil {
		return ECMGStreamStatus{}, err
	}
	streamID, err := requireUint16(resp, ParamECMStreamID)
	if err != nil {
		return ECMGStreamStatus{}, err
	}
	ecmID, err := requireUint16(resp, ParamECMID)
	if err != nil {
		return ECMGStreamStatus{}, err
	}
	return ECMGStreamStatus{Message: resp, ChannelID: channelID, StreamID: streamID, ECMID: ecmID}, nil
}

func (c *ECMGClient) CloseStream() error {
	if c.conn == nil {
		return nil
	}
	msg := NewMessage(MsgStreamCloseReq,
		Uint16Parameter(ParamECMChannelID, c.cfg.ECMChannelID),
		Uint16Parameter(ParamECMStreamID, c.cfg.ECMStreamID),
	)
	resp, err := c.exchangeMessage(msg, MsgStreamCloseResp, MsgStreamError)
	if err != nil {
		return fmt.Errorf("ECMG stream close: %w", err)
	}
	if id, ok := resp.First(ParamECMChannelID); ok {
		got, err := Uint16Value(Parameter{Type: ParamECMChannelID, Value: id})
		if err != nil {
			return err
		}
		if got != c.cfg.ECMChannelID {
			return fmt.Errorf("ECMG close channel mismatch: got %d want %d", got, c.cfg.ECMChannelID)
		}
	}
	return nil
}

func (c *ECMGClient) TestChannel() error {
	if c.conn == nil {
		return fmt.Errorf("ECMG client not connected")
	}
	msg := NewMessage(MsgChannelTest, Uint16Parameter(ParamECMChannelID, c.cfg.ECMChannelID))
	return c.exchange(msg, MsgChannelStatus, MsgChannelError)
}

func (c *ECMGClient) TestStream() error {
	if c.conn == nil {
		return fmt.Errorf("ECMG client not connected")
	}
	msg := NewMessage(MsgStreamTest,
		Uint16Parameter(ParamECMChannelID, c.cfg.ECMChannelID),
		Uint16Parameter(ParamECMStreamID, c.cfg.ECMStreamID),
	)
	return c.exchange(msg, MsgStreamStatus, MsgStreamError)
}

func (c *ECMGClient) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *ECMGClient) exchange(msg Message, okType, errType uint16) error {
	_, err := c.exchangeMessage(msg, okType, errType)
	return err
}

func (c *ECMGClient) exchangeMessage(msg Message, okType, errType uint16) (Message, error) {
	if c.conn == nil {
		return Message{}, fmt.Errorf("ECMG client not connected")
	}
	if err := c.writeMessage(msg); err != nil {
		return Message{}, err
	}
	resp, err := c.readMessage()
	if err != nil {
		return Message{}, err
	}
	if resp.ProtocolVersion != ECMGProtocolVersion {
		return Message{}, fmt.Errorf("unsupported ECMG protocol version 0x%02x", resp.ProtocolVersion)
	}
	if resp.Type == errType {
		return Message{}, fmt.Errorf("ECMG returned error message 0x%04x", resp.Type)
	}
	if resp.Type != okType {
		return Message{}, fmt.Errorf("%w: got 0x%04x want 0x%04x", ErrUnexpectedMessage, resp.Type, okType)
	}
	return resp, nil
}

func (c *ECMGClient) writeMessage(msg Message) error {
	data, err := Encode(msg)
	if err != nil {
		return err
	}
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.cfg.IOTimeout)); err != nil {
		return err
	}
	_, err = c.conn.Write(data)
	if err != nil {
		return fmt.Errorf("ECMG write: %w", err)
	}
	return nil
}

func (c *ECMGClient) readMessage() (Message, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.cfg.IOTimeout)); err != nil {
		return Message{}, err
	}
	var hdr [5]byte
	if _, err := io.ReadFull(c.conn, hdr[:]); err != nil {
		return Message{}, fmt.Errorf("ECMG read header: %w", err)
	}
	bodyLen := int(hdr[3])<<8 | int(hdr[4])
	data := make([]byte, 5+bodyLen)
	copy(data, hdr[:])
	if bodyLen > 0 {
		if _, err := io.ReadFull(c.conn, data[5:]); err != nil {
			return Message{}, fmt.Errorf("ECMG read body: %w", err)
		}
	}
	return Decode(data)
}

func requireUint16(m Message, typ uint16) (uint16, error) {
	raw, ok := m.First(typ)
	if !ok {
		return 0, fmt.Errorf("ECMG response missing parameter 0x%04x", typ)
	}
	return Uint16Value(Parameter{Type: typ, Value: raw})
}
