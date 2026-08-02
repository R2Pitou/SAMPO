# MAS-H MVP Architecture Decision Record and Codex Handoff

**Status:** Product-owner approved  
**Date:** 2026-08-02  
**Authority:** This document records the product-owner decisions made after the MAS-H Great Reset architecture audit. Where it conflicts with provisional material in `Spec.md` or stale implementation material in `README.md`, this decision record wins. `MANIFESTO.md` remains constitutional.

---

# Part I: Codex instruction

You are working in the MAS-H repository after a complete implementation reset.

The previous implementation was deleted because it encoded unsafe and contradictory assumptions. Only the surviving product documents and the architecture audit should be considered historical inputs. Do not recover, imitate, or preserve the deleted implementation.

Read, in this order:

1. `MANIFESTO.md`
2. `Spec.md`
3. `README.md`
4. the Sol architecture audit, if it exists in the repository
5. this decision record

Authority order:

1. `MANIFESTO.md` is constitutional.
2. This product-owner decision record resolves the architecture audit’s open questions.
3. `Spec.md` is vision and vocabulary, but must be corrected where this record supersedes it.
4. `README.md` is explanatory and currently contains stale implementation debris.
5. The architecture audit is analysis and advice, not product authority.

## Your task

Perform a documentation and architecture pass only.

Do not write production code.  
Do not create database migrations.  
Do not scaffold packages or services.  
Do not select a programming language merely because the deleted prototype used one.  
Do not recreate the deleted Go implementation.  
Do not introduce microservices, a message broker, a virtual filesystem, SMB, WebDAV, sync, RAID behavior, or a Git-like revision engine.

Create or update the repository documentation so that a competent implementation team can build the approved MVP without guessing.

Required deliverables:

1. Create `MVP-ARCHITECTURE-DECISIONS.md` containing the approved decisions in Part II of this document.
2. Create `MVP-ARCHITECTURE.md` containing:
   - system boundaries;
   - Staff responsibilities and authority limits;
   - the minimum domain model;
   - provider capability boundaries;
   - contract lifecycle;
   - observation and reconciliation flow;
   - copy creation and verification flow;
   - managed-copy deletion and external-edit flows;
   - provider enrollment flow;
   - access-routing flow;
   - security boundary;
   - failure handling;
   - implementation milestones;
   - acceptance-test mapping.
3. Update `MANIFESTO.md` only where instructed below, preserving its existing laws and tone.
4. Rewrite `README.md` as a truthful project entry point. Remove all stale build instructions, endpoints, ports, code claims, test claims, and links to missing documents.
5. Update `Spec.md` so that approved MVP decisions are not contradicted by provisional language. Preserve larger future ambitions, but label deferred features clearly.
6. Create `MVP-ACCEPTANCE-TESTS.md` from Part V.
7. Create `PARKING-LOT.md` from Part VI.
8. Create `IMPLEMENTATION-MILESTONES.md` with small vertical milestones. Each milestone must produce a demonstrable capability and preserve all prior safety tests.
9. Produce a final change report listing:
   - files created;
   - files changed;
   - contradictions removed;
   - unresolved questions that genuinely remain;
   - any place where the approved decisions are still technically ambiguous.

## Required architecture quality

The architecture must be concrete enough to implement, but it must not pretend unsettled implementation choices are product decisions.

For every major component, state:

- what it owns;
- what it may read;
- what it may change;
- what it may never do;
- its inputs and outputs;
- failure behavior;
- which decisions require user approval;
- which operations are authorized by an existing contract.

Separate these concepts explicitly:

- raw observation;
- reconciled catalogue fact;
- user intent;
- protection contract;
- plan;
- approved job;
- staged copy;
- verified managed copy;
- user-owned appearance;
- managed appearance;
- provider availability;
- contract fulfillability.

Do not use path, drive letter, bucket key, or provider identity as logical file identity.

Do not make metadata a requirement for opening ordinary files.

Do not call a copy durable, protected, or contract-satisfying until MAS-H has independently verified the completed bytes.

Do not infer destructive authority from directory location.

When an approved decision below conflicts with the audit’s recommended default, follow the approved decision.

---

# Part II: Fourteen approved decision areas

## Decision 1: Accessibility, custody, and the First Law

The operational interpretation of the First Law for the MVP is:

> MAS-H may discover, read, hash, catalogue, search, observe, and create additional ordinary copies. It may not destructively alter user-originated data.

### User-owned appearances

A pre-existing file or a file created outside MAS-H is user-owned.

MAS-H may:

- read it;
- hash it;
- catalogue it;
- observe it;
- search it;
- copy from it;
- associate non-destructive catalogue metadata with it.

MAS-H may not:

- overwrite it;
- truncate it;
- rename it;
- move it;
- delete it;
- replace it;
- silently adopt it;
- infer ownership merely because it sits inside a MAS-H-looking directory.

### MAS-H-managed appearances

A copy is MAS-H-managed only when:

- MAS-H created it through a recorded, approved job; or
- the user explicitly adopts it into MAS-H custody.

MAS-H may reorganize, replace, recreate, move, or retire managed appearances only within the authority granted by active contracts and provider permissions.

### Custody safety rule

> Custody is established by provenance or explicit adoption, never by path.

When custody cannot be proven, MAS-H must treat the file as user-owned and protected from destructive action.

Removing MAS-H must leave all committed files readable through ordinary provider-native tools.

---

## Decision 2: Exact content identity, appearances, and rename handling

A complete cryptographic content hash ignores:

- filename;
- path;
- drive letter;
- provider;
- timestamps;
- user naming choices.

### Logical treatment

> Exact verified byte equality means MAS-H treats the data as one logical file content with multiple appearances.

Each appearance separately records:

- provider;
- locator or path;
- current filename;
- observed names and locators where useful;
- user-owned or MAS-H-managed custody;
- availability;
- verification state;
- first and last observation;
- provider-native identity evidence where available.

This allows MAS-H to tell the user:

> This content is already protected. A verified managed copy exists here. You also have user-created copies here, here, here, here, and here.

Repeated requests to protect byte-identical content are idempotent. MAS-H reports the existing contract and known appearances rather than creating an unnecessary copy.

### Rename and move evidence

- A reliable provider-native file identifier or rename event is strong evidence of continuity.
- If an old locator disappears and one new locator with the same complete hash appears, MAS-H may record a probable rename or move.
- If both locators remain, MAS-H records two appearances, not a rename.
- If several identical candidates make the mapping ambiguous, MAS-H records uncertainty rather than inventing certainty.

A hash proves byte equality. It does not prove the exact human action that produced the observed state.

The implementation may use opaque internal identifiers, but logical content equality must not depend on path or provider identity.

---

## Decision 3: External edits create forks, not synchronization

When an appearance changes bytes, it no longer represents the previous content.

MAS-H must:

- recognize the new byte content;
- leave all other appearances unchanged;
- never propagate the edit automatically;
- never merge;
- never overwrite another appearance;
- never call this a sync conflict.

The changed content is a fork from the previous content in the ordinary human sense: it was derived from it but is now different.

For the MVP, do not build a Git-like revision graph, merge engine, commit model, or automatic ancestry inference. A minimal directly observed `derived-from` fact may be stored when MAS-H actually witnessed the transition, but the MVP must not depend on a full history graph.

---

## Decision 4: Protection contracts persist independently of current fulfillability

A protection request is a contract over exact content.

Example:

> Keep two MAS-H-managed copies of this content.

User-owned originals and duplicates do not count as MAS-H-managed copies unless explicitly adopted.

A contract has at least these states:

### Fulfilled

The required number of verified managed copies currently exists.

### Unfulfilled but fulfillable

The contract is not currently satisfied, but MAS-H can access the exact content and has an approved eligible destination.

### Unfulfillable

The contract remains active, but MAS-H cannot currently access any copy of the exact required bytes or cannot meet the approved terms.

Reality may make a contract unfulfillable. Reality does not cancel it.

Only the user may amend or cancel the contract.

If the exact content later reappears, MAS-H recognizes it and may fulfill the existing contract within its approved terms.

MAS-H must never reinterpret “similar,” “edited,” or “probably the same” bytes as satisfying a contract for an exact content hash.

---

## Decision 5: Home catalogue and provider-local `.mas-h` memory

MAS-H uses layered metadata.

### Home Seshat catalogue

Each installation has a home catalogue containing the complete view known to that installation, including:

- providers;
- content identities;
- appearances;
- custody;
- contracts;
- jobs;
- verification state;
- project membership;
- observations;
- activity and audit history.

### Provider-local control area

Where applicable, an enrolled writable provider may contain one hidden control area:

```text
.mas-h/
```

It should contain one compact provider-local database or equivalent ledger, not one sidecar per user file.

The provider-local ledger may store:

- stable provider identity;
- schema version;
- provider-local appearance inventory;
- content hashes;
- custody evidence;
- MAS-H-created copy records;
- scan checkpoints;
- previous observations;
- node or installation attribution;
- compact change history sufficient to compare reconnects.

A drive may travel between two separate MAS-H installations. Each installation should be able to recognize the provider, compare the provider-local memory with its own home catalogue and the provider’s current contents, and record what changed between connections.

This is portable provenance and memory, not distributed consensus.

### Metadata safety

- No MAS-H metadata may be required to open committed files.
- Loss or corruption of `.mas-h` must leave ordinary data readable.
- Loss of the home catalogue must leave ordinary data readable.
- Rebuilding may recover only what surviving evidence can prove.
- If metadata loss makes custody uncertain, MAS-H loses destructive authority and treats affected files as user-owned.
- Per-file sidecars are not the default. They may later be optional export or archival aids.
- Provider-native metadata may later be used as redundant evidence, never as the sole byte-access dependency.

Creating `.mas-h` on user-originated media requires explicit enrollment.

---

## Decision 6: Unknown-provider prompt and enrollment

When MAS-H sees a storage medium it has not seen before, the user-facing prompt is:

> I see you connected a drive I haven’t seen before. Would you like this to become part of MAS-H?

Choices:

### Yes

Open a setup wizard.

The wizard may ask whether MAS-H may:

- catalogue existing files;
- create and maintain `.mas-h`;
- use the provider as a destination for managed copies.

These are separate permissions. Joining the library does not silently grant destructive authority over existing files.

### No

Add the provider to an excluded list and stop asking for that provider.

### Ask me next time

Do nothing now and ask again when it reconnects.

Provider recognition must use the best available persistent identity evidence, not the current drive letter alone.

Unknown media must not be modified before the user chooses Yes and grants the relevant permission.

---

## Decision 7: No canonical copy; operation-specific access routing

The MVP has no canonical-copy concept.

All verified byte-identical appearances contain the same content. None is metaphysically more true than the others.

MAS-H may choose the best eligible appearance for the requested operation.

### Read or open

Prefer, in broad order:

1. verified available local fast storage;
2. other verified local storage;
3. local-network storage;
4. remote object or cloud storage;
5. unavailable appearances shown as unavailable rather than attempted blindly.

Thus MAS-H serves the M.2 copy instead of fetching the same bytes from Dropbox over a Cambodian wirehanger.

### Edit

Prefer a writable, user-owned local appearance.

Never open a managed preservation copy as the default editable target.

If only a managed copy is available, create a user-owned working copy through an explicit or clearly explained action, then open that.

### Share or remote access

A remote appearance may be preferable when the requested operation is sharing, remote access, or provider-specific delivery.

### Rule

> Preferred access is temporary routing for one operation. It is not authority, ownership, synchronization, or canonical status.

The user normally sees one logical file. Detailed appearance information remains available without being inflicted on every interaction.

---

## Decision 8: Projects are overlapping logical collections with snapshot policies in the MVP

A project is a Seshat collection, not a directory and not a provider location.

The same logical content may belong to multiple projects.

Adding or removing project membership changes catalogue metadata only. It does not move, copy, rename, or delete bytes.

For the MVP, a project-level action operates on a reviewed snapshot.

Example:

> Protect Sunday Short Mix 5.

MAS-H shows the current unique file count and asks for approval. Approval creates file-level contracts for the current members.

Files added later do not silently inherit the earlier project action.

The project may quietly report:

> Three files were added after the last protection snapshot.

The user may then protect those additions in a batch.

Deferred:

- living project contracts;
- automatic policy inheritance;
- retention windows for temporary drafts;
- automatic garbage detection;
- project-wide fork management.

---

## Decision 9: First complete workflow and local browser interface

The first release must prove one complete vertical journey:

1. The user connects an unknown drive.
2. MAS-H shows the three-choice provider prompt.
3. The user chooses Yes and completes setup.
4. MAS-H discovers and hashes files without changing them.
5. The user searches by name.
6. MAS-H shows one logical file with its known appearances.
7. The user selects `Keep a copy`.
8. MAS-H shows a concise plan with destination and required space.
9. The user approves.
10. Boatman creates a new staged copy.
11. MAS-H independently verifies the completed bytes.
12. Only then does the copy count as managed and contract-satisfying.
13. The original provider becomes unavailable.
14. The user searches again.
15. MAS-H opens the best available managed appearance.

### Interface

The MVP user interface is a local browser application served by the MAS-H process.

It is:

- search-first;
- local-only;
- visually simple;
- not a public website;
- not a native .NET requirement;
- not an Electron requirement;
- not a command-line-only product;
- not SMB, WebDAV, or a virtual drive.

The home screen should prioritize:

- library search;
- provider availability;
- contract status requiring attention;
- recent activity.

It must not greet the user with a storage-administrator dashboard.

---

## Decision 10: Provider proof is mounted filesystems plus S3-compatible object storage

The MVP supports two genuinely different provider classes.

### Mounted hierarchical filesystem provider

Covers ordinary mounted storage such as:

- internal drives;
- external drives;
- USB media;
- mounted network shares where the operating system presents ordinary filesystem semantics.

Capabilities may include:

- enumerate;
- read;
- create;
- rename;
- remove managed files;
- native file identity;
- directory semantics;
- local availability and performance evidence.

Individual providers may advertise different capability subsets.

### S3-compatible object-storage provider

Used as actual object storage, not mounted as a fake drive.

It proves that MAS-H understands:

- keys rather than true directories;
- different commit semantics;
- remote latency;
- lack of inode-style identity;
- upload staging;
- provider-specific integrity evidence;
- transfer cost and availability.

Development must work locally against a disposable S3-compatible service, ideally containerized, without requiring a public cloud account or credit card.

The architecture must not depend on one particular local S3 implementation unless a separate build-versus-integrate decision selects it.

### Byte integrity

MAS-H must not treat an S3 ETag or a successful upload response alone as proof of exact content equality.

A copy becomes contract-satisfying only after verification grounded in MAS-H’s own complete content digest or an equivalently strong independently validated method.

Both provider classes must store ordinary original bytes without proprietary wrapping as the only representation.

Dropbox, Google Drive, R2, Git, GitHub, and other providers are deferred.

---

## Decision 11: First-release security boundary

The MVP is:

- single-user;
- single-node;
- one local installation;
- local-only by default.

The local web Gateway must bind to loopback only.

The MVP has no:

- MAS-H user accounts;
- roles;
- shared libraries;
- LAN control plane;
- public listener;
- remote administration;
- multi-principal ACL mapping.

MAS-H accesses providers with the permissions of the operating-system user running it.

Local-only does not mean any web page may command it. The browser UI must still have appropriate local session, origin, request-forgery, body-size, and timeout protections.

Credentials must not be embedded in ordinary user files or provider-visible per-file metadata.

Remote and multi-user operation are later explicit modes, not accidental defaults.

---

## Decision 12: Contract approval authorizes continuous maintenance within approved terms

A direct user request such as:

> Keep two managed copies.

produces a concise plan.

One approval creates the protection contract and authorizes the bounded work required to maintain it.

Within the approved terms, Boatman may without repeated confirmation:

- stage the copy;
- retry interrupted work;
- resume an incomplete transfer;
- verify completion;
- clean up MAS-H-owned staging material;
- recreate a managed copy that disappeared;
- restore the required managed-copy count;
- record and report the action.

A materially changed plan requires new approval.

Examples of material change:

- using a new provider;
- incurring an unapproved cost;
- changing the required number of copies;
- changing durability or location constraints;
- touching user-owned data;
- expanding the contract to unrelated content.

Background observations may suggest new contracts, but may not create them automatically.

### External deletion

If the user externally deletes a managed copy required by an active contract, MAS-H treats this as loss, not contract cancellation.

When exact source bytes remain accessible and the approved terms can still be met, Boatman silently recreates and verifies the missing managed copy.

The activity log may say:

> A managed copy required by your contract disappeared. Boatman created and verified a replacement.

### External edit of a managed copy

If a managed copy is edited outside MAS-H:

1. MAS-H must not overwrite or revert the edited file.
2. The edited appearance immediately becomes user-owned.
3. It no longer satisfies the contract for the old exact content.
4. If the old bytes remain accessible elsewhere, Boatman may recreate the required managed copy within the existing contract.
5. If the old bytes do not remain accessible, the contract becomes unfulfillable but remains active.

This is the real-world library rule:

> You scribbled in the Librarian’s copy. That copy is yours now.

---

## Decision 13: Deleting managed data means amending the contract

Deleting one managed appearance and reducing or cancelling a contract are different actions.

### Delete this managed appearance

If the appearance is required by an active contract, MAS-H explains that Boatman will replace it.

The user may choose a different eligible destination, amend the contract, or cancel the action.

### Reduce the contract

Example:

> Change from two managed copies to one.

MAS-H previews:

- which managed copy will remain;
- which managed copy will be retired;
- that user-owned appearances will not be changed.

After approval, Boatman may retire managed copies that are no longer required by any active contract.

### Cancel protection

MAS-H asks what should happen to its managed copies:

- hand them to the user, transferring custody;
- remove eligible MAS-H-managed copies;
- cancel nothing.

Contract amendment grants destructive authority only over MAS-H-managed appearances that are no longer required by any active contract.

User-owned appearances are never silently included.

---

## Decision 14: SAMPO and naming law

### Product naming

- Human-facing product name: `MAS-H`
- Expansion: `Memory Abstraction Storage Hypervisor`
- Spoken name: “mash”
- Repository, executable, package, and command identifier: `mash`
- Provider metadata directory: `.mas-h`
- Avoid inconsistent mutations such as `MASH`, `Mas-h`, `mas_h`, or `MaSH`

### SAMPO

SAMPO is the internal orchestration engine of MAS-H.

Retroactive expansion:

> Storage Abstraction Management & Policy Orchestrator

SAMPO is:

- not a separate product;
- not necessarily a separate process;
- not a required service boundary;
- not a message broker;
- not a microservice platform;
- not the owner of user bytes.

For the MVP, MAS-H and all Staff responsibilities run inside one local application with clear module and authority boundaries.

The Staff are responsibilities:

- **Gateway** talks to the local browser and translates user actions into bounded requests.
- **Observer** notices the outside world and emits untrusted observations. It never acts.
- **Seshat** maintains reconciled catalogue knowledge and contract state.
- **Tuoni** turns intent, contracts, current facts, and provider capabilities into explainable plans. It performs no storage I/O.
- **Boatman** executes approved jobs. It never invents policy.
- **Caretaker** checks existing contracts and safe maintenance needs. It may trigger only work already authorized by contracts or present suggestions requiring approval.

Staff may use direct queries and commands inside the process. Durable domain events record committed facts. “Event driven” does not require every function call to become an asynchronous event.

### Required Manifesto text

Add the following section to `MANIFESTO.md`, preserving the exact quote:

```markdown
## SAMPO

SAMPO came first.

The name existed before the architecture, before MAS-H, and before anyone knew what the letters were supposed to mean.

Later, it was retroactively expanded into:

**Storage Abstraction Management & Policy Orchestrator**

> **What exactly is SAMPO?**  
> “Your guess is as good as mine. I was high. I liked the name.”
>
> — Arttu Pitou, Founder
```

The README may use the respectable summary:

> SAMPO, retroactively expanded as the Storage Abstraction Management & Policy Orchestrator, is MAS-H’s internal orchestration engine.

It should point readers to the Manifesto for the historically accurate explanation.

---

# Part III: Minimum domain model

Do not overbuild this. The MVP needs enough structure to support the approved workflows and no more.

## Provider

Represents one registered storage resource.

Minimum concerns:

- stable internal provider ID;
- provider class;
- best-effort persistent identity evidence;
- enrollment status;
- excluded or ask-next-time state;
- capabilities;
- availability;
- permission to host `.mas-h`;
- permission to host managed copies;
- local or remote classification;
- performance and cost hints sufficient for access routing.

Provider identity is not content identity.

## Content

Represents one exact byte sequence recognized through complete verified digest evidence.

Minimum concerns:

- stable internal ID;
- digest algorithm and digest;
- byte size;
- known type hints;
- verification history.

Do not make the path the ID.

Do not assume the digest must be the database primary key. Algorithm migration must remain possible.

## Appearance

Represents one observed occurrence of Content.

Minimum concerns:

- provider;
- locator;
- current and observed names;
- custody;
- availability;
- verification status;
- first and last observation;
- native identity evidence;
- whether it is staged, committed, missing, changed, or uncertain.

## Project

Represents an overlapping logical collection.

Minimum concerns:

- project ID;
- name;
- membership;
- snapshot action history.

## Protection contract

Represents a persistent user-approved obligation over exact Content.

Minimum concerns:

- target Content;
- required managed-copy count;
- approved provider constraints;
- approved cost or locality constraints where applicable;
- state;
- created and amended history;
- cancellation state.

## Plan

An explainable proposed way to create or amend a contract or execute a materially changed action.

A plan is not authority until approved.

## Job

A durable bounded execution record created from approved authority.

Minimum concerns:

- idempotency;
- preconditions;
- source evidence;
- destination;
- staging state;
- verification state;
- retry state;
- result;
- audit trail.

## Observation

Untrusted evidence from a provider or watcher.

An observation may suggest that a file appeared, disappeared, moved, or changed. It is not authoritative until reconciled.

## Catalogue fact

A reconciled statement Seshat currently accepts, including its evidence and confidence where appropriate.

---

# Part IV: Required corrections to surviving documents

## `README.md`

Remove or rewrite:

- stale Go prerequisites and build commands;
- references to deleted source paths, scripts, configuration files, tests, ports, and REST endpoints;
- links to documents that do not exist;
- claims that SAMPO is a separate reference storage engine;
- claims that all Staff communicate only through SAMPO events;
- current SMB and WebDAV claims;
- distributed deployment claims presented as current implementation;
- automatic migration, cache eviction, deduplication, and semantic indexing presented as current behavior;
- `MASH` spelling.

Add:

- concise product purpose;
- laws summary;
- current MVP status;
- local browser UI;
- filesystem and local S3-compatible provider proof;
- single-user local-only security scope;
- search-first workflow;
- ordinary-file guarantee;
- link to the decision record, architecture, acceptance tests, parking lot, and Manifesto;
- respectable SAMPO definition plus Manifesto pointer.

## `Spec.md`

Correct or qualify:

- `MASH` to `MAS-H`;
- “canonical” language: remove it from MVP semantics;
- “applications believe they use ordinary files” as a future aspiration, not MVP behavior;
- “distributed system” as future deployment potential, not first-release architecture;
- “every Staff member communicates through events” to clear module responsibilities with queries, commands, and durable events;
- automatic maintenance tasks as deferred;
- deduplication, migration, cache eviction, tiering, and sync-like behavior as deferred;
- projects: many-to-many collection membership with snapshot project actions for MVP;
- object model: exact byte content plus appearances and custody;
- provider abstraction: capability-addressable, not capability-equivalent;
- policy model: persistent protection contracts over exact content;
- user interface: local browser search and direct ordinary-file access;
- optional AI remains explicitly non-essential.

## `MANIFESTO.md`

Preserve all existing laws.

Add the exact SAMPO origin section from Decision 14.

Do not turn the Manifesto into an implementation manual.

---

# Part V: MVP acceptance tests

These tests are product behavior requirements. Architecture and milestones must map to them.

## AT-01: MAS-H removal does not trap data

Given pre-existing ordinary files on an enrolled filesystem provider, after every supported MVP operation:

- stop MAS-H;
- remove or ignore MAS-H metadata;
- open the original and committed managed files with ordinary tools.

Pass condition: user data remains readable without MAS-H.

## AT-02: User-owned source remains untouched

Import a provider containing user files.

Pass condition:

- no user file is renamed, moved, overwritten, truncated, deleted, or adopted;
- only explicitly approved `.mas-h` metadata may be added;
- hashes before and after match.

## AT-03: The cement USB

- Enroll a USB.
- Find a file.
- Approve one managed copy.
- Verify the copy.
- Disconnect the USB.

Pass condition:

- search still returns one logical file;
- the USB appearance is unavailable;
- the managed appearance is available;
- opening the file uses the managed appearance;
- MAS-H does not call the absent USB deleted.

## AT-04: Five idiot copies

Place byte-identical files under different names, folders, and providers.

Pass condition:

- search shows one logical content item;
- all appearances are listed;
- user-created and MAS-H-managed appearances are distinguished;
- a repeated `Keep a copy` request reports existing protection and does not create an unnecessary managed copy.

## AT-05: Rename without panic

Rename a user-owned file without changing its bytes.

Pass condition:

- MAS-H does not present it as unrelated new content;
- it records the new appearance locator;
- it uses strong native identity evidence where available;
- otherwise it marks the relationship probable if evidence is unambiguous.

## AT-06: External edit creates different content, not sync

Edit one of several identical user-owned appearances.

Pass condition:

- the edited appearance becomes new content;
- other appearances remain unchanged;
- MAS-H performs no propagation, merge, or rollback;
- no sync-conflict model is invoked.

## AT-07: Scribbled Librarian copy

- Create one managed copy of Content A.
- Externally edit that managed file into Content B.

Pass condition:

- Content B is preserved;
- the edited appearance becomes user-owned;
- MAS-H never reverts or overwrites it;
- it no longer satisfies the contract for A.

## AT-08: Scribbled final copy makes contract unfulfillable

Repeat AT-07 when no other accessible appearance of A exists.

Pass condition:

- the contract for A remains active;
- its state becomes unfulfillable;
- MAS-H retains A’s digest and known history;
- MAS-H does not claim B satisfies A’s contract.

## AT-09: The lost content returns

After AT-08, reconnect a provider containing exact Content A.

Pass condition:

- Seshat recognizes A;
- the existing contract becomes fulfillable;
- Boatman may restore the required managed copy within already approved terms.

## AT-10: Cuntily deleted managed copy

- Create a contract requiring two managed copies.
- Verify two managed copies.
- Externally delete one while an accessible source remains.

Pass condition:

- the contract remains active;
- MAS-H detects the missing managed appearance;
- Boatman creates and verifies a replacement without asking for the same approval again;
- the activity log explains why.

## AT-11: Contract amendment controls permanent deletion

- With two required managed copies, request reduction to one.

Pass condition:

- MAS-H previews the exact managed copy to retire;
- user-owned appearances are excluded;
- after approval, one managed copy is retired;
- one verified managed copy remains;
- no replacement is created because the contract was amended first.

## AT-12: Cambodian wirehanger routing

Make identical verified appearances available on:

- local M.2;
- remote S3-compatible storage.

Pass condition:

- ordinary Open uses the local copy;
- the remote copy remains known;
- no canonical status changes;
- a remote-oriented operation may intentionally choose the remote appearance.

## AT-13: Editing when only a managed copy exists

Make only a managed appearance available and request Edit.

Pass condition:

- MAS-H does not open the managed preservation copy as the default writable target;
- it creates or proposes a user-owned working copy;
- the managed appearance remains unchanged.

## AT-14: Unknown-drive prompt

Connect an unknown provider and test:

- Yes;
- No;
- Ask me next time.

Pass condition:

- Yes opens setup;
- No persists exclusion;
- Ask me next time performs no enrollment and prompts on reconnect;
- MAS-H writes nothing before permission.

## AT-15: Provider-local memory travels

- Enroll a writable drive on MAS-H installation A.
- Disconnect it.
- Connect it to installation B.
- Change or add ordinary files.
- Return it to A.

Pass condition:

- both systems recognize the provider identity where evidence permits;
- each compares home catalogue, provider ledger, and actual contents;
- no ledger claim is trusted blindly;
- no distributed-consensus mechanism is required;
- ambiguous custody results in hands-off behavior.

## AT-16: Provider ledger loss

Delete or corrupt `.mas-h`.

Pass condition:

- ordinary files remain readable;
- MAS-H can rescan;
- uncertain custody becomes user-owned/protected;
- no destructive operation is authorized by missing metadata.

## AT-17: Home catalogue loss

Remove the home catalogue while leaving providers intact.

Pass condition:

- ordinary files remain readable;
- rebuild recovers only evidence that can be proven;
- MAS-H does not claim full history or custody it cannot prove.

## AT-18: Project snapshot

- Create a project with 37 unique contents.
- Approve project protection.
- Add three new contents.

Pass condition:

- contracts exist for the original 37;
- the three additions do not silently inherit protection;
- the interface quietly reports additions since the snapshot;
- the user may protect them as a batch.

## AT-19: Interrupted filesystem copy

Kill Boatman during transfer.

Pass condition:

- partial staging does not count as a managed copy;
- no existing user file is truncated or replaced;
- the job can safely retry or resume;
- completion is reported only after full verification.

## AT-20: Interrupted S3 upload

Kill the local S3-compatible service during upload.

Pass condition:

- the contract is not marked fulfilled;
- partial or multipart state does not count as a managed copy;
- retry is idempotent;
- ETag alone is not accepted as MAS-H verification.

## AT-21: Local-only security

Pass condition:

- Gateway binds only to loopback by default;
- LAN clients cannot access the control plane;
- state-changing requests require the local UI session protections selected by the architecture;
- request limits and timeouts exist.

## AT-22: No metadata byte dependency

For every committed filesystem and S3-compatible object copy:

- remove MAS-H processes and metadata;
- access the bytes using provider-native tools.

Pass condition: the content is available without decoding through MAS-H.

---

# Part VI: Explicit parking lot

These are not rejected forever. They are excluded from the MVP so they cannot sneak into implementation through “helpful” assumptions.

## Storage behavior

- automatic migration;
- source retirement;
- cache eviction;
- automatic hot/cold tiering;
- storage deduplication that removes appearances;
- RAID-like redundancy not created by an explicit contract;
- block-level virtualization;
- proprietary containers;
- automatic deletion of user-owned data.

## Sync and version control

- Dropbox-style synchronization;
- write-through propagation;
- conflict resolution;
- automatic merges;
- Git-like commit and revision DAG;
- inferred ancestry beyond minimal directly observed facts;
- automatic reverts;
- lineage-wide protection policies.

## Interfaces

- SMB;
- WebDAV;
- virtual drives;
- filesystem overlays;
- Explorer/Finder shell extensions;
- public API;
- remote administration;
- native .NET GUI;
- Electron packaging.

## Distribution and identity

- multi-node control plane;
- distributed catalogue writes;
- consensus;
- simultaneous multi-system provider mutation;
- multi-user accounts;
- roles and shared libraries;
- provider ACL mapping;
- cloned-provider identity resolution beyond safe detection and user review.

## Providers

- Dropbox;
- Google Drive;
- R2;
- public S3 accounts;
- Git and GitHub as providers;
- another MAS-H node as a provider;
- cloud billing optimization.

## Intelligence and search

- semantic search;
- AI-required classification;
- AI authority over correctness, deletion, custody, identity, or contract safety;
- generic knowledge graphs;
- arbitrary relationship ontology.

## Projects and automation

- living project contracts;
- automatic inheritance;
- automatic draft cleanup;
- automatic garbage detection;
- interruption for every new file;
- opportunistic resource scheduling beyond what the approved contract directly requires.

## Metadata

- one sidecar per file as the default;
- reliance on extended attributes or alternate streams as sole identity evidence;
- full catalogue reconstruction claims when evidence is absent.

---

# Part VII: Suggested implementation milestones

Codex must refine these into architecture milestones, not production code in this pass.

## Milestone 0: Documentation truth

- Decision record merged.
- README and Spec contradictions removed.
- Manifesto SAMPO section added.
- Acceptance tests documented.
- No code selected.

## Milestone 1: Read-only filesystem catalogue

- One local process.
- Loopback browser UI.
- Register one filesystem provider.
- Read-only discovery.
- Hashing.
- Search.
- Group exact duplicates as appearances.
- No copying yet.

Demonstrates AT-01, AT-02, AT-04, AT-05, and part of AT-21.

## Milestone 2: Provider enrollment and portable memory

- Three-choice unknown-provider prompt.
- `.mas-h` enrollment.
- Provider-local ledger.
- Home Seshat catalogue.
- Reconnect comparison.
- Metadata-loss behavior.

Demonstrates AT-14 through AT-17.

## Milestone 3: One verified managed filesystem copy

- `Keep a copy`.
- Explainable plan.
- User approval.
- Durable job.
- Staging.
- Full verification.
- Managed custody.
- Search and open routing.

Demonstrates AT-03, AT-12, AT-19, and AT-22.

## Milestone 4: Persistent protection contracts

- Contract states.
- Continuous maintenance within approved terms.
- Deleted managed-copy replacement.
- Scribbled managed-copy custody transfer.
- Unfulfillable contract behavior.
- Contract amendment.

Demonstrates AT-07 through AT-11 and AT-13.

## Milestone 5: Local S3-compatible provider

- Local disposable object store.
- Capability-based provider interface.
- Upload staging and verification.
- Remote access routing.
- Failure injection.

Demonstrates AT-12, AT-20, and AT-22 against a genuinely non-filesystem provider.

## Milestone 6: Projects by snapshot

- Many-to-many membership.
- Reviewed project snapshot.
- File-level contract creation.
- Quiet report of later additions.

Demonstrates AT-18.

## Milestone 7: MVP hardening

- Crash recovery.
- idempotency.
- catalogue integrity.
- audit explanations.
- resource limits.
- security review.
- all acceptance tests green.

---

# Part VIII: Architecture questions Codex may still resolve

Codex may propose answers to these in `MVP-ARCHITECTURE.md`, but must label them implementation choices rather than product-owner decisions:

- specific programming language;
- specific web framework;
- specific SQLite library;
- exact digest algorithm, provided the algorithm is recorded and migration remains possible;
- exact local S3-compatible development tool;
- exact provider-local database schema;
- exact home data directory per operating system;
- exact native watcher integration;
- scan and rehash optimization;
- how often contract reconciliation runs;
- exact local-session and CSRF mechanism;
- exact staging filenames and atomic commit strategy;
- whether the MVP supports Windows only first or remains portable from milestone one;
- how local S3 bytes are made provider-natively retrievable without MAS-H;
- how much provider-local change history is retained;
- whether minimal directly observed fork lineage is stored in MVP.

For each choice, document:

- alternatives considered;
- safety consequences;
- portability consequences;
- build-versus-integrate reasoning;
- how acceptance tests prove the choice.

Do not ask the product owner to decide low-level implementation details unless the choice changes user-visible behavior, safety, cost, or future compatibility.

---

# Final instruction to Codex

Work slowly and treat ambiguity as a defect to report, not permission to improvise.

Produce the documentation and architecture deliverables. Do not write implementation code yet.

At the end, give the product owner:

1. a concise summary of the architecture;
2. the exact files changed;
3. every approved decision now represented in documentation;
4. every remaining ambiguity;
5. the proposed next implementation milestone;
6. confirmation that no production code was created.
