# SAMPO MVP Architecture Decisions

**Status:** Product-owner approved
**Authority:** `MANIFESTO.md` remains constitutional. This document records the approved MVP decisions and supersedes conflicting provisional language in `Spec.md` and stale material in `README.md`. Decision 14 incorporates the later product-owner naming correction that retired the MAS-H working title. `SAMPO-IMPLEMENTATION-ADR-OWNERSHIP.md` records subsequent product-owner constraints and delegates bounded implementation choices without reopening these decisions.

This document defines product and architecture constraints, not implementation technologies. It does not select a programming language, framework, database engine, object-store implementation, or packaging model.

## 1. Accessibility, custody, and the First Law

For the MVP, SAMPO may discover, read, hash, catalogue, search, observe, and create additional ordinary copies. It may not destructively alter user-originated data.

A **user-owned appearance** is a pre-existing file or a file created outside SAMPO. SAMPO may read, hash, catalogue, observe, search, copy from, and associate non-destructive catalogue metadata with it. SAMPO may not overwrite, truncate, rename, move, delete, replace, or silently adopt it.

A **SAMPO-managed appearance** is established only when SAMPO created it through a recorded approved job or the user explicitly adopted it into SAMPO custody. Managed appearances may be reorganized, recreated, replaced, moved, or retired only within active contract authority and provider permissions.

Custody is established by provenance or explicit adoption, never by path. If custody cannot be proven, SAMPO treats the appearance as user-owned. Removing SAMPO must leave every committed file readable through ordinary provider-native tools.

## 2. Exact content identity, appearances, and rename handling

One logical **Content** represents one exact byte sequence proven by a complete cryptographic digest. Content equality ignores filename, path, drive letter, provider, timestamps, and naming choices.

Each occurrence is a separate **Appearance**, recording its provider, opaque locator, current and observed names, custody, availability, verification state, observation times, and provider-native identity evidence.

Repeated protection requests for byte-identical Content are idempotent. SAMPO reports the existing contract and appearances rather than creating an unnecessary managed copy.

A reliable provider-native identifier or rename event is strong continuity evidence. One disappeared locator and one unambiguous new locator with the same complete hash may be recorded as a probable rename or move. If both remain, they are two appearances. Ambiguous candidates remain uncertain. A hash proves byte equality, not the human action that produced it.

Internal identifiers may be opaque, and a digest need not be a database primary key. Logical equality must never depend on path or provider identity.

## 3. External edits create forks, not synchronization

When an Appearance changes bytes, it represents new Content. SAMPO preserves the changed bytes, leaves all other appearances unchanged, and performs no propagation, merge, rollback, or automatic overwrite.

The change may be described as a human-visible fork. The MVP does not build a Git-like revision graph, commit model, merge engine, or automatic ancestry inference. A minimal `derived-from` fact is allowed only when SAMPO directly witnessed the transition, and MVP correctness must not depend on it.

## 4. Protection contracts persist independently of fulfillability

A **Protection Contract** is a persistent user-approved obligation over exact Content, such as keeping two verified SAMPO-managed copies.

User-owned appearances do not satisfy a managed-copy count unless explicitly adopted. A contract is:

- **Fulfilled** when the required verified managed appearances exist.
- **Unfulfilled but fulfillable** when work is required and exact source bytes plus an approved eligible destination are available.
- **Unfulfillable** when exact bytes are inaccessible or approved terms cannot currently be met.

Reality may make a contract unfulfillable, but only the user may amend or cancel it. If exact Content later returns, SAMPO recognizes it and may resume fulfillment within the already approved terms. Similar, edited, or probable content never satisfies an exact-content contract.

### Copy count and failure domains

An unconstrained **Keep two copies** Contract means:

> Maintain at least two distinct, verified, committed SAMPO-managed Appearances of the exact same Content across two independent failure domains.

The count is a required minimum, not an automatic deletion target. Surplus managed Appearances are reported and remain in place unless a later approved Contract amendment authorizes retirement.

A user-owned Appearance does not count unless explicitly adopted. Planned, staged, partial, unverified, digest-mismatched, or aliased references to the same underlying storage instance do not count.

Provider identity and failure-domain identity are separate. Different Providers do not prove independence: a filesystem Provider and a local S3-compatible Provider may share one physical disk, host, enclosure, or other loss boundary. SAMPO must record or establish the relevant failure-domain relationship separately. Unknown independence does not satisfy an unconstrained multi-copy Contract.

If approved Providers cannot supply two independent failure domains, the Contract remains unfulfilled and SAMPO explains why. The user may approve a materially weaker Plan allowing two distinct managed instances within one failure domain; that exception becomes an explicit Contract term.

Provider unavailability is not deletion. An unavailable managed Appearance remains known but currently unverifiable. It is reported separately from available and confirmed-missing Appearances, and its unavailability alone does not authorize replacement. Replacement occurs only after the Appearance is positively confirmed missing or invalid.

## 5. Home catalogue and provider-local `.sampo` memory

Each installation has a home **Seshat catalogue** containing its complete known view: providers, Content, Appearances, custody, Contracts, Jobs, verification state, projects, observations, and audit history.

An explicitly enrolled writable provider may contain one hidden `.sampo/` control area. It contains one compact provider-local ledger or equivalent, not one sidecar per user file. It may preserve provider identity, schema version, local inventory, content hashes, custody evidence, SAMPO-created copy records, scan checkpoints, prior observations, installation attribution, and compact reconnect history.

Provider-local memory is portable provenance, not distributed consensus. A provider may travel between installations; each compares its home catalogue, provider-local evidence, and current provider reality without trusting any one source blindly.

No SAMPO metadata may be required to open committed bytes. Metadata loss must leave ordinary data readable. Reconstruction recovers only what surviving evidence proves. If custody becomes uncertain, destructive authority is lost and affected appearances become user-owned for safety. Provider-native metadata and per-file sidecars may be optional redundant evidence later, never the sole byte-access dependency.

Creating `.sampo/` on user-originated media requires explicit enrollment and permission.

## 6. Unknown-provider prompt and enrollment

For an unknown storage medium, SAMPO asks:

> I see you connected a drive I haven’t seen before. Would you like this to become part of SAMPO?

Choices are:

- **Yes:** open setup and separately request permission to catalogue existing files, maintain `.sampo/`, and use the provider for managed copies.
- **No:** persist exclusion and stop asking for that recognized provider.
- **Ask me next time:** make no change and ask again on reconnect.

Provider recognition uses the best persistent identity evidence available, not a current drive letter alone. Unknown media is not modified before enrollment and the relevant permission grants.

## 7. No canonical copy; operation-specific routing

The MVP has no canonical-copy concept. Verified byte-identical appearances contain the same Content; no appearance is inherently more true.

SAMPO selects an eligible Appearance per operation:

- **Read/Open:** prefer verified available local fast storage, then other local storage, local-network storage, and remote object storage. Unavailable appearances remain visible as unavailable.
- **Edit:** prefer a writable user-owned local appearance. Never default to editing a managed preservation copy. If only a managed copy exists, explicitly create or propose a user-owned working copy.
- **Share/remote use:** a remote appearance may be preferable when that operation calls for it.

Preferred access is temporary routing, not authority, ownership, synchronization, or canonical status.

## 8. Projects are overlapping collections with snapshot actions

A Project is a Seshat collection, not a directory or provider location. The same Content may belong to multiple Projects. Membership changes catalogue metadata only and never moves, copies, renames, or deletes bytes.

An MVP project-level action operates on a reviewed snapshot. Approval creates file-level Contracts for the current unique Content members. Later additions do not inherit protection silently; the Project reports them and lets the user approve a later batch.

Living project contracts, automatic inheritance, draft retention, garbage detection, and project-wide fork management are deferred.

## 9. First complete workflow and local browser interface

The first release proves this vertical journey:

1. Connect an unknown drive and answer the enrollment prompt.
2. Discover and hash files without changing them.
3. Search by name and show one logical Content item with known Appearances.
4. Select **Keep a copy** and review destination and required space.
5. Approve the Plan.
6. Boatman stages, commits, and independently verifies a new copy.
7. Only the verified result becomes managed and contract-satisfying.
8. Disconnect the original provider.
9. Search again and open the best available managed Appearance.

The MVP UI is a local browser application served by the SAMPO process. It is search-first, loopback-only, visually simple, and not a public website, CLI-only product, SMB/WebDAV endpoint, virtual drive, native .NET requirement, or Electron requirement.

The home screen prioritizes search, provider availability, contract attention, and recent activity rather than storage-administrator controls.

## 10. Provider proof: filesystems and S3-compatible object storage

The MVP supports two genuinely different provider classes:

1. **Mounted hierarchical filesystem:** ordinary mounted internal, external, USB, or network storage with provider-specific subsets of enumerate, read, create, rename, managed-file removal, native file identity, hierarchy, availability, and performance evidence.
2. **S3-compatible object storage:** used as object storage, not mounted as a fake drive. It proves key-based addressing, non-directory semantics, remote latency, upload staging, different commit and integrity behavior, lack of inode identity, and transfer cost/availability.

Development must be possible against a disposable local S3-compatible service without a public cloud account. The first control plane runs on Windows; Ubuntu Server participates first by hosting S3-compatible storage. No particular S3-compatible implementation is selected here.

An ETag or successful upload response alone is not proof of exact equality. A copy satisfies a Contract only after SAMPO verifies its complete digest or an equivalently strong independently validated result. Both classes store ordinary original bytes without a proprietary wrapper as the sole representation.

Other cloud, Git, GitHub, R2, and SAMPO-node providers are deferred.

## 11. First-release security boundary

The MVP is single-user, single-node, one local installation, and local-only by default. Gateway binds to loopback only.

There are no SAMPO accounts, roles, shared libraries, LAN control plane, public listener, remote administration, or multi-principal ACL mapping. SAMPO uses the permissions of its operating-system user.

The browser UI still requires appropriate local session, origin, request-forgery, body-size, and timeout protections. Credentials must not be embedded in ordinary user files or provider-visible per-file metadata.

## 12. Contract approval authorizes bounded continuous maintenance

One approved Plan creates a Protection Contract and authorizes the work needed to maintain it within its terms. Boatman may stage, retry, resume, verify, clean SAMPO-owned staging, recreate a positively confirmed missing or invalid managed copy, restore the required managed count and failure-domain constraints, and report the action without asking the same question repeatedly.

New approval is required for a material change: a new Provider, changed copy count, changed durability/location constraint, touching user-owned data, or expansion to unrelated Content. Enabling a paid Provider makes it eligible within approved Contracts; SAMPO reports factual usage but does not estimate bills or create provider-budget authority. Observations may suggest Contracts but never create them automatically.

Positive confirmation that a required managed Appearance was deleted or became invalid is loss, not contract cancellation. If exact source bytes remain and approved terms permit, SAMPO recreates and verifies it. Provider disconnection or temporary unavailability is not positive confirmation and does not trigger replacement.

If a managed Appearance is externally edited, SAMPO preserves the edit, transfers that Appearance to user custody, stops counting it for the old Content, and repairs the old Contract only if the old bytes remain accessible. Otherwise the Contract becomes unfulfillable but remains active.

## 13. Managed-data deletion is contract amendment

Deleting a managed Appearance and reducing or cancelling its Contract are distinct.

If an active Contract requires an Appearance, SAMPO explains that simple deletion will cause replacement. The user may choose another destination, amend the Contract, or cancel the request.

Reducing a Contract previews the exact managed Appearance to retain and retire and explicitly excludes user-owned data. After approval, Boatman may retire managed appearances no longer required by any active Contract.

Cancelling protection asks whether to transfer managed copies into user custody, remove eligible managed copies, or cancel nothing. Amendment grants destructive authority only over proven managed appearances no longer required by any active Contract.

## 14. SAMPO product and naming law

- Product and human-facing name: **SAMPO**
- Retroactive expansion: **Storage Abstraction Management & Policy Orchestrator**
- Repository, executable, package, and command identifier: `sampo`
- Provider metadata directory: `.sampo`
- Application data and configuration names use `sampo`

MAS-H was an earlier working title and is retired. It appears only in historical material that predates this correction.

SAMPO is the product, the local application, and the orchestration/control-plane boundary. It is not necessarily a collection of processes, a message broker, a microservice platform, or the owner of user bytes.

The product origin remains part of its Manifesto:

> SAMPO came first.
>
> The name existed before the architecture and before anyone knew what the letters were supposed to mean.
>
> Later, it was retroactively expanded into:
>
> **Storage Abstraction Management & Policy Orchestrator**
>
> **What exactly is SAMPO?**
>
> “Your guess is as good as mine. I was high. I liked the name.”
>
> — Arttu Pitou, Founder

For the MVP, all Staff responsibilities run in one local application with clear module and authority boundaries:

- Gateway translates local-browser actions into bounded requests.
- Observer reports untrusted external observations and never acts.
- Seshat maintains reconciled catalogue knowledge and Contract state.
- Tuoni creates explainable Plans without storage I/O.
- Boatman executes approved Jobs without inventing policy.
- Caretaker checks Contracts and maintenance needs, triggering only already authorized work or suggestions requiring approval.

Staff may use direct queries and commands in-process. Durable domain events record committed facts; event-driven does not mean turning every call into asynchronous messaging.

## 15. Managed-copy layout

Committed filesystem-managed Appearances use readable paths under `library/`. SAMPO preserves the first relative human-readable path approved for that copy and preserves the original file extension.

Later byte-identical source renames update Seshat’s observed names and locators but do not automatically rebuild the committed managed path. Machine metadata and staging remain under `.sampo/`.

When names collide or a Provider cannot represent the source path exactly, SAMPO applies a deterministic readable disambiguation selected by implementation ADR. Committed files remain directly retrievable without SAMPO. Directory location alone never establishes managed custody; a file manually placed under `library/` remains user-owned until explicit adoption.

## 16. Initial platform topology

Windows hosts the first SAMPO control plane and local browser UI. Ubuntu Server participates first through S3-compatible storage. Linux control-plane support is deferred, while portability remains an architecture constraint.

The implementation ADR selects Windows application lifecycle, volume notifications, file identity, watching, default-application integration, application-data paths, packaging, updates, and privilege boundaries.

## 17. Adoption behavior

A file manually placed in SAMPO-managed space remains user-owned. Observer notices it and SAMPO queues a durable prompt with:

- **Adopt:** explicitly transfer the Appearance into SAMPO custody after verification and any required Contract association.
- **Leave it mine:** retain user custody and do not ask again for that observed Appearance unless materially changed.
- **Ask later:** retain user custody and preserve the prompt for later review.

Location is never custody. Adoption must account for changed bytes, multiple potentially relevant Contracts, and the case where no Contract exists. Custody transfer is audited.

## 18. Factual usage reporting

SAMPO reports facts such as bytes stored, uploaded, and downloaded; operation counts where exposed; last successful operations; quota/provider errors; billing-related provider failures; and transfer history.

SAMPO does not estimate currency bills or enforce Provider budgets. Enabling a paid Provider makes it eligible under approved Contract terms. Setup warns the user to configure budgets, quotas, and billing alerts with the Provider.

## Implementation delegation

The product owner delegates the following choices to implementation, with recorded alternatives and acceptance-test evidence. The accepted selections are in `IMPLEMENTATION-ADRS.md`:

- language and web framework;
- catalogue and provider-ledger storage libraries;
- digest algorithm and migration approach;
- local S3-compatible development tool;
- operating-system data locations;
- watcher and scan strategy;
- reconciliation cadence;
- local-session and request-forgery protection;
- staging names and provider-specific atomic commit mechanics;
- initial operating-system portability;
- provider-native retrieval guidance for S3 objects;
- provider-local history retention;
- whether to retain directly witnessed minimal fork lineage.

These are implementation choices unless they change visible behavior, safety, cost, or future compatibility.
