# Phase 1 validation

Phase 1 validates the MPEG-TS foundation before moving into service routing and scrambling.

## Generate deterministic fixture

From the repository root:

```bash
go run ./cmd/tsfixture
```

This creates `build/testdata/phase1.ts` containing:

- PAT on PID `0x0000`.
- PAT network/NIT PID entry `0x0010`.
- Program 100 with PMT on PID `0x0100`.
- H.264 video stream on PID `0x0101`.
- MPEG-4 AAC audio stream on PID `0x0102`.
- PCR carried on the video PID.
- Deterministic continuity counters and payload bytes.

## TSDuck acceptance checks

With TSDuck installed, run:

```bash
tsp -I file build/testdata/phase1.ts -P analyze -O drop

tsp -I file build/testdata/phase1.ts -P tables --pid 0x0000 -O drop

tsp -I file build/testdata/phase1.ts -P tables --pid 0x0100 -O drop
```

The acceptance target is that TSDuck recognizes a valid transport stream, finds the PAT, resolves program 100 to PMT PID `0x0100`, and reports the PMT streams/PCR PID without transport-level corruption.

## Automated checks

GitHub Actions runs:

```bash
go test ./...
```

TSDuck execution remains an environment-level acceptance test until a TSDuck installation step is added to CI.

## Not yet validated

- 500+ SPTS throughput/density.
- RTP ingest.
- CBR pacing and null-packet shaping.
- Full PSI/SI repetition scheduler.
- SimulCrypt SCS/ECMG/EMMG.
- DVB-CSA2 scrambling.
