# WO-005: Tests, Benchmarks, Documentation, And Rollout

## Status
- `state`: `pending`
- `owner`: `unassigned`
- `depends_on`: `WO-001`, `WO-002`, `WO-003`, `WO-004`
- `updates_index`: `required after every completed component`

## Objective
Deliver complete validation, performance evidence, and operational documentation for availability persistence + hierarchical availability + lazy refinement rollout.

## In Scope
- Unit, restart, protocol compatibility, and integration tests.
- Performance benchmarks for startup and memory overhead.
- Operational docs and rollout sequencing.
- Completion gating for all prior work orders.

## Out Of Scope
- New feature development beyond validation/documentation.

## Files To Touch (Exact)
- New: `internal/core/availability_store_test.go` (expand if already created in WO-001)
- New: `internal/core/tier_availability_test.go` (expand if already created in WO-002)
- New: `internal/core/swarm_restart_test.go`
- New: `internal/core/protocol_compat_test.go`
- New: `internal/core/scheduler_refinement_test.go`
- New: `internal/core/startup_benchmark_test.go`
- Update: `README.md`
- Update: `dev/workorders/index.md`

## Component Breakdown

### Component 1: Unit Test Completion
#### Implementation Directive
- Ensure robust unit coverage for:
  - availability store load/save/validation
  - tier mapping math
  - tier promotion logic
  - refinement in-flight state transitions
- Prefer deterministic fixtures and minimal external dependencies.

#### Success State
- `go test ./...` passes with new unit tests.
- Edge cases around final-partial tier blocks are covered.

#### Index Update Requirement
- After completion, set WO-005 Component 1 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 2: Restart And Proof-Availability Regression Tests
#### Implementation Directive
- Add tests in `internal/core/swarm_restart_test.go` that simulate:
  - partial file + partial proofs persisted
  - client restart
  - serving eligibility checks
- Assert no false availability after restart and no startup full-scan when availability store exists.

#### Success State
- Restart path reproduces and prevents the historical proof-error scenario.

#### Index Update Requirement
- After completion, set WO-005 Component 2 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 3: Protocol Compatibility Tests
#### Implementation Directive
- Add tests in `internal/core/protocol_compat_test.go` for:
  - new<->new capability negotiation
  - new<->legacy fallback behavior
  - payload serialization round-trip for new message types
- Validate handlers do not crash when fields are missing in legacy payloads.

#### Success State
- Compatibility guarantees are enforced by automated tests.

#### Index Update Requirement
- After completion, set WO-005 Component 3 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 4: Integration And Performance Benchmarking
#### Implementation Directive
- Add scheduler/integration tests in `internal/core/scheduler_refinement_test.go`.
- Add benchmarks in `internal/core/startup_benchmark_test.go`:
  - metadata startup vs legacy scan startup
  - memory use proxy for peer availability state scale
- Provide reproducible benchmark command examples in `README.md`.

#### Success State
- Measurable startup reduction is demonstrated for large files.
- Refinement scheduling progresses without deadlocks under test.

#### Index Update Requirement
- After completion, set WO-005 Component 4 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 5: Rollout Documentation And Operational Guidance
#### Implementation Directive
- Update `README.md` with:
  - architecture summary of proof store + availability store + tiers
  - migration behavior for existing nodes
  - repair mode usage (`BAOBUN_REPAIR_AVAILABILITY=1`)
  - compatibility expectations across mixed-version peers
  - troubleshooting flow for proof/availability mismatch
- Align section names with existing README structure.

#### Success State
- Operators and contributors can deploy and debug the feature set without code spelunking.

#### Index Update Requirement
- After completion, set WO-005 Component 5 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

## Acceptance Criteria
- Complete automated test suite covers functional and compatibility goals.
- Benchmarks demonstrate startup-time improvement over legacy scan behavior.
- Documentation is sufficient for operation and development handoff.

## Risks
- Flaky integration tests around async/session timing.
- Benchmark noise without stable fixture sizing.

## Rollback Plan
- If regressions occur, gate hierarchical/refinement paths with feature flags and keep persistence protections from WO-001 active.

