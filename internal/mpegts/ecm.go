package mpegts

import "fmt"

// ECMInjector packetizes ECM sections onto the configured ECM PID while
// maintaining an independent MPEG-TS continuity counter for that PID.
type ECMInjector struct {
	PID uint16
	CC  uint8
}

func NewECMInjector(pid uint16, cc uint8) (*ECMInjector, error) {
	if pid > 0x1fff {
		return nil, fmt.Errorf("invalid ECM PID")
	}
	return &ECMInjector{PID: pid, CC: cc & 0x0f}, nil
}

// InjectSection converts one ECM section/datagram into TS packets and advances
// the ECM PID continuity counter. The returned packets are ready for mux
// scheduling; no payload scrambling is performed here.
func (e *ECMInjector) InjectSection(section []byte) ([]Packet, error) {
	if e == nil {
		return nil, fmt.Errorf("nil ECM injector")
	}
	packets, err := PacketizeSection(e.PID, section, e.CC)
	if err != nil {
		return nil, err
	}
	e.CC = (e.CC + uint8(len(packets))) & 0x0f
	return packets, nil
}
