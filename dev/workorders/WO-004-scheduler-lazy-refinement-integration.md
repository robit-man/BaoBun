# WO-004: Scheduler Lazy Refinement Integration

## Status
- `state`: `pending`
- `owner`: `unassigned`
- `depends_on`: `WO-001`, `WO-002`, `WO-003`
- `updates_index`: `required after every completed component`

## Objective
Refactor scheduling so downloads are selected coarse-first and refined on demand, instead of globally scanning random T1 units, while preserving throughput and compatibility behavior.

## In Scope
- Transfer scheduling integration with tier summaries and refinement responses.
- Per-peer refinement in-flight tracking with timeout/retry.
- Candidate generation from refined windows.
- Compatibility path for legacy peers (direct T1 behavior).

## Out Of Scope
- Long-form benchmarks/docs and release notes (WO-005).

## Files To Touch (Exact)
- Update: `internal/core/transferunit_manager.go`
- Update: `internal/core/transferunit.go`
- Update: `internal/core/p2p_handler.go`
- Update: `internal/core/swarm.go`
- Update: `internal/core/tier_availability.go`
- New: `internal/core/refinement_state.go`
- New: `internal/core/refinement_state_test.go`
- Update: `README.md`
- Update: `dev/workorders/index.md`

## Component Breakdown

### Component 1: Refinement State Model
#### Implementation Directive
- Add `internal/core/refinement_state.go` with structures for:
  - per-peer coarse block candidates
  - in-flight refine request map keyed by `(peer, fromTier, toTier, index)`
  - timestamps and retry counts
  - timeout constants and backoff policy hooks
- Ensure thread-safe access from scheduler and message handlers.

#### Success State
- Refinement in-flight state is deterministic and concurrency safe.

#### Index Update Requirement
- After completion, set WO-004 Component 1 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 2: Scheduler Candidate Flow Refactor
#### Implementation Directive
- Refactor `internal/core/transferunit_manager.go`:
  - stage 1: choose coarse block (prefer T3, then T2) from peer summaries.
  - stage 2: trigger refinement request for chosen block if T1 detail unavailable.
  - stage 3: once refined response arrives, schedule T1 requests from that window.
- Keep existing per-peer/max-active request caps.
- Preserve legacy peer behavior by bypassing refinement and using current T1 path.

#### Success State
- Scheduler no longer depends on full global random T1 scan for capable peers.
- Throughput remains stable and request windows stay bounded.

#### Index Update Requirement
- After completion, set WO-004 Component 2 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 3: Message Handling For Refinement
#### Implementation Directive
- In `internal/core/p2p_handler.go`:
  - handle refine request messages and emit refine responses.
  - apply received refine responses to peer tier availability cache.
  - clear in-flight entries and wake scheduler when refinement data arrives.
- In `internal/core/swarm.go` and `internal/core/tier_availability.go`:
  - expose APIs needed to fetch refined windows and apply peer tier deltas.

#### Success State
- Refinement requests/responses produce actionable T1 scheduling windows.
- Timeout/retry handles missing refine responses without deadlock.

#### Index Update Requirement
- After completion, set WO-004 Component 3 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 4: State Persistence And Recovery Hooks
#### Implementation Directive
- Ensure local tier state used by scheduler persists via availability store.
- Define restart behavior for peer-refinement cache:
  - local tier state persisted
  - peer transient refinement state dropped on restart
- Document this behavior in `README.md`.

#### Success State
- Restart does not lose local availability tiers.
- Scheduler can resume from local tier state without requiring full recomputation.

#### Index Update Requirement
- After completion, set WO-004 Component 4 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

## Acceptance Criteria
- For capable peers, download flow is coarse selection -> refinement -> T1 requests.
- For legacy peers, behavior remains equivalent to current T1 scheduling.
- Refinement timeouts and retries do not stall swarm progress.

## Validation Plan
- Unit tests in `internal/core/refinement_state_test.go`:
  - in-flight lifecycle
  - timeout and retry behavior
  - concurrency safety for insert/apply/clear
- End-to-end manual smoke:
  - multi-peer run confirms refinement message exchange before dense T1 requests.

## Risks
- Scheduler complexity may introduce starvation if coarse block choice is biased.
- Races between response handling and request timeout cleanup.

## Rollback Plan
- Feature-flag refined scheduling path; fallback to legacy T1 scheduler while retaining message handling code.

