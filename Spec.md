# SAMPO

## Storage Abstraction Management & Policy Orchestrator

## Vision

SAMPO is a storage orchestration product and Digital Librarian.

Its purpose is to make physical storage irrelevant to human memory. Users should think in terms of information, projects, relationships, protection, and search rather than disks, partitions, filesystems, buckets, or drive letters.

Storage abstraction is successful when the computer remembers where exact content exists, how it may be accessed, which appearances SAMPO manages, what the user has asked SAMPO to protect, and what safe work remains.

SAMPO operates at the file and object layer. It is not block-level virtualization, a proprietary filesystem, a sync engine, or a replacement for ordinary provider tools.

## Product laws

The [Manifesto](MANIFESTO.md) is constitutional. In summary:

1. SAMPO must never make user data less accessible than it was before SAMPO existed.
2. The user expresses intent; the Librarian determines implementation.
3. Existing tools should be orchestrated before new replacements are invented.
4. Human cognitive time is the primary optimization target.
5. Providers are implementation details, not Content identities.

If SAMPO is removed:

- committed files remain ordinary provider-native bytes;
- ordinary filesystems remain intact;
- provider-native tools can still retrieve committed data;
- loss of SAMPO metadata does not prevent byte access;
- reconstruction recovers only facts supported by surviving evidence.

SAMPO uses no proprietary wrapping as the only representation of Content and no mandatory catalogue for decoding bytes.

## Current MVP boundary

The first release is one single-user, single-node Windows application with clear logical modules and a loopback-only browser UI. Ubuntu Server participates first through S3-compatible storage. Linux control-plane support follows later, while portability remains required.

The MVP is search-first. It discovers and hashes user-owned files without modifying them, recognizes exact Content across Appearances, creates user-approved Protection Contracts, and produces additional verified ordinary copies through durable Jobs.

The MVP proves provider abstraction with two classes:

- mounted hierarchical filesystems; and
- S3-compatible object storage used through native object semantics.

The first release is not physically distributed and does not require microservices, a network event broker, a public API, a virtual filesystem, SMB, or WebDAV.

## The Digital Librarian

Knowledge about storage is the product. SAMPO should know, with explicit evidence and uncertainty:

- which exact Content is known;
- where each Appearance was observed;
- which Appearances are available and verified;
- which Appearances are user-owned or SAMPO-managed;
- which Providers are enrolled and what they can safely do;
- which Projects reference Content;
- which Protection Contracts are fulfilled, fulfillable, or unfulfillable;
- which Jobs are planned, active, failed, or complete;
- why an action occurred and what should happen next.

The user should not need to remember those things.

## Core domain vocabulary

### Content

Content is one exact byte sequence recognized by a complete verified cryptographic digest. It has a stable internal identity independent of path, filename, provider, drive letter, timestamp, and naming choice.

Byte-identical data is treated as one logical Content item with multiple Appearances. The digest need not be the internal database identity; algorithm migration must remain possible.

### Appearance

An Appearance is one occurrence of Content at one Provider and provider-native opaque locator. It records:

- Provider and locator;
- current and observed names;
- user-owned or SAMPO-managed custody;
- availability and verification state;
- first and last observation;
- provider-native identity evidence;
- staged, committed, missing, changed, or uncertain state.

Paths and object keys are provider-native locators, never global Content identities.

### Custody

Custody determines destructive authority over an Appearance.

A pre-existing or externally created Appearance is user-owned. SAMPO may discover, read, hash, catalogue, search, observe, and copy from it, but may not destructively alter it.

An Appearance becomes SAMPO-managed only through an approved recorded creation Job or explicit user adoption. Custody is established by provenance or adoption, never directory location. Ambiguity always resolves to user-owned safety.

### External changes and forks

When an Appearance changes bytes, it represents new Content. SAMPO preserves the edit and performs no automatic propagation, merge, rollback, or synchronization.

If the changed Appearance was managed, it immediately becomes user-owned and stops satisfying the old Content’s Contract. SAMPO may replace the lost managed copy only if exact old bytes remain accessible and the existing Contract authorizes replacement.

A provider-native identifier or reliable rename event is strong continuity evidence. Complete hash equality can support probable rename reasoning, but a hash proves byte equality rather than the exact human action. Ambiguous histories remain uncertain.

The MVP may retain minimal directly witnessed `derived-from` facts, but it does not build a Git-like revision graph, commit model, or merge engine.

### Project

A Project is an overlapping logical Seshat collection, not a directory or provider location. Content may belong to multiple Projects. Membership changes catalogue metadata only.

MVP project protection uses a reviewed snapshot. Approval creates Content-level Contracts for current unique members. Later additions do not inherit automatically; the Project quietly reports them for optional later batch approval.

### Protection Contract

A Protection Contract is a persistent user-approved obligation over exact Content, such as keeping two verified SAMPO-managed copies.

It remains active when temporarily unfulfilled or unfulfillable. Reality cannot cancel user intent. Only the user may amend or cancel a Contract. If exact Content later returns, SAMPO may resume fulfillment within the already approved terms.

User-owned duplicates do not count toward a managed-copy obligation unless explicitly adopted. Similar, edited, or probably related bytes never satisfy an exact-Content Contract.

An unconstrained **Keep two copies** Contract requires at least two distinct, verified, committed SAMPO-managed Appearances of exact Content across two independent failure domains. This is a minimum, not an exact ceiling.

A Failure Domain is a separately represented loss boundary, not a synonym for Provider. Two Providers may depend on the same disk, host, enclosure, account, or service and therefore fail together. SAMPO must establish sufficient independence evidence rather than infer it from Provider count. If independent approved placement is unavailable, the Contract remains unfulfilled and explains the constraint. The user may explicitly approve a weaker same-domain Contract term.

Provider unavailability leaves an Appearance known but currently unverifiable. It is not positive evidence of deletion and does not authorize replacement. Replacement occurs only after reconciliation positively confirms that a required managed Appearance is missing or invalid. Managed Appearances above the required minimum are reported and are never automatically removed.

### Plan and Job

Tuoni converts intent, current Catalogue Facts, Contract terms, and Provider capabilities into an explainable Plan. A Plan has no execution authority until approved.

Approval creates or amends a Contract and records durable bounded Jobs. Boatman executes Jobs without inventing policy. A Job records authority, idempotency, preconditions, source evidence, destination, staging, verification, retries, result, and audit history.

A copy counts toward a Contract only after complete independent byte verification and safe provider commit.

### Observation and Catalogue Fact

An Observation is untrusted evidence from a Provider, scan, watcher, reconnect, or completed operation. It may suggest appearance, disappearance, movement, or change, but it does not directly rewrite truth.

A Catalogue Fact is a reconciled statement Seshat accepts with supporting evidence and confidence. Negative listings and Provider unavailability never directly authorize deletion.

### No canonical copy in the MVP

All verified byte-identical Appearances contain the same Content. The MVP has no canonical-copy status.

SAMPO routes each operation to the best eligible Appearance. Open generally prefers verified available local storage over remote storage. Edit prefers a writable user-owned local Appearance and never defaults to modifying a managed preservation copy. Remote-oriented operations may deliberately select remote storage.

Routing is temporary operation choice, not authority, ownership, synchronization, or canonical status.

## Protection and managed-copy maintenance

One user approval authorizes bounded continuous work within the Contract terms:

- staging and verification;
- safe retry and resume;
- cleanup of SAMPO-owned staging;
- replacement of a positively confirmed missing or invalid managed copy;
- restoration of approved managed-copy count and failure-domain constraints;
- audit and reporting.

A new Provider, changed copy count, changed durability/location constraint, interaction with user-owned data, or expansion to unrelated Content is material and requires a new Plan and approval. Once a paid Provider is enabled, factual usage within approved Contract terms does not require speculative per-operation currency approval from SAMPO.

Changing required failure-domain independence or accepting a same-domain alternative is a material durability change and requires explicit approval.

Positive confirmation that a required managed Appearance was deleted or became invalid does not cancel its Contract; SAMPO will replace it where fulfillable. Provider unavailability does not trigger replacement. Permanent reduction requires Contract amendment. Cancellation asks whether managed Appearances should transfer to user custody, be removed where eligible, or remain unchanged.

SAMPO never retires a user-owned Appearance and never retires a managed Appearance still required by any active Contract.

## Provider model

Providers are capability-addressable, not capability-equivalent. Each advertises explicit support and guarantees for operations such as:

- enumerate and stat;
- read;
- isolated staging/write;
- atomic or conditional publish;
- managed-data removal;
- server-side copy;
- native identity or native version evidence;
- name and hierarchy preservation;
- free-space, cost, health, and availability reporting;
- notifications;
- conditional writes, leases, or locks;
- removable operation;
- immediate or eventual consistency.

The core refuses a Plan whose required guarantee is absent. It does not silently fall back to weaker semantics.

### Mounted filesystem Provider

This class covers internal, external, USB, and mounted network storage where the operating system presents filesystem semantics. Individual Providers may differ in identity, atomic rename, case sensitivity, permissions, removability, free-space reporting, and watcher support.

### S3-compatible object-storage Provider

This class uses object keys rather than pretending keys are filesystem paths. It demonstrates remote latency, different commit semantics, multipart staging, lack of inode identity, provider-specific integrity evidence, availability, and cost.

Development uses a locally runnable disposable SeaweedFS service selected by implementation ADR and accessed only through a replaceable S3-compatible adapter. Ubuntu Server hosts the first S3-compatible participation topology. No public cloud account is required for MVP development. A successful upload or ETag alone is not complete-content verification.

## Layered metadata

Each installation has a home Seshat catalogue containing its complete known view.

An explicitly enrolled writable Provider may contain one `.sampo/` control area with a compact Provider-local ledger or equivalent. It may preserve Provider identity, schema/version evidence, Appearance inventory, hashes, custody provenance, SAMPO-created copy records, scan checkpoints, installation attribution, and reconnect history.

Provider-local memory supports a drive moving between installations. Each installation compares its home catalogue, Provider-local evidence, and current provider reality. This is portable provenance, not distributed consensus.

No metadata is required to open Content. Loss of `.sampo/` or the home catalogue preserves byte access. Rebuild recovers only proven facts. Lost custody evidence removes destructive authority and yields user-owned safety.

## Provider enrollment

Unknown media is not modified before the user chooses:

- **Yes:** enter setup and grant catalogue, `.sampo/`, and managed-copy-destination permissions separately.
- **No:** persist exclusion for the recognized Provider.
- **Ask me next time:** make no change and ask on reconnect.

Enrollment does not adopt existing files. Provider recognition uses the best persistent identity evidence available, not the current mount point alone.

## Staff responsibilities

SAMPO is the product, local application, and orchestration/control-plane boundary. Staff are logical responsibilities inside it:

- **Gateway:** local browser boundary and bounded user requests.
- **Observer:** untrusted reports from external reality; never acts directly.
- **Seshat:** authoritative reconciled catalogue and Contract state.
- **Tuoni:** explainable planning without storage I/O.
- **Boatman:** execution of approved durable Jobs without policy invention.
- **Caretaker:** Contract-drift detection and already authorized maintenance or suggestions requiring approval.

Staff may use direct queries and commands inside the process. Durable domain events record committed facts. Event-driven architecture does not require asynchronous messaging for every interaction and does not use a fire-and-forget bus as a correctness foundation.

## First complete user journey

1. Connect unknown media and answer enrollment.
2. Discover and hash files without changing them.
3. Search by name.
4. See one Content item and its Appearances.
5. Select **Keep a copy**.
6. Review destination and required space.
7. Approve the Plan.
8. Stage and transfer through a durable Job.
9. Independently verify completed bytes.
10. Count the committed Appearance as managed and contract-satisfying.
11. Disconnect the original Provider.
12. Search and Open through the best remaining eligible Appearance.

The UI is a visually simple loopback-only browser application, not a storage-administrator dashboard, public website, CLI-only product, SMB/WebDAV server, or virtual drive.

## First-release security

The MVP is single-user, single-node, and local-only. Gateway binds to loopback. There are no accounts, roles, shared libraries, LAN control plane, public listener, remote administration, or provider ACL mapping.

The local UI still requires session, origin, request-forgery, request-size, timeout, concurrency, and untrusted-display protections. SAMPO uses the operating-system user’s Provider permissions. Credentials are not stored in ordinary user files or visible per-file metadata.

## Managed-copy layout and adoption

Committed filesystem-managed Appearances use readable paths under `library/`, preserve the first approved human-readable relative path and file extension, and remain directly retrievable without SAMPO. Later byte-identical renames update Seshat only; they do not automatically rename committed managed storage. Machine metadata and staging remain under `.sampo/`.

Collision suffixes, sanitization, reserved names, Unicode normalization, length limits, and unrepresentable paths are implementation ADRs. They must remain deterministic and readable.

A file manually placed in managed space remains user-owned. SAMPO notices it and offers **Adopt**, **Leave it mine**, or **Ask later**. Only explicit adoption transfers custody, and the transfer is verified, associated with applicable Contract authority, and audited.

## Factual Provider usage

SAMPO may report bytes stored, uploaded, or downloaded; operation counts exposed by a Provider; last successful operations; quota errors; billing-related failures; and transfer history.

SAMPO does not estimate currency bills or enforce Provider budgets. Enabling a paid Provider makes it eligible under approved Contracts. Enrollment warns users to configure budgets, quotas, and billing alerts using Provider-side controls.

## Search first

Search is the primary MVP interaction. Baseline search covers names and deterministic catalogue fields such as Provider, availability, custody, Project, Contract status, digest, and activity.

AI-assisted semantic search is a future option and must never become necessary for identity, correctness, integrity, Contract safety, recovery, or ordinary search.

## Future aspirations, explicitly deferred

The domain and module boundaries should permit later distribution without requiring the MVP to be distributed. Staff responsibilities may eventually move between machines only after multi-node authority, conflict handling, security, and consensus needs are explicitly designed.

Possible later presentation layers include generated Project views, links or materialized working sets, WebDAV, SMB, or a virtual filesystem. “Applications believe they use ordinary files” is a future presentation aspiration, not a current claim. Direct provider-native access remains the safety escape hatch.

The following are deferred:

- automatic migration, source retirement, tiering, cache eviction, and destructive deduplication;
- synchronization, merge, rollback, or Git-like version control;
- living Project Contracts and automatic inheritance;
- public APIs, remote administration, multi-user libraries, and distributed catalogue writes;
- arbitrary cloud, Git, GitHub, or other SAMPO-node Providers;
- opportunistic maintenance beyond active Contract authority;
- semantic indexing, AI classification, and arbitrary relationship ontologies.

See [PARKING-LOT.md](PARKING-LOT.md) for the normative exclusion list.

## Existing technology research

Technology names in historical research—including filesystem search tools, transfer tools, relational catalogues, browser frameworks, S3-compatible services, and provider-native watchers—are candidates only unless selected by an accepted implementation ADR.

No language, database, framework, storage server, search engine, or deployment system is selected merely because it appears in historical documents or the retired prototype. The current selections—Go, SQLite, AWS SDK for Go v2, and SeaweedFS as disposable test infrastructure—derive from `IMPLEMENTATION-ADRS.md`.

Every selection records:

- alternatives considered;
- safety consequences;
- portability consequences;
- build-versus-integrate reasoning;
- acceptance-test evidence.

## Accepted implementation direction

Product behavior is approved. `IMPLEMENTATION-ADRS.md` now resolves the delegated choices for:

- language and browser framework;
- durable home-catalogue and Provider-ledger technologies;
- complete digest algorithm and algorithm migration;
- local S3-compatible development dependency;
- Windows lifecycle, notifications, file identity, packaging, and privilege details while retaining future portability;
- watcher, scan, rehash, and reconciliation strategies;
- local-session and request-forgery mechanisms;
- staging names and atomic commit mechanics per Provider;
- Provider-local history retention;
- whether minimal directly witnessed fork lineage is retained.

The ADRs remain revisable only within the product-owner delegation. Questions that alter user-visible behavior, safety, cost, Contract meaning, recoverability, or future compatibility return to the product owner. Low-level choices are proven by the [MVP acceptance tests](MVP-ACCEPTANCE-TESTS.md).
