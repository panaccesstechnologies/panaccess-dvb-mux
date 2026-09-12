# Transport foundation

Phase 2 keeps transport I/O independent from service routing, PSI/SI, SimulCrypt and scrambling.

## UDP TS

A raw UDP source expects each datagram to contain one or more complete 188-byte MPEG-TS packets. The parser therefore accepts common aggregation such as 1316 bytes (7 packets) while rejecting malformed datagram lengths and bad sync bytes.

## RTP TS

RTP v2 is parsed before MPEG-TS validation. CSRC lists, header extensions and padding are handled. The current transport layer records sequence gaps/duplicates but does not conceal packet loss; recovery belongs to a later policy layer.

## Queue and pacing

`PacketQueue` is a bounded channel used as the Phase 2 back-pressure boundary. Queue insertion is cancellation-aware so a full queue cannot prevent a clean runtime shutdown. `Pacer` provides bitrate-derived timing without coupling the transport layer to a sleep policy.

## Runtime worker

`Runtime` connects the live path as:

```text
UDP/RTP socket
    -> TS validation
    -> bounded packet queue
    -> live CC/PCR monitor
    -> UDP egress
```

The processing stage is intentionally a passthrough in Phase 2. Later phases insert service routing, PSI/SI regeneration, SimulCrypt and DVB-CSA2 between the queue and egress without changing the transport boundary.

`TSMonitor` records continuity-counter errors, PCR observations, malformed PCRs and backwards PCR movement. These counters are atomic and can be exposed directly to the future Web API/GUI.

## Egress

`UDPSink.WritePackets` aggregates TS packets into configurable UDP datagrams. Seven TS packets (1316 bytes) is supported directly and is the default-sized batch used by the Phase 2 runtime/tests.

## Operational metrics

`StreamStats` uses atomic counters so future API/GUI readers can observe packet, byte, datagram, RTP, sequence-gap and parse-error counters without taking the transport data path lock. `Runtime.Stream.Monitor.Snapshot()` supplies live CC/PCR health telemetry.
