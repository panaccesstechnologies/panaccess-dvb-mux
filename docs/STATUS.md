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
- Target-bitrate pacing primitive.
- Per-stream identity, protocol selection and atomic operational counters.

### Transport API layout

```text
internal/transport/
├── udp.go          # UDP/multicast ingest + TS datagram parsing
├── udp_sink.go     # UDP egress and TS aggregation
├── rtp.go          # RTP/TS parsing and sequence tracking
├── pipeline.go     # queue, batching and bitrate pacing
└── stream.go       # stream identity, queue and atomic statistics
```

## Verification status

**Go tests:** added for UDP/TS parsing, RTP parsing, sequence tracking, batching and pacing. The latest GitHub Actions workflow must be observed before declaring the suite green.  
**TSDuck:** Phase 1 fixture remains the bitstream acceptance gate.  
**Live network test:** not yet run from this chat; requires the target Linux host/network interfaces and multicast sources.

## Next step

Build the Phase 2 runtime worker that connects UDP/RTP ingest → bounded queue → transport processing → UDP egress, then add packet-level CC/PCR monitoring to that live path. After that, Phase 3 will own PSI/SI regeneration and scheduling.

## Planned phases

- Phase 3 — PSI/SI regeneration and scheduling
- Phase 4 — DVB-SimulCrypt SCS/ECMG/EMMG protocol layer
- Phase 5 — ECM/EMM integration
- Phase 6 — DVB-CSA2 scrambling engine integration
- Phase 7 — 500+ service concurrency, NUMA/threading and performance tuning
- Phase 8 — Web API and real-time Web GUI
- Phase 9 — Docker deployment, observability and operational tooling
- Phase 10 — interoperability, fault-injection and acceptance testing
