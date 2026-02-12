# WO-003: Wire Protocol, Handshake Capability Negotiation, And Compatibility

## Status
- `state`: `pending`
- `owner`: `unassigned`
- `depends_on`: `WO-001`, `WO-002`
- `updates_index`: `required after every completed component`

## Objective
Add hierarchical availability messages and capability negotiation to the peer protocol while preserving compatibility with legacy peers that only speak flat T1 bitfield/HAVE behavior.

## In Scope
- Protocol message additions for summaries and refinement requests/responses.
- Handshake capability negotiation fields.
- Serializer and message model updates.
- Session and peer handler behavior updates for downgrade compatibility.

## Out Of Scope
- Scheduler refinement strategy and candidate selection policy.
- Performance benchmarking and release docs (WO-005).

## Files To Touch (Exact)
- Update: `pkg/protocol/proto/peer_protocol.proto`
- Update: `pkg/protocol/proto/peer_protocol.pb.go` (generated)
- Update: `pkg/protocol/proto/peer_protocol_vtproto.pb.go` (generated)
- Update: `pkg/protocol/p2p.go`
- Update: `internal/core/serializer.go`
- Update: `internal/core/protobuf.serializer.go`
- Update: `internal/core/json_serializer.go`
- Update: `internal/core/session_manager.go`
- Update: `internal/core/p2p_handler.go`
- Update: `internal/core/swarm.go`
- Update: `README.md`
- Update: `dev/workorders/index.md`

## Component Breakdown

### Component 1: Protocol Schema Additions
#### Implementation Directive
- In `pkg/protocol/proto/peer_protocol.proto`, add:
  - `MSG_AVAILABILITY_SUMMARY`
  - `MSG_AVAILABILITY_REFINE_REQUEST`
  - `MSG_AVAILABILITY_REFINE_RESPONSE`
- Add payload messages:
  - `AvailabilitySummaryPayload { uint32 tier; uint64 start_index; bytes bits; }`
  - `AvailabilityRefineRequestPayload { uint32 from_tier; uint32 to_tier; uint64 index; }`
  - `AvailabilityRefineResponsePayload { uint32 tier; uint64 start_index; bytes bits; }`
- Extend `HandshakePayload` with:
  - `uint32 protocol_version`
  - `uint32 max_tier_supported`
  - `bool supports_hierarchical_availability`
- Regenerate protobuf artifacts:
  - `peer_protocol.pb.go`
  - `peer_protocol_vtproto.pb.go`

#### Success State
- New schema compiles and generated files are in sync.
- Existing payloads remain backward compatible.

#### Index Update Requirement
- After completion, set WO-003 Component 1 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 2: Protocol Model And Serializer Integration
#### Implementation Directive
- Update `pkg/protocol/p2p.go` to include:
  - new message types
  - new payload structs
  - extended handshake struct fields
- Extend serializer interface in `internal/core/serializer.go`:
  - marshal/unmarshal methods for each new payload.
- Implement protobuf serializer support in `internal/core/protobuf.serializer.go`.
- Implement JSON serializer parity in `internal/core/json_serializer.go` for test/dev completeness.

#### Success State
- All serializers can encode/decode new message types and extended handshake fields.
- Build passes without interface mismatch.

#### Index Update Requirement
- After completion, set WO-003 Component 2 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 3: Session Handshake Capability Negotiation
#### Implementation Directive
- In `internal/core/session_manager.go` and `internal/core/p2p_handler.go`:
  - send extended handshake capabilities.
  - record peer capabilities on handshake receipt.
  - if peer does not support hierarchical availability, mark as legacy mode.
- Preserve current connect/handshake state transitions.
- Avoid breaking peers compiled against the old protocol.

#### Success State
- New peers negotiate hierarchical availability successfully.
- Legacy peers still connect and exchange data via existing T1 flow.

#### Index Update Requirement
- After completion, set WO-003 Component 3 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 4: Coarse-First Availability Exchange (Transport Layer)
#### Implementation Directive
- In `internal/core/p2p_handler.go`:
  - on connection with capable peer:
    - send T3 summary first
    - defer T2/T1 details unless requested.
  - for legacy peers:
    - keep current bitfield/HAVE path unchanged.
- In `internal/core/swarm.go`, expose helper methods used by handler:
  - summary extraction by tier windows
  - refinement response generation for requested tier/index.

#### Success State
- Capable peers exchange coarse summaries first.
- Legacy peers observe no behavior changes from their perspective.

#### Index Update Requirement
- After completion, set WO-003 Component 4 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

## Acceptance Criteria
- New<->new peer handshake negotiates hierarchical capability and can exchange tier summaries.
- New<->old peer path downgrades to T1 bitfield/HAVE with no connection failures.
- Wire changes are contained to protocol + serializers + handler/session integration points.

## Validation Plan
- Add targeted protocol tests (WO-005) but verify compile/runtime smoke now:
  - start two new peers and confirm summary messages observed in logs.
  - connect new peer to legacy build and confirm fallback behavior.

## Risks
- Protocol drift if generated files are stale.
- Silent fallback bugs if capability flags are not persisted in handler state.

## Rollback Plan
- Guard hierarchical behavior behind negotiated capability; if issues occur, force legacy path while retaining protocol fields.

