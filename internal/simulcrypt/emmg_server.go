package simulcrypt

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	EMMGErrInvalidMessage       uint16 = 0x0001
	EMMGErrUnsupportedVersion   uint16 = 0x0002
	EMMGErrUnknownMessage       uint16 = 0x0003
	EMMGErrMessageTooLong       uint16 = 0x0004
	EMMGErrUnknownStream        uint16 = 0x0005
	EMMGErrUnknownChannel       uint16 = 0x0006
	EMMGErrTooManyChannels      uint16 = 0x0007
	EMMGErrTooManyStreams       uint16 = 0x0008
	EMMGErrTooManyStreamsMUX    uint16 = 0x0009
	EMMGErrUnknownParameter     uint16 = 0x000A
	EMMGErrBadParameterLength   uint16 = 0x000B
	EMMGErrMissingParameter     uint16 = 0x000C
	EMMGErrInvalidParameter     uint16 = 0x000D
	EMMGErrUnknownClient        uint16 = 0x000E
	EMMGErrExceededBandwidth    uint16 = 0x000F
	EMMGErrUnknownDataID        uint16 = 0x0010
	EMMGErrChannelInUse         uint16 = 0x0011
	EMMGErrStreamInUse          uint16 = 0x0012
	EMMGErrDataIDInUse          uint16 = 0x0013
	EMMGErrClientInUse          uint16 = 0x0014
	EMMGErrUnknown              uint16 = 0x7000
	EMMGErrUnrecoverable        uint16 = 0x7001
)

type EMMGData struct {
	ClientID         uint32
	DataChannelID    uint16
	DataStreamID     uint16
	DataID           uint16
	DataType         byte
	SectionTSPktFlag byte
	Datagrams        [][]byte
}

type EMMGDataHandler func(EMMGData) error

type emmgChannelKey struct {
	clientID uint32
	channel  uint16
}

type emmgStreamKey struct {
	clientID uint32
	dataID   uint16
}

type emmgStream struct {
	clientID      uint32
	channelID     uint16
	streamID      uint16
	dataID        uint16
	dataType      byte
	sectionFlag   byte
	bandwidth     uint16
}

type EMMGServer struct {
	Addr      string
	Handler   EMMGDataHandler
	Bandwidth uint16
	IOTimeout time.Duration

	mu      sync.Mutex
	ln      net.Listener
	udp     *net.UDPConn
	done    chan struct{}
	channels map[emmgChannelKey]*emmgConnection
	streams  map[emmgStreamKey]*emmgStream
}

func NewEMMGServer(addr string, handler EMMGDataHandler) *EMMGServer {
	return &EMMGServer{
		Addr:      addr,
		Handler:   handler,
		IOTimeout: 10 * time.Second,
		done:      make(chan struct{}),
		channels:  make(map[emmgChannelKey]*emmgConnection),
		streams:   make(map[emmgStreamKey]*emmgStream),
	}
}

func (s *EMMGServer) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	udp, err := net.ListenUDP("udp", udpAddr(s.Addr))
	if err != nil {
		_ = ln.Close()
		return err
	}

	s.mu.Lock()
	s.ln, s.udp = ln, udp
	s.mu.Unlock()

	go s.serveUDP(udp)
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
			}
			return err
		}
		go s.serveTCP(conn)
	}
}

func (s *EMMGServer) Close() error {
	s.mu.Lock()
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	ln, udp := s.ln, s.udp
	s.ln, s.udp = nil, nil
	s.channels = make(map[emmgChannelKey]*emmgConnection)
	s.streams = make(map[emmgStreamKey]*emmgStream)
	s.mu.Unlock()

	var first error
	if ln != nil {
		if err := ln.Close(); err != nil {
			first = err
		}
	}
	if udp != nil {
		if err := udp.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

type emmgConnection struct {
	server          *EMMGServer
	conn            net.Conn
	clientID        uint32
	dataChannelID   uint16
	sectionTSPktFlag byte
	channelReady    bool
	dataStreamID    uint16
	dataID          uint16
	dataType        byte
	streamReady     bool
}

func (s *EMMGServer) serveTCP(conn net.Conn) {
	c := &emmgConnection{server: s, conn: conn}
	defer func() {
		s.cleanupConnection(c)
		_ = conn.Close()
	}()

	for {
		if s.IOTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.IOTimeout))
		}
		msg, err := readEMMGMessage(conn)
		if err != nil {
			if err != io.EOF {
				_ = c.writeError(EMMGMsgChannelError, EMMGErrInvalidMessage, err.Error())
			}
			return
		}
		if msg.ProtocolVersion != EMMGProtocolVersion {
			_ = c.writeError(EMMGMsgChannelError, EMMGErrUnsupportedVersion, "unsupported protocol version")
			return
		}

		var keep bool
		switch msg.Type {
		case EMMGMsgChannelSetup:
			keep = c.handleChannelSetup(msg)
		case EMMGMsgChannelTest:
			keep = c.handleChannelTest(msg)
		case EMMGMsgChannelClose:
			keep = c.handleChannelClose(msg)
		case EMMGMsgStreamSetup:
			keep = c.handleStreamSetup(msg)
		case EMMGMsgStreamTest:
			keep = c.handleStreamTest(msg)
		case EMMGMsgStreamCloseReq:
			keep = c.handleStreamClose(msg)
		case EMMGMsgStreamBWRequest:
			keep = c.handleBandwidthRequest(msg)
		case EMMGMsgDataProvision:
			keep = c.handleDataProvision(msg)
		default:
			// ETSI TS 103 197: unknown message types shall be ignored.
			continue
		}
		if !keep {
			return
		}
	}
}

func (s *EMMGServer) serveUDP(conn *net.UDPConn) {
	buf := make([]byte, 64*1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-s.done:
				return
			default:
			}
			return
		}
		msg, err := Decode(buf[:n])
		if err != nil || msg.ProtocolVersion != EMMGProtocolVersion || msg.Type != EMMGMsgDataProvision {
			continue
		}
		if clientRaw, ok := msg.First(EMMGParamClientID); ok {
			clientID, e := Uint32Value(Parameter{Type: EMMGParamClientID, Value: clientRaw})
			if e != nil {
				continue
			}
			dataIDRaw, ok := msg.First(EMMGParamDataID)
			if !ok {
				continue
			}
			dataID, e := Uint16Value(Parameter{Type: EMMGParamDataID, Value: dataIDRaw})
			if e != nil {
				continue
			}
			if _, ok := msg.First(EMMGParamDataChannelID); ok {
				continue
			}
			if _, ok := msg.First(EMMGParamDataStreamID); ok {
				continue
			}
			datagrams := parameterValues(msg, EMMGParamDatagram)
			if len(datagrams) == 0 {
				continue
			}
			s.mu.Lock()
			st := s.streams[emmgStreamKey{clientID: clientID, dataID: dataID}]
			s.mu.Unlock()
			if st == nil {
				continue
			}
			if err := validateDatagrams(st.sectionFlag, datagrams); err != nil {
				continue
			}
			if s.Handler != nil {
				_ = s.Handler(EMMGData{
					ClientID: clientID, DataChannelID: st.channelID, DataStreamID: st.streamID,
					DataID: dataID, DataType: st.dataType, SectionTSPktFlag: st.sectionFlag,
					Datagrams: datagrams,
				})
			}
		}
		_ = addr
	}
}

func (c *emmgConnection) handleChannelSetup(m Message) bool {
	client, ok := getUint32(m, EMMGParamClientID)
	channel, ok2 := getUint16(m, EMMGParamDataChannelID)
	flag, ok3 := getUint8(m, EMMGParamSectionTSPktFlag)
	if !ok || !ok2 || !ok3 {
		return c.fail(EMMGMsgChannelError, EMMGErrMissingParameter, "missing channel setup parameter")
	}
	if flag != 0x00 && flag != 0x01 {
		return c.fail(EMMGMsgChannelError, EMMGErrInvalidParameter, "unsupported section_TSpkt_flag")
	}

	c.server.mu.Lock()
	key := emmgChannelKey{clientID: client, channel: channel}
	if c.server.channels[key] != nil && c.server.channels[key] != c {
		c.server.mu.Unlock()
		return c.fail(EMMGMsgChannelError, EMMGErrChannelInUse, "channel already in use")
	}
	c.server.channels[key] = c
	c.server.mu.Unlock()

	c.clientID, c.dataChannelID, c.sectionTSPktFlag, c.channelReady = client, channel, flag, true
	return c.write(NewMessage(EMMGMsgChannelStatus,
		Uint32Parameter(EMMGParamClientID, client),
		Uint16Parameter(EMMGParamDataChannelID, channel),
		Uint8Parameter(EMMGParamSectionTSPktFlag, flag),
	)) == nil
}

func (c *emmgConnection) handleChannelTest(m Message) bool {
	client, ok := getUint32(m, EMMGParamClientID)
	channel, ok2 := getUint16(m, EMMGParamDataChannelID)
	if !ok || !ok2 {
		return c.fail(EMMGMsgChannelError, EMMGErrMissingParameter, "missing channel test parameter")
	}
	if !c.channelReady || client != c.clientID || channel != c.dataChannelID {
		return c.fail(EMMGMsgChannelError, EMMGErrUnknownChannel, "channel is not open")
	}
	return c.write(NewMessage(EMMGMsgChannelStatus,
		Uint32Parameter(EMMGParamClientID, c.clientID),
		Uint16Parameter(EMMGParamDataChannelID, c.dataChannelID),
		Uint8Parameter(EMMGParamSectionTSPktFlag, c.sectionTSPktFlag),
	)) == nil
}

func (c *emmgConnection) handleChannelClose(m Message) bool {
	if c.streamReady {
		c.removeStream()
	}
	if c.channelReady {
		c.server.mu.Lock()
		delete(c.server.channels, emmgChannelKey{clientID: c.clientID, channel: c.dataChannelID})
		c.server.mu.Unlock()
		c.channelReady = false
	}
	return true
}

func (c *emmgConnection) handleStreamSetup(m Message) bool {
	if !c.channelReady {
		return c.fail(EMMGMsgStreamError, EMMGErrUnknownChannel, "channel is not open")
	}
	client, ok := getUint32(m, EMMGParamClientID)
	channel, ok2 := getUint16(m, EMMGParamDataChannelID)
	stream, ok3 := getUint16(m, EMMGParamDataStreamID)
	dataID, ok4 := getUint16(m, EMMGParamDataID)
	dataType, ok5 := getUint8(m, EMMGParamDataType)
	if !ok || !ok2 || !ok3 || !ok4 || !ok5 {
		return c.fail(EMMGMsgStreamError, EMMGErrMissingParameter, "missing stream setup parameter")
	}
	if client != c.clientID || channel != c.dataChannelID {
		return c.fail(EMMGMsgStreamError, EMMGErrInvalidParameter, "channel identity mismatch")
	}
	if dataType != 0x00 && dataType != 0x01 {
		return c.fail(EMMGMsgStreamError, EMMGErrInvalidParameter, "invalid data_type")
	}

	c.server.mu.Lock()
	key := emmgStreamKey{clientID: client, dataID: dataID}
	if existing := c.server.streams[key]; existing != nil {
		c.server.mu.Unlock()
		return c.fail(EMMGMsgStreamError, EMMGErrDataIDInUse, "data_id already in use")
	}
	c.server.streams[key] = &emmgStream{
		clientID: client, channelID: channel, streamID: stream, dataID: dataID,
		dataType: dataType, sectionFlag: c.sectionTSPktFlag,
	}
	c.server.mu.Unlock()

	c.dataStreamID, c.dataID, c.dataType, c.streamReady = stream, dataID, dataType, true
	return c.write(NewMessage(EMMGMsgStreamStatus,
		Uint32Parameter(EMMGParamClientID, client),
		Uint16Parameter(EMMGParamDataChannelID, channel),
		Uint16Parameter(EMMGParamDataStreamID, stream),
		Uint16Parameter(EMMGParamDataID, dataID),
		Uint8Parameter(EMMGParamDataType, dataType),
	)) == nil
}

func (c *emmgConnection) handleStreamTest(m Message) bool {
	client, ok := getUint32(m, EMMGParamClientID)
	channel, ok2 := getUint16(m, EMMGParamDataChannelID)
	stream, ok3 := getUint16(m, EMMGParamDataStreamID)
	if !ok || !ok2 || !ok3 {
		return c.fail(EMMGMsgStreamError, EMMGErrMissingParameter, "missing stream test parameter")
	}
	if !c.streamReady || client != c.clientID || channel != c.dataChannelID || stream != c.dataStreamID {
		return c.fail(EMMGMsgStreamError, EMMGErrUnknownStream, "stream is not open")
	}
	return c.write(NewMessage(EMMGMsgStreamStatus,
		Uint32Parameter(EMMGParamClientID, c.clientID),
		Uint16Parameter(EMMGParamDataChannelID, c.dataChannelID),
		Uint16Parameter(EMMGParamDataStreamID, c.dataStreamID),
		Uint16Parameter(EMMGParamDataID, c.dataID),
		Uint8Parameter(EMMGParamDataType, c.dataType),
	)) == nil
}

func (c *emmgConnection) handleStreamClose(m Message) bool {
	client, ok := getUint32(m, EMMGParamClientID)
	channel, ok2 := getUint16(m, EMMGParamDataChannelID)
	stream, ok3 := getUint16(m, EMMGParamDataStreamID)
	if !ok || !ok2 || !ok3 {
		return c.fail(EMMGMsgStreamError, EMMGErrMissingParameter, "missing stream close parameter")
	}
	if !c.streamReady || client != c.clientID || channel != c.dataChannelID || stream != c.dataStreamID {
		return c.fail(EMMGMsgStreamError, EMMGErrUnknownStream, "stream is not open")
	}
	if err := c.write(NewMessage(EMMGMsgStreamCloseResp,
		Uint32Parameter(EMMGParamClientID, c.clientID),
		Uint16Parameter(EMMGParamDataChannelID, c.dataChannelID),
		Uint16Parameter(EMMGParamDataStreamID, c.dataStreamID),
	)); err != nil {
		return false
	}
	c.removeStream()
	return true
}

func (c *emmgConnection) handleBandwidthRequest(m Message) bool {
	if !c.streamReady {
		return c.fail(EMMGMsgStreamError, EMMGErrUnknownStream, "stream is not open")
	}
	client, ok := getUint32(m, EMMGParamClientID)
	channel, ok2 := getUint16(m, EMMGParamDataChannelID)
	stream, ok3 := getUint16(m, EMMGParamDataStreamID)
	if !ok || !ok2 || !ok3 || client != c.clientID || channel != c.dataChannelID || stream != c.dataStreamID {
		return c.fail(EMMGMsgStreamError, EMMGErrInvalidParameter, "bandwidth request identity mismatch")
	}
	requested, present := getUint16(m, EMMGParamBandwidth)
	if present {
		if c.server.Bandwidth != 0 && requested > c.server.Bandwidth {
			return c.fail(EMMGMsgStreamError, EMMGErrExceededBandwidth, "requested bandwidth exceeds allocation")
		}
		c.server.mu.Lock()
		if st := c.server.streams[emmgStreamKey{clientID: c.clientID, dataID: c.dataID}]; st != nil {
			st.bandwidth = requested
		}
		c.server.mu.Unlock()
	}
	var allocated uint16
	c.server.mu.Lock()
	if st := c.server.streams[emmgStreamKey{clientID: c.clientID, dataID: c.dataID}]; st != nil {
		allocated = st.bandwidth
	}
	c.server.mu.Unlock()

	params := []Parameter{
		Uint32Parameter(EMMGParamClientID, c.clientID),
		Uint16Parameter(EMMGParamDataChannelID, c.dataChannelID),
		Uint16Parameter(EMMGParamDataStreamID, c.dataStreamID),
	}
	if allocated != 0 {
		params = append(params, Uint16Parameter(EMMGParamBandwidth, allocated))
	}
	return c.write(NewMessage(EMMGMsgStreamBWAlloc, params...)) == nil
}

func (c *emmgConnection) handleDataProvision(m Message) bool {
	if !c.streamReady {
		return c.fail(EMMGMsgStreamError, EMMGErrUnknownStream, "stream is not open")
	}
	client, ok := getUint32(m, EMMGParamClientID)
	channel, ok2 := getUint16(m, EMMGParamDataChannelID)
	stream, ok3 := getUint16(m, EMMGParamDataStreamID)
	dataID, ok4 := getUint16(m, EMMGParamDataID)
	if !ok || !ok2 || !ok3 || !ok4 {
		return c.fail(EMMGMsgStreamError, EMMGErrMissingParameter, "missing data_provision parameter")
	}
	if client != c.clientID || channel != c.dataChannelID || stream != c.dataStreamID || dataID != c.dataID {
		return c.fail(EMMGMsgStreamError, EMMGErrInvalidParameter, "data_provision identity mismatch")
	}
	datagrams := parameterValues(m, EMMGParamDatagram)
	if len(datagrams) == 0 {
		return c.fail(EMMGMsgStreamError, EMMGErrMissingParameter, "missing datagram")
	}
	if err := validateDatagrams(c.sectionTSPktFlag, datagrams); err != nil {
		return c.fail(EMMGMsgStreamError, EMMGErrInvalidParameter, err.Error())
	}
	if c.server.Handler != nil {
		if err := c.server.Handler(EMMGData{
			ClientID: client, DataChannelID: channel, DataStreamID: stream, DataID: dataID,
			DataType: c.dataType, SectionTSPktFlag: c.sectionTSPktFlag, Datagrams: datagrams,
		}); err != nil {
			return c.fail(EMMGMsgStreamError, EMMGErrUnknown, err.Error())
		}
	}
	return true
}

func (c *emmgConnection) write(m Message) error {
	if c.server.IOTimeout > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.server.IOTimeout))
	}
	b, err := Encode(m)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(b)
	return err
}

func (c *emmgConnection) writeError(messageType, status uint16, info string) error {
	return c.write(NewMessage(messageType,
		Uint16Parameter(ParamErrorStatus, status),
		BytesParameter(ParamErrorInformation, []byte(info)),
	))
}

func (c *emmgConnection) fail(messageType, status uint16, info string) bool {
	_ = c.writeError(messageType, status, info)
	return false
}

func (c *emmgConnection) removeStream() {
	if !c.streamReady {
		return
	}
	c.server.mu.Lock()
	delete(c.server.streams, emmgStreamKey{clientID: c.clientID, dataID: c.dataID})
	c.server.mu.Unlock()
	c.streamReady = false
}

func (s *EMMGServer) cleanupConnection(c *emmgConnection) {
	c.removeStream()
	if c.channelReady {
		s.mu.Lock()
		delete(s.channels, emmgChannelKey{clientID: c.clientID, channel: c.dataChannelID})
		s.mu.Unlock()
		c.channelReady = false
	}
}

func readEMMGMessage(r io.Reader) (Message, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(r, header); err != nil {
		return Message{}, err
	}
	bodyLen := int(header[3])<<8 | int(header[4])
	if bodyLen > 65535 {
		return Message{}, fmt.Errorf("message too long")
	}
	body := make([]byte, 5+bodyLen)
	copy(body, header)
	if _, err := io.ReadFull(r, body[5:]); err != nil {
		return Message{}, err
	}
	return Decode(body)
}

func udpAddr(addr string) *net.UDPAddr {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return &net.UDPAddr{Port: 0}
	}
	p, err := net.LookupPort("udp", port)
	if err != nil {
		p = 0
	}
	if host == "" {
		return &net.UDPAddr{Port: p}
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return &net.UDPAddr{Port: p}
	}
	return &net.UDPAddr{IP: ip, Port: p}
}

func getUint8(m Message, typ uint16) (byte, bool) {
	p, ok := m.First(typ)
	if !ok || len(p) != 1 {
		return 0, false
	}
	return p[0], true
}

func getUint16(m Message, typ uint16) (uint16, bool) {
	p, ok := m.First(typ)
	if !ok || len(p) != 2 {
		return 0, false
	}
	v, err := Uint16Value(Parameter{Type: typ, Value: p})
	return v, err == nil
}

func getUint32(m Message, typ uint16) (uint32, bool) {
	p, ok := m.First(typ)
	if !ok || len(p) != 4 {
		return 0, false
	}
	v, err := Uint32Value(Parameter{Type: typ, Value: p})
	return v, err == nil
}

func parameterValues(m Message, typ uint16) [][]byte {
	var out [][]byte
	for _, p := range m.Parameters {
		if p.Type == typ {
			out = append(out, append([]byte(nil), p.Value...))
		}
	}
	return out
}

func validateDatagrams(flag byte, datagrams [][]byte) error {
	switch flag {
	case 0x00:
		for _, d := range datagrams {
			if len(d) == 0 {
				return fmt.Errorf("empty section datagram")
			}
		}
	case 0x01:
		for _, d := range datagrams {
			if len(d) != 188 {
				return fmt.Errorf("TS packet must be exactly 188 bytes")
			}
			if d[0] != 0x47 {
				return fmt.Errorf("invalid MPEG-2 TS sync byte")
			}
		}
	default:
		return fmt.Errorf("unsupported section_TSpkt_flag 0x%02X", flag)
	}
	return nil
}
