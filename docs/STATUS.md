# Project Status

**Project:** panaccess-dvb-mux  
**Status:** Phase 2 — IP transport foundation  
**Updated:** 2026-09-12

## Phase 1 status

The MPEG-TS foundation remains the correctness base: packet parsing, PAT/PMT sections, CRC, PSI packetization and PCR primitives are implemented. Deterministic fixture generation and GitHub Actions test execution are also wired in.

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

### Transport API layout

```text
internal/transport/
├── udp.go           # UDP/multicast ingest + TS datagram parsing
├── udp_sink.go      # UDP egress and TS aggregation
├── rtp.go           # RTP/TS parsing and sequence tracking
├── pipeline.go      # queue, batching and bitrate pacing
├── monitor.go       # live CC/PCR telemetry
├── stream.go        # stream identity, queue, RTP tracking and stats
├── worker.go        # live ingest → queue → monitor → egress runtime
└── worker_test.go   # UDP loopback runtime integration test
```

## Verification status

**Go tests:** tests are present for UDP/TS parsing, RTP parsing, sequence tracking, batching, pacing, CC/PCR monitoring and the UDP loopback runtime. The latest GitHub Actions workflow must be observed before declaring the suite green.  
**TSDuck:** Phase 1 fixture remains the bitstream acceptance gate.  
**Live network test:** not yet run from this chat; the loopback integration test is the first automated live-path check, while multicast/multi-NIC validation still requires the target Linux host/network interfaces and sources.

## Next step

Harden the Phase 2 runtime for production transport behavior: explicit egress-interface selection for multi-NIC deployments, packet-level CC/PCR policy and counters, improved pacing/buffer handling, and higher-density allocation paths. Then Phase 3 will own PSI/SI regeneration and scheduling.

## Planned phases

- Phase 3 — PSI/SI regeneration and scheduling
- Phase 4 — DVB-SimulCrypt SCS/ECMG/EMMG protocol layer
- Phase 5 — ECM/EMM integration
- Phase 6 — DVB-CSA2 scrambling engine integration
- Phase 7 — 500+ service concurrency, NUMA/threading and performance tuning
- Phase 8 — Web API and real-time Web GUI
- Phase 9 — Docker deployment, observability and operational tooling
- Phase 10 — interoperability, fault-injection and acceptance testing
