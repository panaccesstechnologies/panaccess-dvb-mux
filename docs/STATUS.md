# Project Status

**Project:** panaccess-dvb-mux  
**Status:** Phase 0 — repository and reference baseline initialized  
**Updated:** 2026-09-12

## Completed

- GitHub repository initialized.
- Project target recorded: 500+ concurrent SPTS DVB IP multiplexing and scrambling.
- Initial reference set reviewed:
  - TSDuck User Guide — operational/diagnostic reference.
  - ETSI EN 300 468 V1.18.1 (2023-12) — DVB Service Information (SI).
  - ETSI TS 102 470-2 V1.2.1 (2011-09) — IP Datacast PSI/SI for DVB-SH.
- Initial repository documentation added.

## Reference-driven observations

ETSI EN 300 468 describes SI used to help receivers select services/events and configure themselves. Its table coverage includes NIT, BAT, SDT, EIT, TDT, TOT, RST and related tables/descriptors.

The ETSI TS 102 470-2 reference specifically documents PSI/SI usage in DVB-SH systems, including PAT, PMT, CAT, TSDT, NIT and SDT requirements.

For the multiplexer, this means PSI/SI generation and regeneration will be treated as a first-class pipeline component rather than an afterthought.

## Next implementation phase

**Phase 1 — Core MPEG-TS model and pipeline skeleton**

1. Define transport-stream packet primitives and validation.
2. Define service/program/PID configuration models.
3. Implement PAT/PMT parsing and generation foundations.
4. Add continuity-counter and PCR tracking primitives.
5. Add deterministic test vectors and TSDuck-based diagnostics.
6. Establish the architecture needed to scale from a small test configuration to 500+ SPTS services.

## Planned phases

- Phase 2 — IP multicast/RTP ingest and egress
- Phase 3 — PSI/SI regeneration and scheduling
- Phase 4 — DVB-SimulCrypt SCS/ECMG/EMMG protocol layer
- Phase 5 — ECM/EMM integration
- Phase 6 — DVB-CSA2 scrambling engine integration
- Phase 7 — 500+ service concurrency, NUMA/threading and performance tuning
- Phase 8 — Web API and real-time Web GUI
- Phase 9 — Docker deployment, observability and operational tooling
- Phase 10 — interoperability, fault-injection and acceptance testing

## Important development rule

Do not mark a subsystem complete only because it compiles. Each major subsystem must have protocol/bitstream tests and TSDuck diagnostics before being considered production-ready.
