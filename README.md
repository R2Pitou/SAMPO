# SAMPO

**Storage Abstraction Management & Policy Orchestrator**

SAMPO is a storage orchestration product with a Digital Librarian control plane. It builds a searchable logical library over ordinary data, recognizes exact content across storage providers, and maintains protection obligations approved by the user.

SAMPO does not require users to reorganize their existing digital life. It treats provider placement as an implementation detail while preserving ordinary provider-native access.

> SAMPO came first. The expansion came later. See the [Manifesto](MANIFESTO.md) for the historically accurate explanation.

## Constitutional laws

The [Manifesto](MANIFESTO.md) governs the project:

1. SAMPO must never make user data less accessible than it was before SAMPO.
2. The user expresses intent; the Librarian determines implementation.
3. Existing tools should be orchestrated before replacements are invented.
4. Human attention is more valuable than machine time.
5. Storage providers are implementation details, not identities.

Removing SAMPO must leave committed files readable through ordinary provider-native tools. Neither the home catalogue nor provider-local metadata may be required to decode user bytes.

## Current status

SAMPO is rebuilding from the deliberate Great Reset. The architecture is frozen and **Milestone 1—the read-only filesystem catalogue—is implemented** in Go for the Windows control plane. It is an engineering milestone, not yet a production release.

The current implementation can enroll a filesystem directory without modifying it, perform complete stable SHA-256 hashing, group exact duplicates as Appearances of one Content item, preserve rename continuity where evidence permits, search by name or path, and serve the catalogue through a secured loopback-only browser UI. Every discovered Appearance remains user-owned.

Copy creation, managed custody, Protection Contracts, durable Boatman Jobs, Provider-local `.sampo` metadata, S3, deletion, and repair remain unimplemented and unavailable.

## Approved MVP

The first release is one single-user, single-node local application with a loopback-only browser interface.

It proves one complete journey:

1. The user connects and explicitly enrolls a storage provider.
2. SAMPO discovers and hashes existing files without changing them.
3. The user searches by name and sees one exact Content item with its known Appearances.
4. The user requests **Keep a copy** and reviews an explainable Plan.
5. Approval creates a persistent Protection Contract and durable Job.
6. Boatman stages, commits, and independently verifies an additional ordinary copy.
7. Only verified committed bytes become SAMPO-managed and satisfy the Contract.
8. If the original provider becomes unavailable, Open routes to the best verified available Appearance.

The MVP proves genuine provider abstraction with:

- a mounted hierarchical filesystem provider; and
- an S3-compatible object-storage provider used as object storage rather than a fake drive.

The accepted implementation ADRs select Go, SQLite, AWS SDK for Go v2, and a pinned SeaweedFS development service while keeping Provider contracts independent of those choices.

Windows hosts the first SAMPO control plane. Ubuntu Server participates first through S3-compatible storage. Linux control-plane support remains a portability goal rather than an MVP requirement.

## Core concepts

- **Content:** one exact byte sequence recognized by complete digest evidence.
- **Appearance:** one occurrence of Content at a provider-native opaque locator.
- **Custody:** user-owned or SAMPO-managed authority over an Appearance, established by provenance or explicit adoption—never by path.
- **Protection Contract:** a persistent approved obligation over exact Content.
- **Failure Domain:** an explicitly represented loss boundary shared by one or more Providers or storage instances; Provider identity alone does not establish independence.
- **Plan:** an explainable proposal without execution authority until approved.
- **Job:** a durable, retryable, idempotent unit of already authorized work.
- **Observation:** untrusted evidence from a provider or watcher.
- **Catalogue Fact:** a reconciled statement Seshat currently accepts.

The MVP has no canonical copy. Read and edit access are routed per operation among eligible verified Appearances.

An unconstrained **Keep two copies** Contract requires at least two distinct, verified, committed SAMPO-managed Appearances across two independent failure domains. A filesystem Provider and an S3-compatible Provider can still share one failure domain when both ultimately use the same disk. If independent placement is unavailable, the Contract remains unfulfilled unless the user explicitly approves the weaker same-domain alternative.

Unavailable is not missing. SAMPO reports an unavailable managed Appearance as currently unverifiable and does not replace it merely because its Provider disconnected. Surplus managed Appearances above a required minimum are reported but not automatically removed.

Filesystem-managed copies remain human-readable under `library/`, using the first approved relative path. Machine metadata and staging remain under `.sampo/`. Later source renames update Seshat rather than silently rebuilding managed paths.

A file manually placed under `library/` remains user-owned. SAMPO offers **Adopt**, **Leave it mine**, or **Ask later**; only explicit adoption transfers custody.

SAMPO reports factual Provider usage and failures but does not estimate cloud bills or enforce Provider budgets. Paid-Provider budgets and billing alerts remain Provider-side controls.

## Staff responsibilities

SAMPO is the product, local application, and orchestration/control-plane boundary. Its Staff names are logical responsibilities, not implied processes or services:

- **Gateway** serves the local browser and translates user actions into bounded requests.
- **Observer** reports untrusted external observations and never acts on them directly.
- **Seshat** owns reconciled catalogue knowledge and Contract state.
- **Tuoni** creates explainable Plans without storage I/O.
- **Boatman** executes approved Jobs without inventing policy.
- **Caretaker** detects Contract maintenance needs and triggers only already authorized work or suggestions requiring approval.

Direct in-process queries and commands are allowed. Durable domain events record committed facts; SAMPO does not depend on a fire-and-forget event bus for correctness.

## Safety boundary

Pre-existing and externally created files are user-owned. SAMPO may read, hash, catalogue, observe, search, and copy from them, but may not overwrite, truncate, rename, move, delete, replace, or silently adopt them.

An Appearance becomes SAMPO-managed only when SAMPO created it through an approved recorded Job or the user explicitly adopted it. Managed-copy retirement is allowed only through explicit Contract amendment and only when no active Contract still requires that Appearance.

Provider-local metadata uses an explicitly permitted `.sampo/` control area. Loss of `.sampo/` or the home catalogue must leave ordinary data readable; uncertain custody reverts to user-owned safety.

## Explicit MVP exclusions

The MVP does not include:

- synchronization or automatic edit propagation;
- automatic migration, tiering, eviction, or destructive deduplication;
- deletion of user-owned data;
- Git-like revision history or merge behavior;
- SMB, WebDAV, virtual drives, or filesystem overlays;
- multi-node or multi-user operation;
- public or LAN administration;
- semantic or AI-required search;
- arbitrary cloud providers beyond the local S3-compatible proof.

See the [Parking Lot](PARKING-LOT.md) for the complete deferred list.

## Architecture documents

Read in this order:

1. [Manifesto](MANIFESTO.md)
2. [Product specification](Spec.md)
3. [MVP architecture decisions](MVP-ARCHITECTURE-DECISIONS.md)
4. [Implementation ADR ownership](SAMPO-IMPLEMENTATION-ADR-OWNERSHIP.md)
5. [Implementation ADRs](IMPLEMENTATION-ADRS.md)
6. [MVP architecture](MVP-ARCHITECTURE.md)
7. [MVP acceptance tests](MVP-ACCEPTANCE-TESTS.md)
8. [Implementation milestones](IMPLEMENTATION-MILESTONES.md)
9. [Parking Lot](PARKING-LOT.md)

The historical [architecture decision handoff](mash-mvp-architecture-decision-record-and-codex-handoff.md) records the meeting that established these decisions. It predates the final product-naming correction and therefore contains the retired working title **MAS-H**.

The historical [Great Reset audit](sol-audit.md) is non-authoritative archaeology. It predates the approved decisions and naming correction and must not be used as current implementation guidance.

## Implementation status

Milestone 1 uses Go 1.26, `net/http`, server-rendered HTML, SQLite through `modernc.org/sqlite`, and Windows handle identity through `golang.org/x/sys/windows`.

Run the local development application:

```powershell
go run ./cmd/sampo
```

SAMPO binds to an operating-system-assigned `127.0.0.1` port, opens a one-time bootstrap URL in the default browser, and stores its home catalogue under the current user's local application-data directory. For an isolated development catalogue without automatic browser launch:

```powershell
go run ./cmd/sampo -data-dir .sampo-data -no-browser
```

Verify the implementation:

```powershell
go test ./...
go vet ./...
go build ./cmd/sampo
```

The test suite covers provider non-mutation, complete hashing, exact duplicate grouping, rename continuity, changed-byte history, SQLite durability settings and corruption rejection, session bootstrap, Host and Origin enforcement, CSRF protection, and the Milestone 1 journey.
