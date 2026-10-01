package service

import (
	"fmt"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/simulcrypt"
)

// ECMInserter binds one discovered CA endpoint to an MPEG-TS ECM injector.
// It is the bridge between an ECMG response and the service's ECM PID.
type ECMInserter struct {
	Endpoint CAEndpoint
	Injector *mpegts.ECMInjector
}

func NewECMInserter(s Service, caSystemID uint16, initialCC uint8) (*ECMInserter, error) {
	for _, endpoint := range s.ECM {
		if endpoint.CASystemID == caSystemID {
			injector, err := mpegts.NewECMInjector(endpoint.PID, initialCC)
			if err != nil { return nil, err }
			return &ECMInserter{Endpoint: endpoint, Injector: injector}, nil
		}
	}
	return nil, fmt.Errorf("CA system ID 0x%04x is not signalled as an ECM endpoint", caSystemID)
}

// InjectResponse packetizes the ECM datagram returned by ECMG onto the
// discovered ECM PID and advances that PID's continuity counter.
func (e *ECMInserter) InjectResponse(resp simulcrypt.ECMResponse) ([]mpegts.Packet, error) {
	if e == nil || e.Injector == nil {
		return nil, fmt.Errorf("nil ECM inserter")
	}
	if resp.ChannelID == 0 || resp.StreamID == 0 {
		return nil, fmt.Errorf("invalid ECM response channel/stream identity")
	}
	if len(resp.ECMDatagram) == 0 {
		return nil, fmt.Errorf("empty ECM datagram")
	}
	return e.Injector.InjectSection(resp.ECMDatagram)
}
