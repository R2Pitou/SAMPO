# SAMPO Implementation Milestones

These are small vertical milestones, not a technology or package plan. Each produces a demonstrable capability and retains every previously passing safety test.

No production implementation begins until the architecture documentation is approved.

## Milestone 0: Documentation truth

Deliver:

- Approved decision record.
- Truthful README and corrected Spec.
- MVP architecture, acceptance tests, and parking lot.
- Manifesto SAMPO origin section.
- No selected language, framework, or database schema.

Demonstration: a competent implementation team can describe authority, custody, identity, Contract, Plan, Job, observation, and failure behavior without guessing.

## Milestone 1: Read-only filesystem catalogue

Deliver one local process with:

- Loopback browser UI shell.
- Enrollment of one filesystem provider without provider mutation.
- Read-only discovery and complete hashing.
- Search by name.
- Exact duplicate grouping as multiple Appearances of one Content item.
- Custody defaulting to user-owned.

No copying, deletion, managed custody, S3 integration, or automatic repair.

Demonstrates AT-01, AT-02, AT-04, AT-05, and the loopback portion of AT-21.

## Milestone 2: Enrollment and portable provider memory

Deliver:

- Yes / No / Ask next time prompt.
- Separate catalogue, `.sampo/`, and managed-destination permissions.
- Stable provider identity evidence.
- Provider-local ledger and home Seshat catalogue.
- Reconnect comparison and explicit uncertainty.
- Provider-ledger and home-catalogue recovery behavior.

Demonstrates AT-14 through AT-17.

## Milestone 3: One verified managed filesystem copy

Deliver:

- **Keep a copy** action.
- Explainable destination/space Plan and approval.
- Persistent Protection Contract.
- Durable idempotent Job with claim and recovery state.
- Isolated filesystem staging, complete digest verification, safe commit, and managed custody.
- Read/Open routing among available Appearances.

No replacement of user-owned destinations and no generic deletion.

Demonstrates AT-03, AT-12’s routing principle, AT-19, and filesystem AT-22.

## Milestone 4: Continuous Contract maintenance

Deliver:

- Fulfilled, unfulfilled-but-fulfillable, and unfulfillable states.
- Replacement of missing managed copies within existing authority.
- External edit transfer from managed to user custody.
- Return of previously unavailable exact Content.
- Contract amendment and bounded managed-copy retirement.
- User-owned working-copy creation for Edit.

Demonstrates AT-07 through AT-11 and AT-13.

## Milestone 5: Local S3-compatible provider

Deliver:

- A locally runnable disposable S3-compatible development dependency selected through a build-versus-integrate decision.
- Capability-based object-storage adapter using opaque keys.
- Isolated/multipart staging, complete SAMPO digest verification, idempotent recovery, and provider-native retrieval guidance.
- Access routing between local filesystem and object storage.
- Failure injection for latency, service loss, ambiguous completion, and eventual-consistency behavior where applicable.

Demonstrates AT-12, AT-20, and S3-compatible AT-22.

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

MVP completion requires all 22 acceptance tests and cross-cutting fault tests to pass.

## Milestone discipline

For every milestone:

1. Record implementation alternatives and build-versus-integrate reasoning.
2. State new authority and failure modes before implementing them.
3. Add acceptance evidence before expanding scope.
4. Preserve ordinary-file access with SAMPO absent.
5. Do not import Parking Lot behavior as an optimization.
