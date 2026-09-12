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

## Phase 2 verification status

### Phase 2.12 Integration / CI — VERIFIED

The missing `NewRTPSequenceTracker()` constructor was added to `internal/transport/rtp.go` and committed as `d0fcec6a951b7d47c7bf868dd2d3a52ed35681f2`.

Validated on `inst05` after `git pull --ff-only`:

```text
go test ./...      PASS
go test -race ./... PASS
go build ./...     PASS
```

GitHub Actions **Go Test run #25** for commit `d0fcec6a951b7d47c7bf868dd2d3a52ed35681f2` completed successfully.

The local working tree is clean with respect to tracked repository files. `build/` is an untracked generated-output directory from the test fixture and is not treated as a source-code failure.

**Phase 2.12 status: 🟢 VERIFIED**

### TSDuck

Phase 1 fixture remains the bitstream acceptance gate and is verified on `inst05`.

### Live network test

Not yet run beyond the local UDP loopback integration test. Multicast/multi-NIC validation still requires the target Linux host/network interfaces and sources.

## Next step

### Phase 2.13 — Explicit multi-NIC egress interface selection

Add explicit outgoing-interface selection for UDP unicast/multicast egress so a multi-NIC deployment can deterministically bind transport output to the requested interface/address. Verify with Linux interface-level tests before moving to higher-density optimization.

Then proceed through Phase 2.14 live Linux network validation and Phase 2.15 transport-density optimization before Phase 3 PSI/SI regeneration and scheduling.

## Planned phases

- Phase 3 — PSI/SI regeneration and scheduling
- Phase 4 — DVB-SimulCrypt SCS/ECMG/EMMG protocol layer
- Phase 5 — ECM/EMM integration
- Phase 6 — DVB-CSA2 scrambling engine integration
- Phase 7 — 500+ service concurrency, NUMA/threading and performance tuning
- Phase 8 — Web API and real-time Web GUI
- Phase 9 — Docker deployment, observability and operational tooling
- Phase 10 — interoperability, fault-injection and acceptance testing
