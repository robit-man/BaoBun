# WO-001: Availability Store And Startup Scan Removal

## Status
- `state`: `pending`
- `owner`: `unassigned`
- `depends_on`: `none`
- `updates_index`: `required after every completed component`

## Objective
Replace startup non-zero byte scanning as the primary availability source with persisted availability and proof metadata, while preserving a one-time migration path for legacy data.

## In Scope
- New persisted availability store per swarm.
- Startup load path that prioritizes persisted availability and proofs.
- Legacy one-time fallback scan only when persisted availability is missing.
- Proof-authoritative availability reconciliation for incomplete files.
- Optional repair mode via `BAOBUN_REPAIR_AVAILABILITY=1`.
- Atomic persistence semantics.

## Out Of Scope
- Hierarchical tier wire protocol changes.
- Scheduler lazy refinement logic.
- UI changes.

## Files To Touch (Exact)
- New: `internal/core/availability_store.go`
- New: `internal/core/availability_store_test.go`
- Update: `internal/core/swarm.go`
- Update: `internal/core/file_io.go`
- Update: `internal/config/config.go`
- Update: `README.md`
- Update: `dev/workorders/index.md`

## Component Breakdown

### Component 1: Availability Store Format And Atomic IO
#### Implementation Directive
- Implement `AvailabilityStore` in `internal/core/availability_store.go`.
- Storage location: `<file_location>/.baobun/availability/<infohash>.json`.
- Persist fields:
  - `version`
  - `file_length`
  - `transfer_size`
  - `unit_count`
  - `have_units` (bytes, hex or base64)
  - `proven_units` (bytes, hex or base64)
  - `last_updated_unix`
- Add strict load validation:
  - reject mismatched `file_length`, `transfer_size`, `unit_count`
  - reject unsupported version
  - reject malformed bitfield lengths
- Save must be atomic:
  - write temp file
  - fsync temp file where possible
  - rename with Windows-safe replacement path

#### Success State
- Store read/write round-trips without data loss.
- Corrupt or mismatched files are rejected with actionable errors.
- Save path is atomic and resilient on Windows/macOS/Linux.

#### Index Update Requirement
- After completion, set WO-001 Component 1 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 2: Startup Load Order In Swarm Initialization
#### Implementation Directive
- Refactor `NewSwarm` in `internal/core/swarm.go`:
  - Load `ProofStore` first.
  - Load `AvailabilityStore`.
  - If availability store is valid:
    - initialize `FileIO.haveUnits` from persisted `have_units`.
    - reconcile `proven_units` against loaded proofs.
  - If missing/invalid store:
    - run legacy byte scan once.
    - for incomplete files, intersect with proven units:
      - `effectiveHave = scannedHave ∩ provenHave`
    - for complete files, allow all units.
    - immediately persist new availability store.
- Remove byte scan path from steady-state startup.

#### Success State
- Restart uses metadata load path when store exists.
- Legacy scan only executes when store is absent or explicitly repaired.
- Incomplete restart state does not advertise unprovable units.

#### Index Update Requirement
- After completion, set WO-001 Component 2 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 3: Proof-Authoritative Consistency And Persistence Hooks
#### Implementation Directive
- In `internal/core/swarm.go`:
  - On `MarkTransferUnitComplete`, persist updated availability (`have_units` + derived/provided `proven_units` state).
  - On `SaveProof`, persist proof (existing behavior) and update persisted `proven_units`.
- Add helper methods to avoid duplicated persistence logic and maintain lock order safety.
- Ensure `CanServeTransferUnit` semantics remain proof-authoritative for incomplete files.

#### Success State
- Every completed unit/proof transition is reflected on disk.
- Restarted uploader state is consistent with serving constraints.
- No stale availability/proof divergence after normal operation.

#### Index Update Requirement
- After completion, set WO-001 Component 3 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

### Component 4: Repair Mode
#### Implementation Directive
- Add `BAOBUN_REPAIR_AVAILABILITY=1` toggle in `internal/config/config.go` (or explicit helper).
- In `internal/core/swarm.go`, when enabled:
  - force full rescan
  - reconcile with proofs
  - rewrite availability store
- Keep default disabled.
- Document repair behavior in `README.md`.

#### Success State
- Repair mode can rebuild bad/missing availability state deterministically.
- Default startup remains metadata-first.

#### Index Update Requirement
- After completion, set WO-001 Component 4 status in `dev/workorders/index.md` to `completed` and add completion date + commit hash.

## Acceptance Criteria
- Partial client restart does not perform full file byte scan when availability store exists.
- Startup time scales with metadata load, not file size.
- Restarted partial uploader does not emit proof-serving errors caused by stale availability.
- Availability persistence is atomic and cross-platform safe.

## Validation Plan
- Unit tests in `internal/core/availability_store_test.go`:
  - round-trip serialization
  - corruption handling
  - schema mismatch handling
  - bitfield size mismatch handling
- Targeted regression run:
  - create partial file + proofs, restart client, ensure no full scan and correct serving gate.

## Risks
- Lock ordering around swarm/proof/availability persistence.
- Incorrect reconciliation could over-prune recoverable units.

## Rollback Plan
- Keep fallback path guarded and isolated.
- If needed, temporarily disable availability-store loading and revert to previous scan behavior behind a flag.

