# Project Status

**Project:** panaccess-dvb-mux  
**Status:** Phase 2 — IP transport foundation  
**Updated:** 2026-09-12

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
- Phase 2.14 runtime configuration now exposes `OutputInterface` and selects `NewUDPSinkWithInterface()` when configured, allowing the complete runtime path to explicitly bind egress to a selected Linux NIC.

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

## Phase 2.14 Live Linux network validation — IN PROGRESS

Phase 2.14 validates the complete live transport path on Linux using real UDP/RTP traffic rather than only loopback/unit tests.

Validation scope:

- UDP MPEG-TS ingest on a Linux network interface.
- RTP MPEG-TS ingest and sequence tracking.
- Bounded queue behavior under sustained traffic.
- Continuity-counter monitoring.
- PCR observation and malformed/backwards-PCR detection.
- UDP multicast egress through an explicitly selected interface.
- 1316-byte (7 × 188-byte) TS aggregation.
- CBR pacing behavior.
- Runtime cancellation and clean socket shutdown.
- Packet capture evidence for input and output traffic.

### Phase 2.14 implementation step — runtime egress interface selection

`internal/transport/worker.go` now has an `OutputInterface` configuration field. `NewRuntime()` uses `NewUDPSinkWithInterface()` when this field is non-empty, while preserving the existing default behavior when it is empty.

Implementation commit:

`9d1c398f557e80ec9654f43f98dbb53caa87480e`

The earlier live capture on `inst05` demonstrated the selected-interface egress path, but the capture contained **2172 output packets on UDP port 5000 and 0 packets on UDP port 6000**. Therefore it did not yet prove the complete ingest → queue → monitor → egress runtime path.

**Phase 2.14 status: 🟡 IN PROGRESS — LIVE LINUX VALIDATION**

## TSDuck

Phase 1 fixture remains the bitstream acceptance gate and is verified on `inst05`.

## Next step

### Phase 2.14 live end-to-end validation

Pull the runtime-interface change on `inst05`, run the runtime with a controlled UDP or RTP MPEG-TS source on input port `6000`, configure egress to `239.100.1.1:5000` with `OutputInterface=enp2s0`, and capture both paths. Verify input/output packet counts, 1316-byte aggregation, TS integrity, continuity counters, PCR behavior, pacing, and clean cancellation before marking Phase 2.14 VERIFIED.

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
