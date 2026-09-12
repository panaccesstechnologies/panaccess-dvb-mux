# Reference Documents

These references are used as engineering inputs for the project. The original PDFs supplied in the ChatGPT conversation remain the working copies; this repository records the references and their intended use rather than redistributing the full documents.

## 1. TSDuck User Guide

User-supplied reference:

https://tsduck.io/docs/tsduck.html

Use for:

- MPEG-TS inspection and diagnostics
- PSI/SI table inspection
- stream analysis
- packet/table validation
- interoperability checks during development

## 2. ETSI EN 300 468 V1.18.1 (2023-12)

**Digital Video Broadcasting (DVB); Specification for Service Information (SI) in DVB systems**

Supplied PDF: `en_300468v011801p.pdf`

Primary use in this project:

- SI table definitions
- table-section mapping
- repetition-rate requirements
- descriptor coding and placement
- service information required for receiver configuration
- service/event signalling
- scrambling descriptor and CA-related signalling where applicable

The document identifies ISO/IEC 13818-1 as a normative reference for MPEG-2 Systems and defines DVB SI that complements PSI.

## 3. ETSI TS 102 470-2 V1.2.1 (2011-09)

**Digital Video Broadcasting (DVB); IP Datacast: Program Specific Information (PSI)/Service Information (SI); Part 2: IP Datacast over DVB-SH**

Supplied PDF: `ts_10247002v010201p.pdf`

Primary use in this project:

- DVB-SH PSI/SI usage where relevant
- PAT/PMT/CAT/TSDT requirements
- NIT/SDT usage
- DVB-SH-specific descriptors and signalling
- IP Datacast service signalling

## 4. Project implementation specification

The project implementation specification supplied during the earlier design phase remains the functional source for the target product, including:

- 500+ concurrent SPTS target
- multi-NIC IP ingest/egress
- DVB-SimulCrypt SCS/ECMG/EMMG integration
- ECM/EMM handling
- DVB-CSA2 scrambling
- synchronized CW parity rotation
- PSI/SI regeneration
- real-time Web GUI
- TSDuck-based verification and acceptance testing

## Reference precedence

For implementation decisions, preserve the distinction between:

1. **Normative standards** — ETSI/ISO/DVB requirements.
2. **Project specification** — product-specific functional and performance requirements.
3. **TSDuck guidance/tooling** — diagnostics, analysis and interoperability validation.
4. **Implementation choices** — internal architecture selected to satisfy the above.

When a project requirement and a standard appear to conflict, do not silently choose one. Record the issue and resolve it explicitly before coding the affected subsystem.
