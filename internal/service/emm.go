package service

import (
	"fmt"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

const (
	EMMSectionFormat  byte = 0x00
	EMMTSPacketFormat byte = 0x01
)

// EMMInserter packetizes EMM datagrams onto the EMM PID discovered from CAT.
type EMMInserter struct {
	Endpoint        CAEndpoint
	Injector        *mpegts.EMMInjector
	SectionTSPktFlag byte
}

func NewEMMInserter(s Service, caSystemID uint16, initialCC uint8) (*EMMInserter, error) {
	return NewEMMInserterWithFormat(s, caSystemID, initialCC, EMMSectionFormat)
}

// NewEMMInserterWithFormat creates an EMM inserter for the ETSI
// section_TSpkt_flag format negotiated on the EMMG channel.
func NewEMMInserterWithFormat(s Service, caSystemID uint16, initialCC uint8, sectionTSPktFlag byte) (*EMMInserter, error) {
	if sectionTSPktFlag != EMMSectionFormat && sectionTSPktFlag != EMMTSPacketFormat {
		return nil, fmt.Errorf("unsupported EMM section_TSpkt_flag: 0x%02x", sectionTSPktFlag)
	}
	for _, ep := range s.EMM {
		if ep.CASystemID == caSystemID {
			inj, err := mpegts.NewEMMInjector(ep.PID, initialCC)
			if err != nil { return nil, err }
			return &EMMInserter{
				Endpoint: ep, Injector: inj, SectionTSPktFlag: sectionTSPktFlag,
			}, nil
		}
	}
	return nil, fmt.Errorf("EMM CA system 0x%04x not found", caSystemID)
}

// InjectDatagram inserts one EMM datagram using the ETSI channel format
// negotiated by section_TSpkt_flag.
func (e *EMMInserter) InjectDatagram(datagram []byte) ([]mpegts.Packet, error) {
	if e == nil || e.Injector == nil {
		return nil, fmt.Errorf("nil EMM inserter")
	}
	if len(datagram) == 0 {
		return nil, fmt.Errorf("empty EMM datagram")
	}
	switch e.SectionTSPktFlag {
	case EMMSectionFormat:
		return e.Injector.InjectSection(datagram)
	case EMMTSPacketFormat:
		return e.Injector.InjectTSPacket(datagram)
	default:
		return nil, fmt.Errorf("unsupported EMM section_TSpkt_flag: 0x%02x", e.SectionTSPktFlag)
	}
}
