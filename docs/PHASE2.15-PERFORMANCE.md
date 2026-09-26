# Phase 2.15 — High-Density Transport Performance Baseline

## Objective

Establish a reproducible performance baseline for the Phase 2 transport path before optimizing the implementation toward the planned 500+ SPTS target.

## Scope

Measure the current implementation at increasing stream counts:

- 1 SPTS
- 10 SPTS
- 50 SPTS
- 100 SPTS

Each run should record:

- Aggregate input bitrate
- Aggregate output bitrate
- Packets received and transmitted
- Packet errors
- Continuity-counter errors
- PCR observations/errors where applicable
- Queue capacity and observed queue/back-pressure behavior
- Process CPU usage
- Process RSS/memory
- Runtime duration and clean/unclean shutdown

## Acceptance discipline

The baseline is a measurement step, not a performance claim. Do not mark a stream-count level VERIFIED until it has been executed on the target Linux host and the results are recorded.

Phase 2.14 remains the functional transport acceptance gate. Phase 2.15 adds scalability measurements on top of that known-good path.

## Implementation note

The current runtime uses a bounded Go channel as the back-pressure boundary and allocates a batch slice in the run loop. These are explicit optimization candidates after the baseline is measured. The original project specification calls for Rust or C++20; the current Go implementation remains a separate compliance gap.

## Planned execution order

1. Add benchmark instrumentation and a reproducible benchmark command.
2. Build and test on the development host.
3. Run 1/10/50/100 SPTS measurements on the target Linux host.
4. Record raw results and identify the first bottleneck.
5. Optimize only after the baseline is captured.
6. Repeat the same measurements to compare before/after results.
