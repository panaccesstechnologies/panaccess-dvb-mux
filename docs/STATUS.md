# Project Status

**Project:** panaccess-dvb-mux  
**Status:** Phase 5.5 — EMMG transaction → EMM PID end-to-end integration — VERIFIED ON inst05  
**Updated:** 2026-10-01

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

## Phase 3 progress

### Phase 3.1 — PSI/CA service discovery model — VERIFIED ON inst05

The initial 5-stream development profile has been selected because `inst05` is a small test host. The user confirmed that all five streams use the same CA signalling pattern observed in the analyzer.

Observed test profile for the five streams:

- Service IDs are service-specific; Stream 1 was confirmed as service `101`.
- Stream 1 PMT PID: `0x0100`.
- Stream 1 video PID: `0x0201`.
- Stream 1 audio/PCR PID: `0x0200`.
- PMT CA descriptor: CA System ID `0x4AFC`, CA PID `0x1FFE`.
- CAT CA descriptor: CA System ID `0x4AFC`, CA PID `0x1FFE`.
- The same CA signalling pattern was confirmed by the user for Streams 2–5.

Implementation added:

- `internal/mpegts/ca.go`
  - MPEG-2/DVB CA descriptor (tag `0x09`) parser.
  - CA System ID, CA PID and private-data preservation.
  - CAT section marshal/parse with CRC validation.
- `internal/mpegts/ca_test.go`
  - CA descriptor parsing tests.
  - Unknown-descriptor handling test.
  - CAT round-trip test.
- `internal/service/service.go`
  - Normalized service model containing service ID, PMT/PCR, elementary streams, ECM endpoints and EMM endpoints.
  - Discovery from PAT program + PMT + CAT.
- `internal/service/service_test.go`
  - Service 101 CA discovery test using CA System ID `0x4AFC` and CA PID `0x1FFE`.

**Important:** this phase only discovers and models existing CA signalling. It does **not** yet generate ECMs, generate EMMs, implement ECMG/EMMG, or scramble payloads.

The implementation is based on the MPEG-2/DVB CA descriptor structure and the project's EN 300 468 PSI/SI reference set. ETSI EN 300 468 defines CA-related signalling within DVB PSI/SI; the SimulCrypt ECMG/EMMG protocol itself is a separate specification and remains a later phase.

## TSDuck

Phase 1 fixture remains the bitstream acceptance gate and is verified on `inst05`. The Phase 2.14 live multicast capture has also been validated directly with TSDuck's `pcap` input plugin and `analyze` processor.

## Current phase

### Phase 4.1 — DVB-SimulCrypt ECMG protocol foundation — VERIFIED ON inst05

Phase 3.1 was verified on `inst05` after pulling commit `a7403e8`:

```text
go test ./...       PASS
go test -race ./... PASS
go build ./...      PASS
```

Phase 4.1 has now added the initial ECMG ⇔ SCS TLV codec in `internal/simulcrypt/ecmg.go` with protocol version `0x03`, ECMG message-type constants, parameter-type constants, big-endian parameter helpers, and strict message/parameter length validation. Deterministic wire-format tests are in `internal/simulcrypt/ecmg_test.go`.

The codec is deliberately transport-neutral at this step: it does not open TCP connections, negotiate a channel, generate ECMs, or expose CWs.

Validation on `inst05` after pulling commit `d9499e2`:

```text
go test ./...       PASS
go test -race ./... PASS
go build ./...      PASS
```

The ECMG codec is therefore **verified as a wire-format component**. This does not yet prove interoperability with a real ECMG server and does not claim ECM generation, CW exchange, TCP session management, or scrambling.

ETSI TS 103 197 specifies ECMG ⇔ SCS as a TCP-based connection-oriented interface using protocol version `0x03`. Channel setup precedes stream setup, and CW_provision produces an ECM_response. The next implementation step is the TCP session/client layer, followed by a controlled interoperability test against an ECMG endpoint.

**Specification compliance note:** the current implementation remains Go-based; the broader project requirement previously identified Rust or C++20 as the implementation target. This remains a separate compliance gap and should not be conflated with successful functional validation.

### Phase 4.2 — DVB-SimulCrypt ECMG TCP client/session — VERIFIED ON inst05

Implemented:

- TCP ECMG client with configurable dial/I/O timeouts.
- One-channel-per-TCP-connection lifecycle.
- `channel_setup` with `ECM_channel_id` and `Super_CAS_id`.
- `channel_status` / `channel_error` response handling.
- `stream_setup` with `ECM_channel_id`, `ECM_stream_id`, `ECM_id` and `nominal_CP_duration`.
- `stream_status` / `stream_error` response handling.
- Channel and stream test requests.
- Stream close request/response handling.
- Strict framed TCP reads using the existing ECMG codec.
- Deterministic localhost mock-ECMG lifecycle test covering channel setup, stream setup, channel test, stream test and stream close.

GitHub Actions Go Test run #58 for commit `57cea10f3b467aa6e69c01a48a805ae9149fcbb4` completed successfully.

This phase intentionally does **not** generate ECMs or CWs and does not connect to a production ECMG endpoint yet. ETSI TS 103 197 defines the SCS as the TCP client and requires channel establishment before stream establishment; this implementation follows that lifecycle. citeturn1search12turn1search13

**Next verification on inst05:**

```bash
cd ~/panaccess-dvb-mux
git pull --ff-only
go test ./...
go test -race ./...
go build ./...
```

After those pass, Phase 4.2 will be marked verified on `inst05`. The next development step will be controlled ECMG `CW_provision` / `ECM_response` handling.

### Phase 4.3 — ECMG CW_provision / ECM_response — VERIFIED ON inst05

Implemented:

- `CW_provision` request construction.
- `ECM_channel_id`, `ECM_stream_id`, and `CP_number` signalling.
- Repeated `CP_CW_combination` encoding as Crypto-period number + control word.
- Optional `CP_duration` and `access_criteria`.
- `ECM_response` parsing and identity validation.
- ECM datagram extraction.
- Local mock-ECMG integration coverage for a complete CW provision → ECM response transaction.
- 8-byte control-word test vector for the planned DVB-CSA2 integration.

ETSI TS 103 197 defines `CW_provision` as the SCS request to compute an ECM, with `CP_CW_combination` carrying the crypto-period number and control word; `ECM_response` returns the ECM datagram. The specification also notes that `CP_CW_combination` is typically 10 bytes: 2 bytes CP number plus the control word. citeturn1search1turn0search12

This phase still does **not** generate an ECM locally and does **not** perform DVB-CSA2 scrambling. The mock server only returns deterministic test ECM bytes.

**Verification on inst05:**

Commit `3be5ca392e07f0f5194da7c492022b4bb916d9e0` was pulled and verified with `go test ./...`, `go test -race ./...`, and `go build ./...`; all passed.

**Phase 4.3 status: 🟢 VERIFIED ON inst05**

The next development step is ECM datagram integration into the service's ECM PID signalling path. This still precedes DVB-CSA2 scrambling.

```bash
cd ~/panaccess-dvb-mux
git pull --ff-only
go test ./...
go test -race ./...
go build ./...
```

After these pass, Phase 4.3 will be marked verified and the next step will connect the returned ECM datagram to the service's ECM PID signalling path.

### Phase 4.4 — DVB-SimulCrypt EMMG/MUX TCP foundation — VERIFIED ON inst05

Implemented against ETSI TS 103 197:

- EMMG/MUX protocol message types for channel, stream, bandwidth and Data_provision.
- EMMG parameter types: client_id, section_TSpkt_flag, data_channel_id, data_stream_id, datagram, bandwidth, data_type, and data_id.
- TCP channel setup/status lifecycle.
- TCP stream setup/status lifecycle.
- EMM Data_provision generation with one or more datagrams.
- Stream close request/response.
- Deterministic localhost integration test covering channel setup → stream setup → EMM data provision → stream close.
- EMM test profile uses CA-derived client_id 0x4AFC0001, data channel 1, data stream 1, data ID 100, and section-format EMM data.

The implementation intentionally starts with the TCP-based control/data interface. ETSI TS 103 197 specifies the EMMG/PDG as the TCP client and MUX as the server, with Data_provision carrying EMM/private data; the UDP variant uses the same message format but sends only Data_provision over UDP.

Important: Data_provision has no response message. The client therefore writes it without waiting for a reply.

This phase does not yet insert received EMM datagrams into the MPEG-TS and does not yet perform DVB-CSA2 scrambling.

Verification on inst05:

    cd ~/panaccess-dvb-mux
    git pull --ff-only
    go test ./...
    go test -race ./...
    go build ./...

After these pass, Phase 4.4 will be marked verified and the next step will be the first end-to-end ECM/EMM integration into the 5-stream MPEG-TS pipeline.

### Phase 5.2 — ECMG response → service ECM PID integration — VERIFIED ON inst05

Implemented the bridge from a validated ECMG `ECM_response` to the service's discovered ECM PID:

- `internal/service/ecm.go` adds `ECMInserter`.
- Selects the ECM endpoint by CA System ID.
- Uses the discovered CA PID as the ECM PID.
- Packetizes the returned `ECMDatagram` with the MPEG-TS section packetizer.
- Maintains an independent ECM PID continuity counter.
- Rejects invalid/empty ECM responses.
- Integration test covers service 101, CA System ID `0x4AFC`, ECM PID `0x1FFE`, PUSI, pointer field and payload preservation.

Verification on `inst05`:

```text
go test ./...       PASS
go test -race ./... PASS
go build ./...      PASS
```

Phase 5.2 is verified. This still does not scramble payload packets.

### Phase 5.3 — ECMG transaction → ECM PID end-to-end integration — VERIFIED ON inst05

Added an end-to-end integration test covering the existing ECMG client and ECM PID insertion path:

- TCP ECMG channel setup.
- ECMG stream setup.
- CW provision transaction.
- ECMG ECM response.
- Conversion to `ECMResponse`.
- Service ECM endpoint selection.
- MPEG-TS ECM section packetization onto PID `0x1FFE`.
- Verification of PUSI, pointer field, continuity counter, PID and returned ECM payload.

Verification on `inst05`:

```text
go test ./...       PASS
go test -race ./... PASS
go build ./...      PASS
```

Phase 5.3 is verified. This is still a protocol/integration test using a mock ECMG; production ECMG interoperability and actual DVB-CSA2 scrambling remain to be implemented and validated.

### Phase 5.4 — EMM PID insertion foundation — VERIFIED ON inst05

Implemented CAT-derived EMM PID packetization/insertion:

- `internal/mpegts/emm.go` adds an independent EMM PID injector.
- `internal/service/emm.go` selects the EMM endpoint by CA System ID.
- EMM datagrams are packetized as MPEG-2 sections onto the discovered EMM PID.
- Independent continuity counter handling is included.
- Tests verify PID, PUSI, pointer field, continuity counter and payload preservation.

Verification on `inst05`:

```text
go test ./...       PASS
go test -race ./... PASS
go build ./...      PASS
```

Phase 5.4 is verified. EMMG-to-EMM insertion end-to-end validation is the next step.

### Phase 5.5 — EMMG transaction → EMM PID end-to-end integration — VERIFIED ON inst05

Added end-to-end coverage for the existing EMMG TCP client and EMM insertion path:

- EMMG channel setup.
- EMMG stream setup.
- EMMG `data_provision`.
- EMM datagram preservation.
- CAT-derived EMM PID selection.
- MPEG-TS EMM packetization with PUSI, pointer field and continuity counter verification.

Verification on `inst05`:

```text
go test ./...       PASS
go test -race ./... PASS
go build ./...      PASS
```

Phase 5.5 is verified. The next stage is DVB-CSA2 cryptographic implementation and known-answer testing.

### Phase 6.1 — DVB-CSA2 pure-Go cipher foundation — IMPLEMENTATION CORRECTED; VERIFICATION PENDING

Added `internal/csa2` with:

- 64-bit control-word handling.
- CSA2 key expansion.
- 56-round block cipher.
- CSA2 stream cipher state and keystream generation.
- Payload encryption/decryption primitives.
- Known-answer test derived from the public `libdvbcsa` test suite.
- Short-payload validation.

The implementation is intentionally payload-only at this stage; MPEG-TS header/adaptation-field handling and scrambling-control signalling are not yet part of this phase.

The encryption/decryption block chaining was corrected to match the public `libdvbcsa` reference semantics: final 8-byte block first, reverse block processing for encryption, stream stage after the first block, and the corresponding forward inverse for decryption. The stream-cipher state machine was then replaced with a direct Go translation of the classical `libdvbcsa` stream implementation, including its S-boxes, CDEF table, initialization rounds and output extraction. citeturn0search0turn0search1

External validation basis: the public libdvbcsa project documents CSA as a block cipher plus stream cipher using the same 64-bit control word, and its test suite provides deterministic encryption vectors. citeturn0search0turn0search8

**Latest correction:** the `refStreamSBox`/`refStreamCDEF` table translation was malformed and caused the inst05 compile errors at `stream_ref.go:18-27`. It has now been regenerated directly from the pinned libdvbcsa reference, preserving the 7×32 S-box and 1024-entry CDEF table. **Verification remains pending `inst05`. The subsequent compile failure was traced to the CDEF table: unlike the 7×32 S-box, libdvbcsa defines CDEF as a flat 1024-entry array. The Go literal has now been corrected to a flat `[1024]uint16` initializer. The trailing comma was initially placed on a separate line, which is invalid for a multiline Go composite literal. It has now been placed directly after the final element; verification remains pending `inst05`.**

A Go array-to-slice compile issue in the CSA2 block chaining code was corrected after inst05 verification exposed it. The affected `copy()` calls now use `b[:]`.

## Planned phases

- Phase 3.1 — PSI/CA service discovery model
- Phase 4.1 — DVB-SimulCrypt ECMG protocol foundation
- Phase 4.2 — DVB-SimulCrypt ECMG TCP session/client and channel/stream lifecycle
- Phase 4.4 — DVB-SimulCrypt EMMG/MUX protocol foundation
- Phase 5 — ECM/EMM integration into the 5-stream service pipeline
- Phase 6 — DVB-CSA2 scrambling engine integration
- Phase 7 — 500+ service concurrency, NUMA/threading and performance tuning
- Phase 8 — Web API and real-time Web GUI
- Phase 9 — Docker deployment, observability and operational tooling
- Phase 10 — interoperability, fault-injection and acceptance testing


### CSA2 KAT isolation update
- `inst05` reached the CSA2 known-answer test after the stream reference compiled.
- KAT failed with ciphertext beginning `063442...` versus the upstream `libdvbcsa` vector beginning `2d0a47...`.
- The first 8 bytes are produced by the block stage before stream XOR, so the failure was isolated to the block key schedule rather than the stream cipher.
- Replaced the previous legacy `keyPerm` translation with an exact 64-bit basis-mask representation of upstream `libdvbcsa` `kperm[8][256]` semantics.
- Commit: `5f87864076d45d8bcf2fc35bb1de4bddd8d7d6a4`.
- Phase 6.1 remains **PENDING** until `inst05` KAT, full tests, race tests, and build pass.

- Follow-up comparison against all 2048 upstream `kperm` entries found 10 incorrect basis masks in the previous translation; these have now been regenerated directly from `kperm[row][1<<bit]`.
- Latest CSA2 key-schedule commit: `f32c9ec2d3421dcb6ce835be448f599ffe880200`.
- inst05 still failed the KAT after the exact `kperm` regeneration, so Phase 6.1 remains **PENDING**.
- Source-level comparison against pinned libdvbcsa identified an additional integration issue: the stream cipher consumes the nibble-swapped control word (`cws`), not the raw CW. The Go stream path now applies that transformation before keystream generation.
- The copied libdvbcsa test vector is a decrypt-direction acceptance vector: upstream `testdec.c` feeds the `in` plaintext vector to `dvbcsa_decrypt()` and expects the `out` ciphertext vector; `testenc.c` performs the inverse direction. The Go KAT has been corrected to validate `Decrypt(plaintext) == ciphertext` and then the reverse `Encrypt(ciphertext) == plaintext`.
- Correction commits: `31fe9f909e08937cd7790731cab6d39bf79952d5` and `fd46ad12790a4491c2c8ec598dfaa70cfa142218`.
- inst05 still failed the corrected KAT; the failure is now isolated to the Go block-decrypt implementation. Pinned libdvbcsa uses `S = sbox[key[i] ^ W[6]]` and the inverse round writes `W[7]=W[6]`, `W[6]=W[5]^perm[S]`, then derives `L=W[7]^S`. The Go implementation had an index-shifted inverse round. This has been corrected in `6952981fc8666e4de4a5d956cf5ef55f971c86b4`.
- inst05 still failed the KAT after the block-decrypt round correction. Comparison with pinned `libdvbcsa` `dvbcsa_decrypt()` identified the remaining integration error: each decrypted block must be XORed with the previous **ciphertext** block, not the previous plaintext block. The Go implementation has been corrected in `917eddbf1ae5d44d6816d721b303654aa66ac242`.
- Verified the pinned upstream `kperm[8][256]` table programmatically: 62/64 basis masks matched, with the final two masks incorrect in the Go port. Corrected row 7 input bits 6/7 to `0x40400000220` and `0x200040400000220` in `917eddbf1ae5d44d6816d721b303654aa66ac242` follow-up commit.
- inst05 KAT still failed after the key-mask correction. Upstream `dvbcsa_decrypt()` review showed the block stage ordering was still wrong: each block must be XORed with the previous decrypted block **before** calling `block_decrypt()`. The Go implementation has been corrected to match that sequence.
- Phase 6.1 remains **PENDING** until the corrected KAT, full tests, race tests and build pass on `inst05`.

- CSA2 KAT investigation: upstream `testdec.c` defines the test vector direction as `decrypt(ascending plaintext) -> 2d0a...` and `testenc.c` verifies the inverse. The local KAT direction was reversed. Upstream key-permutation basis extraction also confirms the original row-7 masks (`0x200000000000000`, `0x1000000`); the experimental replacement was reverted. Phase 6.1 remains pending inst05 verification.
