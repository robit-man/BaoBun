# Work Orders Index

## Overview
This index tracks implementation progress for the hierarchical availability and proof-persistence initiative.

## Rules
- Update this file immediately after completing each component in any work order.
- For each completed component, fill:
  - `status` (`completed`)
  - `completed_on` (YYYY-MM-DD)
  - `commit` (short SHA)
  - `notes` (optional, concise)
- Do not mark a work order `completed` until all components are `completed`.

## Work Order Links
- `WO-001`: `dev/workorders/WO-001-availability-store-and-startup-scan-removal.md`
- `WO-002`: `dev/workorders/WO-002-local-tier-availability-and-promotion.md`
- `WO-003`: `dev/workorders/WO-003-wire-protocol-hierarchical-availability-and-compat.md`
- `WO-004`: `dev/workorders/WO-004-scheduler-lazy-refinement-integration.md`
- `WO-005`: `dev/workorders/WO-005-tests-benchmarks-docs-and-rollout.md`

## Work Order Status Table

| Work Order | Title | Depends On | Overall Status | Owner | Last Updated |
|---|---|---|---|---|---|
| WO-001 | Availability Store And Startup Scan Removal | none | pending | unassigned | pending |
| WO-002 | Local Hierarchical Availability And Promotion | WO-001 | pending | unassigned | pending |
| WO-003 | Wire Protocol, Handshake Capability Negotiation, And Compatibility | WO-001, WO-002 | pending | unassigned | pending |
| WO-004 | Scheduler Lazy Refinement Integration | WO-001, WO-002, WO-003 | pending | unassigned | pending |
| WO-005 | Tests, Benchmarks, Documentation, And Rollout | WO-001, WO-002, WO-003, WO-004 | pending | unassigned | pending |

## Component Tracker

### WO-001 Components
| Component | Description | Status | Completed On | Commit | Notes |
|---|---|---|---|---|---|
| 1 | Availability Store Format And Atomic IO | pending | - | - | - |
| 2 | Startup Load Order In Swarm Initialization | pending | - | - | - |
| 3 | Proof-Authoritative Consistency And Persistence Hooks | pending | - | - | - |
| 4 | Repair Mode | pending | - | - | - |

### WO-002 Components
| Component | Description | Status | Completed On | Commit | Notes |
|---|---|---|---|---|---|
| 1 | Tier Definitions And Mapping Utilities | pending | - | - | - |
| 2 | TierAvailability Manager | pending | - | - | - |
| 3 | Persist Tier Bitfields | pending | - | - | - |
| 4 | Swarm Integration | pending | - | - | - |

### WO-003 Components
| Component | Description | Status | Completed On | Commit | Notes |
|---|---|---|---|---|---|
| 1 | Protocol Schema Additions | pending | - | - | - |
| 2 | Protocol Model And Serializer Integration | pending | - | - | - |
| 3 | Session Handshake Capability Negotiation | pending | - | - | - |
| 4 | Coarse-First Availability Exchange (Transport Layer) | pending | - | - | - |

### WO-004 Components
| Component | Description | Status | Completed On | Commit | Notes |
|---|---|---|---|---|---|
| 1 | Refinement State Model | pending | - | - | - |
| 2 | Scheduler Candidate Flow Refactor | pending | - | - | - |
| 3 | Message Handling For Refinement | pending | - | - | - |
| 4 | State Persistence And Recovery Hooks | pending | - | - | - |

### WO-005 Components
| Component | Description | Status | Completed On | Commit | Notes |
|---|---|---|---|---|---|
| 1 | Unit Test Completion | pending | - | - | - |
| 2 | Restart And Proof-Availability Regression Tests | pending | - | - | - |
| 3 | Protocol Compatibility Tests | pending | - | - | - |
| 4 | Integration And Performance Benchmarking | pending | - | - | - |
| 5 | Rollout Documentation And Operational Guidance | pending | - | - | - |

## Completion Definition
- WO-001 complete when components 1-4 are complete.
- WO-002 complete when components 1-4 are complete.
- WO-003 complete when components 1-4 are complete.
- WO-004 complete when components 1-4 are complete.
- WO-005 complete when components 1-5 are complete.

