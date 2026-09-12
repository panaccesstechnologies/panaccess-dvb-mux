package main

import (
	"fmt"
	"os"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

const (
	pmtPID   = 0x0100
	videoPID = 0x0101
	audioPID = 0x0102
)

func makeESPacket(pid uint16, cc uint8, pusi bool, payload []byte, pcr uint64, withPCR bool) mpegts.Packet {
	var p mpegts.Packet
	p.Data[0] = mpegts.SyncByte
	p.Data[1] = byte(pid >> 8)
	p.Data[2] = byte(pid)
	if pusi { p.Data[1] |= 0x40 }
	p.Data[3] = 0x10 | (cc & 0x0f)

	off := 4
	if withPCR {
		p.Data[3] = 0x30 | (cc & 0x0f)
		p.Data[4] = 7
		p.Data[5] = 0x10
		pcrBytes := mpegts.EncodePCR(pcr)
		copy(p.Data[6:12], pcrBytes[:])
		off = 12
	}
	if len(payload) > mpegts.PacketSize-off { payload = payload[:mpegts.PacketSize-off] }
	copy(p.Data[off:], payload)
	for i := off + len(payload); i < mpegts.PacketSize; i++ { p.Data[i] = 0xff }
	return p
}

func pesPayload(streamID byte, seed byte, size int) []byte {
	if size < 14 { size = 14 }
	b := make([]byte, size)
	b[0], b[1], b[2] = 0x00, 0x00, 0x01
	b[3] = streamID
	b[4], b[5] = 0, 0 // Unbounded PES payload for deterministic transport fixture.
	b[6], b[7], b[8] = 0x80, 0x00, 0x00
	for i := 9; i < len(b); i++ { b[i] = seed + byte(i) }
	return b
}

func writePacket(f *os.File, p mpegts.Packet) error {
	_, err := f.Write(p.Data[:])
	return err
}

func main() {
	out := "build/testdata/phase1.ts"
	if len(os.Args) > 1 { out = os.Args[1] }
	if err := os.MkdirAll("build/testdata", 0o755); err != nil { panic(err) }
	f, err := os.Create(out)
	if err != nil { panic(err) }
	defer f.Close()

	pat, err := (mpegts.PAT{
		TSID: 1, Version: 0, Current: true,
		Programs: []mpegts.PATProgram{{Number: 0, NetworkPID: mpegts.NITPID}, {Number: 100, PMTPID: pmtPID}},
	}).MarshalSection()
	if err != nil { panic(err) }
	patPackets, err := mpegts.PacketizeSection(mpegts.PATPID, pat, 0)
	if err != nil { panic(err) }
	for _, p := range patPackets { if err := writePacket(f, p); err != nil { panic(err) } }

	pmt, err := (mpegts.PMT{
		ProgramNumber: 100, Version: 0, Current: true, PCRPID: videoPID,
		Streams: []mpegts.PMTStream{{StreamType: 0x1b, PID: videoPID}, {StreamType: 0x0f, PID: audioPID}},
	}).MarshalSection()
	if err != nil { panic(err) }
	pmtPackets, err := mpegts.PacketizeSection(pmtPID, pmt, 0)
	if err != nil { panic(err) }
	for _, p := range pmtPackets { if err := writePacket(f, p); err != nil { panic(err) } }

	var videoCC, audioCC uint8
	var pcr uint64
	for i := 0; i < 300; i++ {
		video := pesPayload(0xe0, byte(i), 176)
		vp := makeESPacket(videoPID, videoCC, i%12 == 0, video, pcr, i%12 == 0)
		if err := writePacket(f, vp); err != nil { panic(err) }
		videoCC = (videoCC + 1) & 0x0f
		pcr += 3600

		audio := pesPayload(0xc0, byte(i+17), 184)
		ap := makeESPacket(audioPID, audioCC, i%12 == 0, audio, 0, false)
		if err := writePacket(f, ap); err != nil { panic(err) }
		audioCC = (audioCC + 1) & 0x0f
	}

	info, err := f.Stat()
	if err != nil { panic(err) }
	fmt.Printf("wrote %s: %d bytes (%d TS packets)\n", out, info.Size(), info.Size()/mpegts.PacketSize)
}
