# SAMPO Implementation Architecture Decision Records

**Status:** Accepted for implementation planning  
**Decision date:** 2026-08-02  
**Authority:** These ADRs exercise the delegation in `SAMPO-IMPLEMENTATION-ADR-OWNERSHIP.md`. They are subordinate to `MANIFESTO.md`, `MVP-ARCHITECTURE-DECISIONS.md`, and the settled product constraints. They select mechanisms; they do not weaken custody, Contract, recoverability, or First-Law behavior.

No schema or implementation interface is defined here. Dependency versions are pinned in source when the relevant milestone begins and are reviewed before upgrade.

## ADR-001: Language, application stack, and module shape

**Decision.** Implement SAMPO in Go, initially on the current supported Go 1.26 line. Produce one `sampo` process containing the Staff responsibilities as explicit in-process modules. Gateway uses `net/http`, `html/template`, embedded static assets, server-rendered HTML, CSS, and narrowly scoped vanilla JavaScript. No SPA framework, Node.js runtime, microservices, or network event broker is part of the MVP.

The package topology follows domain authority rather than Staff theatre: domain types and invariants; Seshat application commands/queries; Tuoni planning; Boatman execution; Observer evidence collection; Caretaker scheduling; Provider ports/adapters; Gateway; and Windows host integration. Dependencies point inward. Provider and host-specific values do not enter the global domain model.

**Why.** Go has a stable, officially supported Windows port, simple cross-compilation, a strong standard HTTP/template/tooling stack, cheap bounded concurrency, and single-binary deployment. The current Go release policy supports each major release until two newer releases exist. This fits one local process better than a browser-heavy split application and leaves a realistic later Linux control-plane path. [Go Windows support](https://go.dev/wiki/Windows), [Go release policy](https://go.dev/doc/devel/release), [Go 1.26 notes](https://go.dev/doc/go1.26).

**Alternatives rejected.** .NET offers excellent Windows integration but makes the later Linux control plane and small portable binary less direct. Rust offers strong low-level safety but increases implementation and review cost for a catalogue-heavy product. Electron/TypeScript duplicates runtime and packaging concerns without adding MVP value. Separate services would introduce distributed failure modes before SAMPO needs them.

**Consequences.** Go-specific types, channels, goroutines, HTTP handlers, and database rows stay outside the domain vocabulary. Concurrency is bounded explicitly; goroutines are not durable work. Use Go's standard `testing`, `httptest`, fuzzing, and race tooling for unit, invariant, adapter-contract, and concurrency tests; use Playwright only for a small browser-level acceptance suite. The normal verification commands become `go test ./...`, `go vet ./...`, and `go build ./cmd/sampo` once code exists. Keep direct dependencies minimal, pinned by `go.mod`/`go.sum`, license-reviewed, vulnerability-scanned, and upgraded through tests rather than floating versions.

## ADR-002: Home catalogue and Provider-local persistence

**Decision.** Use SQLite for both the home Seshat catalogue and each writable Provider-local ledger, via the CGo-free `modernc.org/sqlite` driver. Each database is independent; SAMPO never uses an attached cross-database transaction as distributed truth.

Use rollback-journal `DELETE` mode, `synchronous=FULL`, foreign-key enforcement, a bounded busy timeout, short write transactions, and one application-level writer queue per database. Do not use WAL in the MVP. Run `quick_check` at clean startup, `integrity_check` after an unclean shutdown or suspicious I/O result, and enter read-only safe mode on corruption. Create rotating, verified home-catalogue backups through SQLite’s supported online backup mechanism; never copy a live database file casually.

Apply numbered, forward-only migrations in one transaction where SQLite permits. Before any migration, create and integrity-check a backup. A failed migration closes the new database, preserves failure evidence, restores the verified backup, and leaves the application in the prior compatible version; there are no hand-written destructive down migrations. Refuse to open a database whose schema version is newer than the executable. The selected driver build must embed SQLite 3.51.3 or newer because earlier WAL-capable releases contain the documented WAL-reset defect, even though SAMPO does not enable WAL.

**Why.** SQLite is transactional, embedded, inspectable, and sufficient for a single-user process. Rollback journals keep commit state beside the database and avoid WAL’s same-host/shared-memory constraints and extra persistent files on removable media. SQLite documents that `synchronous=FULL` invokes the VFS sync operation before continuing. A WAL-reset corruption bug affected releases through 3.51.2 and is another reason not to take WAL complexity without a demonstrated need. [SQLite locking and rollback journals](https://www.sqlite.org/lockingv3.html), [SQLite WAL constraints and fixed versions](https://www.sqlite.org/wal.html), [SQLite pragmas](https://www.sqlite.org/pragma.html), [SQLite backup API](https://www.sqlite.org/backup.html), [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite).

**Alternatives rejected.** JSON/event files provide poor transactional indexing and crash recovery. A database server creates lifecycle and credential burden. WAL improves concurrent read/write throughput that the single-writer local MVP does not yet need and complicates removable-ledger handling.

**Consequences.** The exact schema remains milestone work. Driver upgrades must confirm the embedded SQLite version, especially corruption fixes. Database loss still cannot trap bytes; recovery is conservative and destructive authority does not arise from an untrusted recovered ledger.

## ADR-003: Identifiers and complete-content digests

**Decision.** Content identity is the tuple `algorithm`, complete digest bytes, and byte length. The initial algorithm is SHA-256, represented externally as `sha256:<lowercase-hex>`. Store the algorithm explicitly everywhere; never infer it from digest length. A verified byte stream is read end-to-end. Provider ETags, multipart composite checksums, timestamps, and sample hashes are evidence, not SAMPO Content identity.

Logical entities whose identity is not their bytes—Provider, Appearance, Project, Contract, Plan, Job, Observation, Event, and Failure Domain—receive opaque 128-bit random SAMPO IDs from the operating-system cryptographic random source. Paths and Provider locators never become global IDs.

**Why.** SHA-256 is ubiquitous in Go and S3 metadata, strong enough for exact-content recognition, and streamable. Algorithm tagging permits a later dual-digest migration without changing entity identity. Go’s `crypto/rand` provides cryptographically secure randomness. AWS explicitly documents that multipart ETags are not whole-object MD5 digests and distinguishes full from composite checksums. [Go SHA-256](https://pkg.go.dev/crypto/sha256), [Go cryptographic randomness](https://pkg.go.dev/crypto/rand), [S3 object integrity](https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity-upload.html).

**Migration rule.** A later digest algorithm is added beside SHA-256. Content records are linked only after one stable read computes both digests or two independently verified reads prove the same bytes. SAMPO never bulk-relabels identities from metadata alone.

## ADR-004: Consistent hashing of mutable sources

**Decision.** Hash from an open handle, not a pathname alone. Capture a pre-read evidence tuple appropriate to the Provider, stream the complete bytes while counting length, and capture post-read evidence from the same handle or object generation. Accept the digest only when length and all available stability evidence agree. On Windows filesystems that evidence includes file ID plus volume serial where supported, size, last-write time, and handle-level metadata. Reopening the path must still identify the same file before the Observation is associated with that locator.

Retry a changing source twice with bounded backoff. A third change marks the Appearance `unstable`; it remains visible but cannot satisfy verification, adoption, or transfer preconditions. Boatman repeats the stability check while reading a transfer source and fails the Job as a stale source if the approved digest/generation no longer holds.

For S3-compatible objects, prefer native version ID and conditional reads. Otherwise compare ETag, size, and last-modified evidence before and after the full read, while treating those fields only as stability tokens. Providers unable to supply either a stable generation or consistent repeated evidence remain unverifiable.

**Why.** Metadata-before-hash alone has a time-of-check/time-of-use gap. Holding a handle and comparing both sides makes acceptance falsifiable without blocking user writes indefinitely. Windows documents file ID plus volume serial as handle identity evidence. [Windows file identity](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-file_id_info), [conditional S3 reads and writes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/conditional-requests.html).

## ADR-005: Local S3-compatible development service and client

**Decision.** Use SeaweedFS `weed mini` on Ubuntu Server as the pinned disposable development and acceptance-test service. Pin a reviewed 4.x release and container image digest; the initial review target is 4.27. Treat it as replaceable test infrastructure, never as a SAMPO library or semantic authority.

SAMPO’s S3 adapter uses AWS SDK for Go v2 with path-style endpoint support and an explicit endpoint resolver. The adapter relies only on a tested capability subset: bucket access, list, head, get, put or multipart upload, abort, delete of proven SAMPO-owned objects, conditional publication where supported, and metadata/checksum round trips. A Provider enrollment probe records which guarantees actually pass.

**Why.** SeaweedFS currently provides an actively maintained one-command S3 development mode and a single binary. MinIO was considered but its community repository was archived in April 2026, so it is not a sound new development dependency. AWS SDK v2 is the maintained protocol client and prevents coupling to one server. [SeaweedFS repository and `weed mini`](https://github.com/seaweedfs/seaweedfs), [SeaweedFS releases](https://github.com/seaweedfs/seaweedfs/releases), [AWS SDK for Go v2](https://docs.aws.amazon.com/sdk-for-go/).

**Consequences.** Passing against SeaweedFS does not imply universal S3 compatibility. Provider-contract tests run against every supported target. Production deployment of the development service is not automatically supported by SAMPO, and its backing disk must be represented in the Failure Domain model.

## ADR-006: Filesystem staging and publication

**Decision.** Stage each Job beneath `.sampo/staging/<job-id>/` on the same filesystem volume as the target `library/`. Create staging files exclusively, stream without touching the destination, flush and close, reopen, and independently recompute SHA-256. Revalidate source, destination absence, Plan revision, Contract authority, and Provider capability immediately before publication.

Publish with a same-volume handle-based rename that does not replace an existing name. On Windows, flush the staging handle and use `SetFileInformationByHandle` rename semantics with replacement disabled; an implementation may use MoveFileEx only after the same Provider probe proves equivalent same-volume, no-replace behavior. Never allow cross-volume copy as “rename.” Treat any destination-exists result as a collision requiring a new Plan locator. Reopen the committed destination by handle and verify identity, length, and complete digest before Seshat records a committed managed Appearance.

Enrollment performs a destructive-to-staging-only capability probe. If the Provider cannot demonstrate exclusive creation, durable flushing, same-volume no-replace publication, and post-publication readback, it is read-only for the MVP. There is no copy-directly-to-final fallback.

**Why.** Windows `CREATE_NEW` fails rather than truncating an existing file, `FlushFileBuffers` pushes buffered file data to the device, and handle-based file information supports rename with appropriate access. Staging and final destination sharing a volume avoids implicit copy/delete publication. [CreateFile creation dispositions](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilea), [FlushFileBuffers](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-flushfilebuffers), [SetFileInformationByHandle](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-setfileinformationbyhandle).

**Crash rule.** Before publication, only SAMPO-owned staging can remain. After publication but before catalogue commit, reconciliation uses Job ID, planned locator, and complete digest to finalize exactly once. Cleanup never follows an unresolved path outside `.sampo/staging/`.

## ADR-007: S3 staging, publication, and verification

**Decision.** A managed S3 key is immutable and includes the approved readable relative name plus a SAMPO Appearance-ID suffix beneath `library/`, so native browsing remains intelligible and retries target the same unguessable key. Upload directly to that final key: incomplete multipart uploads are not committed objects. Use `If-None-Match: *` on `PutObject` or `CompleteMultipartUpload` when the Provider proves support. A 412 is a collision; a multipart 409 restarts with a new upload ID after revalidation. If conditional create/publication is unsupported or fails its enrollment probe, the Provider is read-only for managed publication in the MVP. High-entropy naming and a prior absence check do not substitute for an atomic no-overwrite guarantee.

Supply supported transport checksums as additional evidence, persist multipart upload IDs in the durable Job, and abort only uploads whose ownership is proven by that Job. After completion, perform a full `GET`, compute SAMPO SHA-256, compare length, and repeat a generation/ETag check. Only then may Seshat commit the Appearance. An ETag, successful completion response, or server checksum alone is insufficient for MVP Contract fulfillment.

**Why.** S3 supports conditional `PutObject` and `CompleteMultipartUpload`; concurrent conditional writes fail rather than overwrite. Multipart uploads require explicit abort or lifecycle cleanup, and ETags are not necessarily complete-content digests. [S3 multipart behavior](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html), [S3 conditional writes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/conditional-writes.html), [S3 checksum semantics](https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity-upload.html).

**Crash rule.** Resume uploaded parts when listed evidence matches the durable Job. After ambiguous completion, `HEAD` then full `GET` decide whether to finalize, retry, or require attention. Lifecycle cleanup for incomplete uploads is configured as a backstop, not as correctness authority.

## ADR-008: Managed-copy names and layout mechanics

**Decision.** Filesystem committed data lives below `library/`; machine state lives below `.sampo/`. Normalize planned human-visible components to Unicode NFC. Replace control characters and Provider-illegal characters with readable underscores, prefix reserved Windows device basenames, remove trailing spaces/dots, and truncate only when required by the destination’s measured limits. Preserve the final extension.

The first candidate is the approved relative path. On collision or lossy sanitization, add `~` plus the first 12 SHA-256 hex characters before the extension. If that is also occupied, append the first eight characters of the stable Appearance ID. The resulting locator is fixed in the Plan and therefore deterministic across retries. No numeric “find the next free name” loop is allowed at commit time.

Keep the normal Windows-visible path within 240 UTF-16 code units for MVP interoperability with ordinary tools, even though SAMPO’s host adapter uses extended-length APIs internally. When a hierarchy cannot fit, truncate leafward components with their own stable digest suffix; never flatten silently. Seshat and the Provider ledger preserve the original display path and the actual Provider locator.

**Why.** Windows has reserved characters, device names, trailing-character rules, case-insensitive collisions, and legacy path-length behavior that ordinary applications still encounter. A content suffix is readable and stable; the Appearance suffix resolves same-content/user-owned collisions without overwriting. [Windows naming rules](https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file), [Go Unicode normalization package](https://pkg.go.dev/golang.org/x/text/unicode/norm).

## ADR-009: Provider identity and reconnect confidence

**Decision.** Provider identity is a SAMPO-generated opaque ID supported by independent evidence, never a mount path, drive letter, endpoint credential, or bucket name alone.

For a writable filesystem Provider, `.sampo/provider-id` holds the Provider ID and ledger generation. Home Seshat also records Windows volume GUID/serial, root directory file ID where available, filesystem type, capacity fingerprint, and last observed device evidence. A matching marker plus compatible physical evidence reconnects automatically. A marker match with conflicting volume/root evidence is a probable clone and requires review. Missing or weak evidence reconnects as uncertain and grants no destructive authority.

For S3, identity evidence is endpoint scheme/authority, bucket, any stable service/account owner evidence, TLS endpoint evidence, and a conditionally created `.sampo/provider-id` object when permission allows. Credentials are access authority, not Provider identity. Marker clones or endpoint changes require review.

**Why.** Windows states that volume serial plus file ID can distinguish open file targets, while network providers may return incomplete identity evidence. Multiple factors let SAMPO recognize remounts without pretending clones are the same resource. [GetFileInformationByHandle](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getfileinformationbyhandle).

## ADR-010: Probable rename confidence

**Decision.** A stable Provider-native file ID observed at a new locator on the same Provider is strong continuity evidence. Otherwise SAMPO may display a **probable rename** only when one complete reconciliation interval contains exactly one disappeared and one newly appeared user-owned locator on the same Provider with the same stable SHA-256 and length, and neither side has another ambiguous candidate.

The matching window is the previous successful complete scan plus seven days of observation history. Ambiguous matches remain separate Appearances. Hash-only rename inference changes display continuity and aliases, never Content identity, custody, Job authority, or Contract fulfillment. No probabilistic score triggers mutation; the UI shows the evidence and confidence class.

**Why.** File IDs provide direct continuity on supporting filesystems. One-to-one exact-content matching is understandable, while heuristic scores create false certainty and hidden behavior.

## ADR-011: Failure Domain representation and independence

**Decision.** Represent Failure Domains as explicit typed nodes with evidence-bearing dependency edges. Initial types are durable medium, enclosure/pool, host, service/cluster, account, and site. Every managed placement must have a known durable-medium domain or remains independence-unknown. Providers and storage instances may depend on multiple domains; Provider identity is never a domain.

For unconstrained **Keep two copies**, the MVP default proof level is distinct durable-medium domains with no known shared enclosure/pool that can destroy both. Shared host, service, account, and site risks are reported and may become explicit Contract constraints, but do not silently masquerade as durable-medium independence. Unknown durable-medium or enclosure relationships fail closed: the Contract remains unfulfilled.

Windows filesystem enrollment derives volume and physical-disk evidence where APIs permit and otherwise asks for confirmation. S3 enrollment cannot infer server backing media from an endpoint; it requires the user or deployment configuration to establish the relevant domain. The local test topology explicitly maps filesystem and SeaweedFS Providers to the same domain when they share a disk.

**Why.** The chosen default directly detects the product owner’s same-physical-disk example while avoiding the impossible claim that two copies share no higher-order risk at all. Typed higher domains keep stronger future Contracts possible without changing copy identity.

**Review trigger.** Changing the default proof level changes visible protection semantics and therefore returns to the product owner.

## ADR-012: Loopback session and browser security

**Decision.** Bind only `127.0.0.1` on an operating-system-assigned port. Generate a 256-bit per-launch bootstrap secret and open the default browser to a fragment-bearing bootstrap page; same-origin JavaScript exchanges the fragment once for a random session cookie so the secret is not sent in the URL request. The cookie is HttpOnly, SameSite=Strict, Path `/`, host-only, and expires after 30 idle minutes or process restart.

Mutations require POST, an exact allowed Host and Origin, a per-session CSRF token, and an expected content type. Reject wildcard hosts, proxy forwarding headers, cross-origin fetch metadata, oversized headers/bodies, and unauthenticated WebSockets. Set a restrictive Content Security Policy, frame denial, no-sniff, referrer policy, request deadlines, response limits, and escaped untrusted names. GET and HEAD are read-only.

Provider secrets are stored in Windows Credential Manager under `sampo` targets and are never written to the home catalogue, `.sampo`, logs, HTML, or Content metadata. Microsoft recommends Credential Manager for per-user credentials. [Windows credential guidance](https://learn.microsoft.com/en-us/windows/win32/secbp/handling-passwords), [Go `net/http`](https://pkg.go.dev/net/http), [Go `crypto/rand`](https://pkg.go.dev/crypto/rand).

**Why.** Loopback reduces exposure but does not prevent DNS rebinding, cross-site requests, malicious local processes, or secret leakage. The controls form one boundary; the bootstrap token alone is not treated as permanent authentication.

## ADR-013: Initial Windows host implementation

**Decision.** SAMPO is a per-user application, not a Windows service, and runs without elevation. Store the home catalogue, configuration, logs, backups, and runtime state beneath the current user’s `FOLDERID_LocalAppData\sampo` directories; store secrets only in Credential Manager. Use a per-user named mutex for single-instance enforcement and launch the local UI with `ShellExecuteW`.

Use `ReadDirectoryChangesW` through a Windows host adapter for recursive notification hints, plus scans for truth. A zero-byte overflow result, unsupported filesystem, watcher restart, or disconnect schedules reconciliation rather than inventing absence. Use handle-based `FILE_ID_INFO` where supported and fall back conservatively. Provider reachability polling handles removable-media return; no device event is correctness-critical.

Development uses an ordinary console binary. The MVP release is a signed per-user installer with an unprivileged Start-menu shortcut and clean uninstall that leaves home backups unless the user explicitly chooses to remove application data. There is no self-updater in MVP; a newer signed installer performs upgrade and database migration with a verified backup first.

**Why.** Microsoft documents `FOLDERID_LocalAppData` as per-user local application data, recommends the Known Folder API for new code, and documents that directory-change buffers can overflow and discard all events. [Windows known folders](https://learn.microsoft.com/en-us/windows/win32/shell/known-folders), [ReadDirectoryChangesW overflow behavior](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-readdirectorychangesw), [FILE_ID_INFO](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-file_id_info), [Go Windows system calls](https://pkg.go.dev/golang.org/x/sys/windows).

**Portability.** Domain, Seshat, Tuoni, Boatman, Caretaker, Gateway, and S3 behavior cannot import Windows packages. Host filesystem identity, watching, known folders, browser launch, credentials, mutex, packaging, and power hints sit behind Windows-only adapters.

## ADR-014: Provider-ledger history and recovery

**Decision.** A writable Provider ledger is a separate SQLite database at `.sampo/ledger.sqlite3`. It retains append-only, hash-chained records for Provider identity evidence, SAMPO-created staging and committed Appearances, complete digest/length, custody provenance, verification results, Provider-relevant Job outcomes, retirement evidence, and references/snapshots sufficient to explain relevant Contract authority. It does not mirror the whole home catalogue or record ordinary user-file scan churn.

Each record has a monotonically increasing local sequence, previous-record digest, event time plus insertion time, producer installation ID, and payload version. The ledger never establishes current availability or sole destructive authority. MVP performs no automatic history compaction. Later compaction requires a signed-off recovery test and retains a hash-linked checkpoint plus all live managed provenance.

On ledger corruption or loss, preserve the damaged file, start no destructive work, rescan bytes, and reconstruct only facts proven by home records, Job locators, complete hashes, and current Provider evidence. Unproven custody becomes user-owned. On home-catalogue loss, Provider ledgers can restore managed provenance and byte identity, but not silently recreate Contracts or deletion authority.

Projects, complete Contract intent/amendment history, UI preferences, and global audit chronology are home-catalogue state. Provider ledgers retain only Provider-relevant Contract snapshots and references; these explain why a copy was created but do not reactivate a lost Contract. After home loss, recovered managed Appearances are preserved and reported, and the user must review and reapprove any ongoing protection obligation.

**Why.** Independent, narrow, append-only evidence improves portable recovery while respecting that Provider reality outranks metadata. Hash chaining detects missing/reordered history but is not treated as cryptographic authorship or distributed consensus.

## ADR-015: Observation, scan, and reconciliation scheduling

**Decision.** Notifications only dirty a scope. Caretaker coalesces events for two seconds and schedules durable scan work; Observer enumerates and reports raw Observations; Seshat alone reconciles them. Run a complete metadata scan at enrollment, application start, Provider reconnect, notification overflow, and at least every 24 hours while the Provider is available. A user can request one immediately.

Hash only new or materially changed candidates, using stable-hash ADR-004. Default to one sequential hashing/transfer reader per durable-medium Failure Domain, two Provider metadata operations per Provider, and two external-work Jobs overall. Interactive Open operations outrank background hashing. Back off exponentially on unavailable/limited Providers, pause low-priority work on critical battery, and expose pause/resume without converting paused work to failure.

Reconciliation uses scan generations: negative absence becomes `confirmed missing` only after a successful complete enumeration of the relevant scope with a stable Provider identity and consistency settling rules. Watcher deletes, timeouts, partial scans, S3 list gaps, and disconnects remain unknown/unavailable.

**Why.** This bounds disk/network contention and makes missed/duplicated/coalesced watcher events harmless. The Windows API explicitly permits buffer overflow with lost events, so periodic scans remain correctness-critical.

## ADR-016: Durable Job execution, leases, retries, and cancellation

**Decision.** Create and commit a Job before external work. Its idempotency key is derived from action kind, Contract or explicit command, Content identity, source generation, destination Provider, planned immutable locator, and Plan revision. Seshat atomically claims eligible Jobs to a random process-instance owner with a 60-second lease; Boatman heartbeats every 15 seconds. A Provider/Failure-Domain concurrency gate is acquired after claim.

On lease expiry, another worker revalidates every precondition and resumes from durable stage evidence; it never assumes the previous worker stopped. Provider publication uniqueness prevents duplicate commit. Retry transient failures with full-jitter exponential delay from one second to five minutes. After ten failed attempts or any ambiguous safety result, move to `needs attention`; unlimited manual retry remains possible after revalidation.

Cancellation is cooperative between chunks and state transitions. Before publication, stop and retain or clean only proven SAMPO staging. During ambiguous publication, reconcile first. After Provider commit, verify and record the result; cancellation never deletes the newly committed Appearance. The UI reports “completed before cancellation took effect” when appropriate.

Stale Plans, changed source generation, revoked Contract authority, changed Failure Domain evidence, capability downgrade, collision, and insufficient space are replan/attention results, not blind retries.

**Why.** Database claiming and leases make work observable and recoverable; immutable destinations and idempotency make duplicate workers safe. Lease expiry is permission to investigate, not proof the old worker died.

## ADR-017: Adoption implementation

**Decision.** Observer creates a durable adoption candidate for any unproven file beneath `library/`, keyed by Provider, locator, and observed generation. Seshat records user custody. Gateway offers exactly **Adopt**, **Leave it mine**, or **Ask later**.

Adopt first computes a stable complete digest, shows whether an active Contract already covers that Content and whether adoption would make the Appearance count, then records explicit approval. Immediately before the custody transaction, revalidate Provider identity, locator generation, digest, and Contract revision. A change invalidates approval and leaves the file user-owned. With no Contract, adoption creates managed custody but no implicit protection obligation. If more than one Contract could apply, adoption may associate all exact-Content Contracts whose existing terms already permit that Provider; expanding authority requires a new Plan.

Leave suppresses the prompt for that exact observed generation. A later byte change creates a new candidate. Ask later retains a visible pending candidate without repeated interruption. All outcomes are audited. Adoption never moves, renames, wraps, or rewrites the bytes.

**Why.** This makes custody transfer explicit and race-safe while avoiding the false implication that a managed directory owns everything inside it.

## ADR-018: Factual Provider usage reporting

**Decision.** Instrument SAMPO at Provider-adapter boundaries. Record logical payload bytes read/written, full-verification bytes, retry bytes, request/operation counts known to SAMPO, operation result, Provider, Job, start/end time, and whether a counter is client-measured or Provider-reported. For filesystems also report known committed managed bytes. For S3, expose quota, throttling, authorization, and billing-related error responses factually.

Aggregate only over explicit time ranges and label unknown/missing counters. Do not multiply operations or bytes by price tables, predict a bill, claim to include Provider-side replication/metadata/taxes, enforce a budget, or block an already authorized Contract because of a SAMPO spending estimate. Enrollment of any paid Provider displays the Provider-side budget/alert warning once and keeps it available in settings.

**Why.** Client instrumentation is auditable and useful for diagnosis, while a complete cloud bill depends on terms and Provider-side activity SAMPO cannot know. This preserves factual observability without inventing monetary authority.

## ADR-019: Operation routing and provider-native retrieval

**Decision.** Tuoni first filters by correctness: exact verified Content, compatible operation, current availability, stable Provider identity, and required custody/permission. It then ranks Open candidates by user-pinned preference, user-owned local filesystem, managed local filesystem, local-network Provider, and remote object Provider; within a class it uses recent measured success and latency, then a stable ID tie-breaker. Unknown cost never outranks known locality. A failed candidate falls through only after its state is recorded; SAMPO does not hide the selected Appearance.

Edit always opens a user-owned filesystem Appearance. If none is eligible, SAMPO offers an approved Job to materialize one; it never edits a managed Appearance in place. Provider-oriented actions use the explicitly selected Provider rather than the general Open ranking.

For filesystem Appearances, Gateway shows and can open the ordinary absolute path through the default application. For S3-compatible Appearances, it shows endpoint, bucket, key, version/generation when available, complete SHA-256, and copyable AWS-CLI-compatible retrieval guidance without embedding secrets. While SAMPO runs it may stream an authenticated download, but provider-native tools and credentials remain the uninstall/recovery path. Readable S3 keys and raw object bytes require no SAMPO decoder.

**Why.** A fixed eligibility-first order makes the “local M.2 beats the Cambodian wirehanger” rule predictable without making one Appearance canonical. Showing the chosen source and native locator keeps routing explainable.

## ADR-020: Managed retirement and Provider deletion results

**Decision.** Retirement requires the already approved Contract amendment/cancellation Job. Immediately before acting, Seshat proves managed custody provenance, current exact identity, no active Contract dependency, stable Provider identity, and exact planned locator/generation.

On filesystems, atomically rename the exact Appearance without replacement into `.sampo/retirement/<job-id>/` on the same volume, record the staged-retirement fact, then delete only that proven SAMPO-owned staged file. A crash before deletion is recoverable and cannot expose a partial replacement. SAMPO reports ordinary deletion, never secure erasure.

On S3, delete the exact version ID when native versioning is present. Without a version ID, require a Provider-tested conditional delete against the revalidated generation/ETag; otherwise automated retirement is unsupported and needs attention. A delete marker, retention lock, delayed visibility, or eventual-consistency response is reported accurately. Completion means the Provider contract has positively established logical removal of the targeted accessible generation after its settling rule, not that every physical byte has been erased.

Cancellation before deletion preserves or restores the staged file. Once deletion is positively committed, cancellation cannot resurrect bytes and the audit says so. No generic cleanup process may operate outside Job-owned staging, retirement, or multipart records.

**Why.** Staged filesystem retirement narrows the last irreversible step and makes crash recovery observable. Version/generation-specific object deletion prevents a stale Plan from deleting changed external data.

## ADR-021: Directly witnessed fork lineage

**Decision.** Retain minimal lineage only when SAMPO directly witnesses a formerly verified managed Appearance change from Content A to stably hashed Content B. Record an immutable `observed-edit-of` relationship from B’s new user-owned Appearance to the prior Appearance/Event, with time and evidence. Do not infer ordered versions from filenames, timestamps, similarity, or later scans, and do not expose this relation as synchronization or a canonical history.

**Why.** The fact helps explain custody transfer and Contract repair without inventing version-control semantics. It is optional descriptive evidence and never affects exact-content identity or authority.

## Decision coverage

| Ownership delegation | ADR |
|---|---|
| Language and application stack | ADR-001 |
| Persistence | ADR-002 |
| Digest strategy | ADR-003 |
| Mutable-file hashing consistency | ADR-004 |
| Local S3-compatible development service | ADR-005 |
| Filesystem staging and atomic publication | ADR-006 |
| S3 staging and publication | ADR-007 |
| Managed-copy layout mechanics | ADR-008 |
| Provider identity | ADR-009 |
| Probable-rename confidence | ADR-010 |
| Failure-domain representation | ADR-011 |
| Session and browser security | ADR-012 |
| Initial Windows implementation | ADR-013 |
| Provider-ledger history and recovery | ADR-014 |
| Scan and reconciliation scheduling | ADR-015 |
| Job execution details | ADR-016 |
| Adoption implementation | ADR-017 |
| Factual usage reporting | ADR-018 |

Residual architecture questions are covered by ADR-019 through ADR-021: access routing and native retrieval, managed retirement semantics, and minimal directly witnessed fork lineage.

## Revisit rule

An implementation ADR may be revised without product-owner escalation only when the replacement preserves user-visible behavior, Contract meaning, custody, recoverability, Provider compatibility, and safety evidence. Any revision that changes those boundaries follows the escalation rules in `SAMPO-IMPLEMENTATION-ADR-OWNERSHIP.md`.
