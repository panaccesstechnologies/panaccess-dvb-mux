# panaccess-dvb-mux

High-density DVB IP Multiplexer and Scrambler project.

## Target

Build a carrier-grade software platform targeting 500+ concurrent SPTS services with:

- MPEG-TS/IP ingest and egress
- PSI/SI regeneration
- DVB-SimulCrypt integration (ECMG/EMMG)
- ECM/EMM handling
- DVB-CSA2 scrambling
- Per-service configuration and routing
- Real-time Web management and monitoring
- Linux multi-NIC operation
- Docker-based deployment and diagnostics

## Reference-driven development

Implementation will be developed incrementally and verified against the project specification and the attached DVB/ETSI reference documents. TSDuck will be used as a primary diagnostic and interoperability tool during development.

See [`docs/STATUS.md`](docs/STATUS.md) for the current implementation status and [`docs/REFERENCES.md`](docs/REFERENCES.md) for the reference set.
