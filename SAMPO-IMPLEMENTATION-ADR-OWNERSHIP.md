# SAMPO Implementation ADR Ownership

**Status:** Product decisions complete  
**Purpose:** Define which remaining choices belong to Codex as implementation ADRs  
**Authority:** Product-owner approved

The product-level architecture decisions are complete.

The remaining choices below belong to Codex. Codex should select, document, and justify them as implementation ADRs without escalating them to the product owner unless a choice changes user-visible behavior, weakens a safety promise, creates monetary authority, or blocks future compatibility.

---

## Governing rule

> Choose implementation details independently. Escalate only when a choice changes user-visible behavior, weakens a safety promise, creates monetary authority, or blocks future compatibility.

The product owner does not need to choose SQLite lock modes, multipart-upload cleanup rules, Windows notification APIs, collision suffix formats, or similar implementation details.

---

## 1. Language and application stack

Codex should select and justify:

- backend language;
- local web framework;
- browser frontend framework or a plain web approach;
- build tooling;
- packaging;
- dependency strategy;
- development workflow;
- test framework.

The choice must support a local browser UI, a single-user local control plane, durable jobs, filesystem observation, and future portability.

---

## 2. Persistence

Codex should select and justify:

- home Seshat database technology;
- provider-local `.sampo` ledger format;
- transaction model;
- corruption detection;
- corruption recovery;
- backup strategy;
- schema migration approach;
- migration rollback behavior;
- concurrency model.

The persistence design must preserve the rule that loss of metadata never makes ordinary files unreadable.

---

## 3. Digest strategy

Codex should select and justify:

- initial complete-content digest algorithm;
- how the algorithm name is stored with the digest;
- whether more than one digest may coexist;
- future digest migration;
- rehash scheduling;
- how digest confidence is represented;
- how digest verification is recorded.

The architecture must not assume one digest algorithm will remain permanent forever.

---

## 4. Mutable-file hashing consistency

Codex should define:

- how SAMPO detects that a file changed while being hashed;
- which metadata is sampled before and after hashing;
- retry behavior;
- maximum retry behavior;
- what happens when a file is continuously changing;
- when a digest result is considered trustworthy;
- how uncertainty is recorded.

SAMPO must never treat an unstable read as verified exact content.

---

## 5. Local S3-compatible development service

Codex should choose and justify a local disposable S3-compatible service, such as:

- MinIO;
- SeaweedFS;
- Garage;
- LocalStack;
- another suitable implementation.

The ADR should cover:

- Docker setup;
- local credentials;
- bucket initialization;
- test reset;
- failure injection;
- interrupted upload testing;
- direct provider-native retrieval without SAMPO;
- cleanup behavior.

The development workflow must not require a public cloud account or credit card.

---

## 6. Filesystem staging and atomic publication

Codex should define:

- staging directory structure;
- temporary filenames;
- write and flush behavior;
- fsync or equivalent durability behavior;
- atomic publication where supported;
- fallback behavior where atomic rename is unavailable;
- interrupted-copy cleanup;
- restart recovery;
- collision handling;
- permission handling.

A staged or partial copy must never count as committed or contract-satisfying.

---

## 7. S3 staging and publication

Codex should define:

- multipart-upload handling;
- incomplete-upload cleanup;
- idempotent retries;
- object naming during staging;
- object publication;
- independent verification;
- metadata placement;
- when an object becomes committed;
- timeout behavior;
- provider error mapping.

An upload-success response or ETag alone must not be treated as proof of exact byte equality.

---

## 8. Managed-copy layout mechanics

The product rule is already fixed:

- preserve the first approved human-readable relative path;
- never automatically rebuild that path after later renames;
- use readable paths under `library/`;
- keep machine metadata and staging under `.sampo/`;
- disambiguate collisions without creating Boatman’s fever dream;
- preserve original file extensions;
- keep committed files directly retrievable without SAMPO.

Codex should decide:

- exact collision suffix format;
- path sanitization rules;
- reserved-name handling;
- maximum path lengths;
- cross-provider character compatibility;
- Unicode normalization;
- source-root context when needed;
- behavior when two different contents claim the same path;
- behavior when one provider cannot represent the original path exactly.

---

## 9. Provider identity

Codex should define:

- which filesystem identifiers to use;
- which hardware identifiers to use;
- how evidence is combined;
- how identity confidence is represented;
- behavior when identifiers are unavailable;
- behavior when identifiers conflict;
- clone detection;
- replacement-drive handling;
- copied `.sampo` identity handling;
- simultaneous duplicate identity handling.

Provider identity must not rely on drive letter alone.

---

## 10. Probable-rename confidence

The product behavior is already fixed:

> Observe eagerly, remember accurately, and intervene reluctantly.

SAMPO may record confirmed, probable, or unresolved rename or move relationships, but must not fabricate certainty.

Codex should define:

- evidence scoring;
- time windows;
- native-ID weighting;
- digest weighting;
- size and timestamp use;
- ambiguity thresholds;
- confidence labels;
- reconciliation behavior;
- user-visible wording.

---

## 11. Failure-domain representation

The product rule is already fixed:

> An unconstrained “Keep two copies” requires at least two verified SAMPO-managed appearances across two independent failure domains.

Codex should define:

- failure-domain data model;
- relationship between providers and failure domains;
- disks, partitions, mounts, buckets, services, and backing storage;
- local S3 on the same disk as a filesystem provider;
- unknown failure-domain handling;
- safe defaults;
- setup-wizard collection;
- user correction;
- contract evaluation;
- unavailable versus missing behavior.

Different providers must not automatically be assumed to be different failure domains.

---

## 12. Session and browser security

Codex should select and justify:

- local session-token design;
- CSRF protection;
- origin checks;
- cookie settings;
- request-body limits;
- timeouts;
- loopback binding;
- browser-launch mechanics;
- local authentication assumptions;
- API exposure rules;
- logging of security-relevant failures.

The MVP remains single-user and local-only.

---

## 13. Initial Windows implementation

The product scope is already fixed:

- Windows hosts the first SAMPO control plane;
- Ubuntu Server participates first through S3-compatible storage;
- Linux control-plane support comes later;
- portability remains an architectural requirement.

Codex should decide:

- Windows service versus user application;
- startup behavior;
- shutdown behavior;
- volume notifications;
- native file-identity APIs;
- watcher implementation;
- open-with-default-application integration;
- application-data paths;
- update and packaging strategy;
- privilege boundaries.

---

## 14. Provider-ledger history and recovery

Codex should define:

- how much observation history remains on each provider;
- append-only versus current-state records;
- compaction;
- checkpoints;
- corruption detection;
- integrity checks;
- provider-ledger backup;
- reconciliation cursors;
- node attribution;
- recovery after partial writes;
- recovery after abrupt removal;
- rebuild scope from raw files;
- rebuild scope from the home catalogue;
- rebuild scope from another SAMPO installation.

The ledger is evidence, not unquestionable truth.

---

## 15. Scan and reconciliation scheduling

Codex should define:

- watcher-plus-scan strategy;
- initial scan;
- reconnect scan;
- periodic verification;
- background throttling;
- resource limits;
- sleeping providers;
- disconnected providers;
- slow providers;
- large files;
- files that change frequently;
- retry intervals;
- user-visible progress.

Provider unavailability must not be treated as deletion.

---

## 16. Job execution details

Codex should define:

- durable job state machine;
- idempotency keys;
- preconditions;
- retries;
- backoff;
- cancellation boundaries;
- crash recovery;
- concurrency limits;
- audit logging;
- resume behavior;
- source revalidation;
- destination revalidation;
- cleanup of SAMPO-owned staging material;
- failure classification.

Boatman may execute only approved work or maintenance already authorized by an active contract.

---

## 17. Adoption implementation

The product rule is already fixed:

- a file manually placed in SAMPO-managed space remains user-owned;
- SAMPO prompts;
- custody transfers only after explicit adoption;
- location is not custody.

Codex should define:

- how Observer detects the file;
- how the prompt is queued;
- prompt persistence;
- Adopt, Leave it mine, and Ask later behavior;
- how the file is matched to known Content;
- how adoption is associated with a Contract;
- behavior when several Contracts could use it;
- behavior when no Contract exists;
- behavior when the file changes during adoption;
- audit records for custody transfer.

---

## 18. Factual usage reporting

The product rule is already fixed:

- SAMPO does not estimate cloud bills;
- SAMPO does not enforce provider budgets;
- enabling a paid Provider makes it eligible under approved Contracts;
- setup warns the user to configure provider-side budgets, quotas, and billing alerts.

Codex may decide how to report factual measurements such as:

- bytes stored;
- bytes uploaded;
- bytes downloaded;
- operation counts where exposed;
- last successful provider operation;
- provider quota errors;
- billing-related provider failures;
- transfer history.

These are facts, not speculative currency estimates.

---

# Product decisions Codex must not reopen

Codex should treat the following as settled product authority:

- SAMPO is the product.
- `sampo` is the machine identifier.
- `.sampo` is provider-local metadata.
- User-originated files are sacred.
- Custody comes from provenance or explicit adoption, never location.
- Exact verified byte equality means one logical Content with multiple Appearances.
- Renames and moves are noted, not mirrored into managed storage.
- External edits create new Content and never trigger synchronization.
- Protection Contracts persist even when temporarily unfulfillable.
- A Contract authorizes continuous maintenance within its approved terms.
- External deletion of a required managed copy is repaired once loss is positively confirmed.
- Provider unavailability is not deletion.
- An unconstrained two-copy Contract requires two independent failure domains.
- Paid Providers are governed by provider-side budget controls, not SAMPO estimates.
- Projects use snapshot semantics in the MVP.
- There is no canonical copy.
- SAMPO routes each operation to the best eligible Appearance.
- The local M.2 copy beats the same bytes over a Cambodian wirehanger.
- The MVP uses a local browser UI.
- The first control-plane host is Windows.
- Ubuntu Server participates first through S3-compatible storage.
- Filesystem and S3 managed storage must remain human-browsable without SAMPO.
- Committed managed copies preserve the first approved human-readable relative path.
- Later byte-identical renames update Seshat only.
- Machine debris stays under `.sampo/`.
- Observer notices.
- Seshat remembers.
- Boatman acts only when a Contract or approved plan requires it.

---

# Escalation test

Codex should escalate a decision only when at least one of these is true:

1. It changes what the user sees or expects.
2. It weakens the First Law or custody safety.
3. It creates new destructive authority.
4. It creates monetary authority.
5. It changes the meaning of a Contract.
6. It changes whether ordinary files remain recoverable without SAMPO.
7. It blocks a clearly approved future compatibility requirement.
8. Two implementation choices produce materially different product behavior.

Otherwise, Codex should choose, document, test, and proceed.
