# SAMPO Implementation Milestones

These are small vertical milestones, not a technology or package plan. Each produces a demonstrable capability and retains every previously passing safety test.

No production implementation begins until the architecture documentation is approved.

## Milestone 0: Documentation truth

**Status:** Complete — 2026-08-02

Deliver:

- Approved decision record.
- Truthful README and corrected Spec.
- MVP architecture, acceptance tests, and parking lot.
- Manifesto SAMPO origin section.
- Accepted implementation ADR set, without a database schema or implementation code.

Demonstration: a competent implementation team can describe authority, custody, identity, Contract, Plan, Job, observation, and failure behavior without guessing.

## Milestone 1: Read-only filesystem catalogue

**Status:** Complete — 2026-08-02

Deliver one local process with:

- Loopback browser UI shell.
- Enrollment of one filesystem provider without provider mutation.
- Read-only discovery and complete hashing.
- Search by name.
- Exact duplicate grouping as multiple Appearances of one Content item.
- Custody defaulting to user-owned.
- Windows per-user application lifecycle and loopback-only Gateway shell.

No copying, deletion, managed custody, S3 integration, or automatic repair.

Demonstrates AT-01, AT-02, AT-04, AT-05, the loopback portion of AT-21, and the Windows-host portion of AT-28.

## Milestone 1.1: Provider root identity safety

**Status:** Complete - 2026-08-03

Deliver:

- Platform-neutral filesystem root identity evidence with explicit confidence.
- Windows identity from an opened directory handle, preferring `FILE_ID_INFO` and retaining fallback evidence.
- Separate submitted, operational, final-path, and physical-identity evidence in Seshat.
- Transactional rejection of physical aliases and parent/child provider overlap.
- Root identity verification before and after each scan.
- Catalogue-only authority for weak remote identity.

This milestone prevents one physical filesystem occurrence from masquerading as multiple Providers. It does not implement Provider-local `.sampo` identity, automatic reconnect, clone adjudication, Failure Domain evidence, managed custody, or destructive authority; those remain in later milestones.

## Milestone 1.2: Bounded Debug Mode

**Status:** Complete - 2026-08-05

Deliver:

- Explicit Start and Stop workflow in the local browser dashboard with a visible active state.
- Incremental, crash-useful local diagnostic bundles outside the Seshat catalogue.
- Correlation of one user action through Gateway, application logic, Seshat, and Observer.
- Semantic Seshat acceptance and refusal evidence without SQL narration or persistence leakage.
- Automatic secret redaction, sanitized configuration, summaries, durations, errors, and captured panic evidence.
- No remote collector, file-content logging, per-file scan narration, or meaningful disabled-mode overhead.

This mode exists only to reconstruct a bounded manual troubleshooting session. Diagnostic recording has no authority over catalogue state and failures in the recorder do not change application behavior.

## Milestone 2: Enrollment and portable provider memory

Deliver:

- Yes / No / Ask next time prompt.
- Separate catalogue, `.sampo/`, and managed-destination permissions.
- Stable provider identity evidence.
- Explicit Failure Domain evidence independent of Provider identity.
- Provider-local ledger and home Seshat catalogue.
- Reconnect comparison and explicit uncertainty.
- Provider-ledger and home-catalogue recovery behavior.

Demonstrates AT-14 through AT-17 and the enrollment/evidence portion of AT-23.

## Milestone 3: One verified managed filesystem copy

Deliver:

- **Keep a copy** action.
- Explainable destination/space Plan and approval.
- Persistent Protection Contract.
- Independent Failure Domain planning and conspicuous same-domain alternative approval.
- Durable idempotent Job with claim and recovery state.
- Isolated filesystem staging, complete digest verification, safe commit, and managed custody.
- Readable committed layout under `library/`, SAMPO-owned staging under `.sampo/`, deterministic collision handling, and stable first-approved paths.
- Read/Open routing among available Appearances.

No replacement of user-owned destinations and no generic deletion.

Demonstrates AT-03, AT-12’s routing principle, AT-19, filesystem AT-22, AT-24, AT-26, and the filesystem side of AT-23.

## Milestone 4: Continuous Contract maintenance

Deliver:

- Fulfilled, unfulfilled-but-fulfillable, and unfulfillable states.
- Replacement of missing managed copies within existing authority.
- Unavailable-versus-positively-missing behavior and surplus reporting without automatic retirement.
- External edit transfer from managed to user custody.
- Return of previously unavailable exact Content.
- Contract amendment and bounded managed-copy retirement.
- User-owned working-copy creation for Edit.
- Durable Adopt / Leave it mine / Ask later handling for unproven files in managed space.

Demonstrates AT-07 through AT-11, AT-13, AT-25, and AT-27.

## Milestone 5: Local S3-compatible provider

Deliver:

- A locally runnable disposable S3-compatible development dependency selected through a build-versus-integrate decision.
- Capability-based object-storage adapter using opaque keys.
- Isolated/multipart staging, complete SAMPO digest verification, idempotent recovery, and provider-native retrieval guidance.
- Access routing between local filesystem and object storage.
- Failure injection for latency, service loss, ambiguous completion, and eventual-consistency behavior where applicable.
- Proof that a filesystem Provider and local S3-compatible Provider sharing backing storage remain one Failure Domain.
- Factual byte, operation, transfer-history, quota, and Provider-failure reporting without bill estimates or budget enforcement.

Demonstrates AT-12, AT-20, AT-23, AT-28, AT-29, and S3-compatible AT-22.

## Milestone 6: Projects by reviewed snapshot

Deliver:

- Many-to-many Project membership.
- Unique-Content snapshot review.
- File-level Contract creation for approved members.
- Quiet report of later additions and batch protection action.

Demonstrates AT-18.

## Milestone 7: MVP hardening

Deliver:

- Crash recovery at every Job transition.
- Duplicate-execution and lease-expiry safety.
- Catalogue integrity and safe mode.
- Provider conformance suite.
- Audit explanations and actionable failure states.
- Resource, request, session, and timeout limits.
- Security review and all acceptance tests green.

MVP completion requires all 29 acceptance tests and cross-cutting fault tests to pass.

## Milestone discipline

For every milestone:

1. Record implementation alternatives and build-versus-integrate reasoning.
2. State new authority and failure modes before implementing them.
3. Add acceptance evidence before expanding scope.
4. Preserve ordinary-file access with SAMPO absent.
5. Do not import Parking Lot behavior as an optimization.
