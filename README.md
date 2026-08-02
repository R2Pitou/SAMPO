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

SAMPO is in an architecture-first MVP phase following a deliberate implementation reset. There is currently no production implementation, build command, runtime configuration, public API, or selected programming language.

The architecture is being documented before implementation so custody, identity, Contracts, provider behavior, failure recovery, and destructive authority are explicit and testable.

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

No specific S3-compatible implementation or other technology has been selected yet.

## Core concepts

- **Content:** one exact byte sequence recognized by complete digest evidence.
- **Appearance:** one occurrence of Content at a provider-native opaque locator.
- **Custody:** user-owned or SAMPO-managed authority over an Appearance, established by provenance or explicit adoption—never by path.
- **Protection Contract:** a persistent approved obligation over exact Content.
- **Plan:** an explainable proposal without execution authority until approved.
- **Job:** a durable, retryable, idempotent unit of already authorized work.
- **Observation:** untrusted evidence from a provider or watcher.
- **Catalogue Fact:** a reconciled statement Seshat currently accepts.

The MVP has no canonical copy. Read and edit access are routed per operation among eligible verified Appearances.

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
4. [MVP architecture](MVP-ARCHITECTURE.md)
5. [MVP acceptance tests](MVP-ACCEPTANCE-TESTS.md)
6. [Implementation milestones](IMPLEMENTATION-MILESTONES.md)
7. [Parking Lot](PARKING-LOT.md)

The historical [architecture decision handoff](mash-mvp-architecture-decision-record-and-codex-handoff.md) records the meeting that established these decisions. It predates the final product-naming correction and therefore contains the retired working title **MAS-H**.

## Implementation status

Implementation has not begun. Technology and dependency choices will be recorded separately only when a milestone requires them, with alternatives, safety consequences, portability consequences, build-versus-integrate reasoning, and acceptance-test evidence.
