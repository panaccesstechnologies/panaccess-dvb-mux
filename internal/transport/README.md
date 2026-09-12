# Transport foundation

Phase 2 keeps transport I/O independent from service routing, PSI/SI, SimulCrypt and scrambling.

## UDP TS

A raw UDP source expects each datagram to contain one or more complete 188-byte MPEG-TS packets. The parser therefore accepts common aggregation such as 1316 bytes (7 packets) while rejecting malformed datagram lengths and bad sync bytes.

## RTP TS

RTP v2 is parsed before MPEG-TS validation. CSRC lists, header extensions and padding are handled. The current transport layer records sequence gaps/duplicates but does not conceal packet loss; recovery belongs to a later policy layer.

## Queue and pacing

`PacketQueue` is a bounded channel used as the Phase 2 back-pressure boundary. It deliberately keeps the public transport abstraction independent from the eventual high-density SPSC implementation. `Pacer` provides bitrate-derived timing without coupling the transport layer to a sleep policy.

## Egress

`UDPSink.WritePackets` aggregates TS packets into configurable UDP datagrams. Seven TS packets (1316 bytes) is supported directly and is the default-sized batch used by the Phase 2 tests.

## Operational metrics

`StreamStats` uses atomic counters so future API/GUI readers can observe packet, byte, datagram, RTP, CC and parse-error counters without taking the transport data path lock.
