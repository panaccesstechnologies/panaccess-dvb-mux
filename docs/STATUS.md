# Project Status

**Project:** panaccess-dvb-mux  
**Status:** Phase 2 — IP transport foundation  
**Updated:** 2026-09-13

## Phase 1 status

The MPEG-TS foundation remains the correctness base: packet parsing, PAT/PMT sections, CRC, PSI packetization and PCR primitives are implemented. Deterministic fixture generation and GitHub Actions test execution are also wired in.

### Phase 1.7 TSDuck validation — VERIFIED

Validated on `inst05` with TSDuck `3.37-3670`.

Acceptance results:

- `tsp -I file build/testdata/phase1.ts -P analyze -O drop` recognized a valid MPEG transport stream with zero invalid sync bytes and zero transport errors.
- PAT on PID `0x0000` resolves program `100` to PMT PID `0x0100`.
- PMT on PID `0x0100` resolves PCR/video PID `0x0101` and audio PID `0x0102`.
- Video continuity/CC errors: 0; duplicated packets: 0; invalid PES starts: 0; PCR leaps: 0.
- `tsp -P tables` successfully decoded both PAT and PMT sections.

## Phase 2 progress

### Implemented

- Raw UDP unicast ingest.
- UDP multicast ingest with explicit interface selection.
- Validation of UDP datagram sizes as integral 188-byte TS packets.
- MPEG-TS datagram splitting without per-packet heap allocation in the parser.
- RTP v2 parsing with CSRC, extension and padding handling.
- RTP MPEG-TS payload validation.
- RTP sequence-gap and duplicate detection primitive.
- UDP unicast/multicast-capable egress through resolved UDP destinations.
- Configurable TS packet aggregation for egress, including 7 × 188 = 1316-byte batches.
- Bounded packet queue for ingest-to-processing back-pressure.
- Cancellation-aware queue insertion for clean runtime shutdown.
- Target-bitrate pacing primitive.
- Per-stream identity, protocol selection and atomic operational counters.
- Live `Runtime` worker connecting UDP/RTP ingest → queue → processing passthrough → UDP egress.
- Live TS monitoring for continuity-counter errors, PCR observations, malformed PCRs and backwards PCR movement.
- Loopback runtime integration test covering UDP ingest and egress.
- Phase 2.12 RTP sequence-tracker constructor fix.
- Phase 2.13 explicit UDP egress interface selection for unicast and multicast.
- Phase 2.14 `transport-live-test` command for reproducible Linux live-path validation.

## Phase 2 verification status

### Phase 2.12 Integration / CI — VERIFIED

The missing `NewRTPSequenceTracker()` constructor was added to `internal/transport/rtp.go` and committed as `d0fcec6a951b7d47c7bf868dd2d3a52ed35681f2`.

Validated on `inst05` after `git pull --ff-only`:

```text
go test ./...       PASS
go test -race ./... PASS
go build ./...      PASS
```

GitHub Actions **Go Test run #25** for commit `d0fcec6a951b7d47c7bf868dd2d3a52ed35681f2` completed successfully.

### Phase 2.13 Explicit multi-NIC egress interface selection — VERIFIED

The implementation was corrected to handle the two-value `UDPConn.SyscallConn()` API and was committed as `86df1a028780f7b02ad727f871522720d87136db`.

Validated on `inst05`:

```text
enp2s0           UP             192.168.10.25/24
go test ./...             PASS
go test -race ./...       PASS
go build ./...            PASS
```

Live multicast egress was captured on `enp2s0` with:

```text
192.168.10.25.48663 > 239.100.1.1.5000: UDP, length 1316
```

The capture recorded **562 packets**, with **0 packets dropped by kernel**. This verifies actual multicast egress using the selected `enp2s0` interface and 7 × 188-byte TS aggregation.

**Phase 2.13 status: 🟢 VERIFIED**

### Phase 2.14 Live Linux network validation — VERIFIED

Phase 2.14 was completed on `inst05` using the reproducible `transport-live-test` path with UDP input on port `6000`, multicast output `239.100.1.1:5000`, explicit output interface `enp2s0`, 4 Mbit/s target bitrate and 7-packet TS aggregation.

The final clean capture contained **92 UDP output packets** and was successfully consumed by TSDuck's native `pcap` input plugin. TSDuck analyzed **602 MPEG-TS packets / 113,176 bytes** with:

```text
Invalid sync:          0
Transport errors:      0
Suspect/ignored:       0
Scrambled packets:     0
```

The captured TS contained one service (service ID `100`), PAT PID `0x0000`, PMT PID `0x0100`, PCR/video PID `0x0101`, and AAC audio PID `0x0102`. Video had **300 packets** with **0 unexpected continuity errors**, audio had **300 packets** with **0 unexpected continuity errors**, PCR analysis reported **25 PCR values with 0 leaps**, and PAT/PMT sections were decoded successfully.

This verifies the live Linux multicast egress path at the MPEG-TS content level, including valid TS framing, PSI signaling, continuity counters and PCR behavior. The earlier truncated PCAP was superseded by the clean capture and is not used as acceptance evidence.

**Phase 2.14 status: 🟢 VERIFIED — LIVE LINUX NETWORK + TSDUCK VALIDATION**

## TSDuck

Phase 1 fixture remains the bitstream acceptance gate and is verified on `inst05`. The Phase 2.14 live multicast capture has now also been validated directly with TSDuck's `pcap` input plugin and `analyze` processor.

## Next step

### Phase 2.15 — High-density transport optimization

Proceed to transport-density work after preserving the Phase 2.14 evidence. The next validation focus should be sustained multi-service throughput, queue/back-pressure behavior, pacing accuracy, packet-loss/continuity monitoring under load, and scaling toward the planned 500+ service target.

**Specification compliance note:** the current implementation remains Go-based; the broader project requirement previously identified Rust or C++20 as the implementation target. This remains a separate compliance gap and should not be conflated with the successful Phase 2.14 functional validation.

## Planned phases

- Phase 2.15 — High-density transport optimization
- Phase 3 — PSI/SI regeneration and scheduling
- Phase 4 — DVB-SimulCrypt SCS/ECMG/EMMG protocol layer
- Phase 5 — ECM/EMM integration
- Phase 6 — DVB-CSA2 scrambling engine integration
- Phase 7 — 500+ service concurrency, NUMA/threading and performance tuning
- Phase 8 — Web API and real-time Web GUI
- Phase 9 — Docker deployment, observability and operational tooling
- Phase 10 — interoperability, fault-injection and acceptance testing
