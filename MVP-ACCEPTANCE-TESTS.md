# SAMPO MVP Acceptance Tests

These tests are product behavior requirements. Every implementation milestone must preserve all tests introduced by earlier milestones. Technology-level tests may add detail but may not weaken these outcomes.

## AT-01: SAMPO removal does not trap data

After every supported MVP operation, stop SAMPO, remove or ignore SAMPO metadata, and open original and committed managed files with provider-native tools.

**Pass:** All user data remains readable without SAMPO.

## AT-02: User-owned source remains untouched

Enroll a filesystem provider containing user files and perform discovery, hashing, search, and protection.

**Pass:** No user file is renamed, moved, overwritten, truncated, deleted, or silently adopted. Only explicitly approved `.sampo/` metadata is added, and before/after hashes match.

## AT-03: The cement USB

Enroll a USB, find Content, approve one managed copy, verify it, and disconnect the USB.

**Pass:** Search returns one Content item; USB Appearance is unavailable, not deleted; managed Appearance is available; Open routes to it.

## AT-04: Five idiot copies

Place identical bytes under different names, folders, and providers.

**Pass:** Search shows one Content item and every Appearance; custody is distinguished; repeated **Keep a copy** reports existing protection without unnecessary copying.

## AT-05: Rename without panic

Rename a user-owned file without changing bytes.

**Pass:** SAMPO records the new locator as continuity when native identity proves it, or as a probable unambiguous relation otherwise. It does not pretend ambiguous evidence is certain.

## AT-06: External edit creates different content, not sync

Edit one of several byte-identical user-owned Appearances.

**Pass:** The changed Appearance becomes new Content; others remain unchanged; no propagation, merge, rollback, or sync-conflict model occurs.

## AT-07: Scribbled Librarian copy

Create a managed copy of Content A, then externally edit it into Content B.

**Pass:** B is preserved; the Appearance becomes user-owned; SAMPO never reverts it; it no longer satisfies A’s Contract.

## AT-08: Scribbled final copy makes the Contract unfulfillable

Repeat AT-07 when no accessible Appearance of A remains.

**Pass:** A’s Contract remains active and becomes unfulfillable; A’s digest/history remains; B does not satisfy A.

## AT-09: Lost Content returns

After AT-08, reconnect a provider containing exact Content A.

**Pass:** Seshat recognizes A; the Contract becomes fulfillable; Boatman may restore protection within existing authority.

## AT-10: Externally deleted managed copy

Fulfill a two-managed-copy Contract, then externally delete one while exact source bytes remain accessible.

**Pass:** The Contract remains active; SAMPO observes the missing Appearance and recreates and verifies a replacement without repeated approval; activity explains why.

## AT-11: Contract amendment controls permanent deletion

Reduce a fulfilled Contract from two managed copies to one.

**Pass:** SAMPO previews the exact managed Appearance to retire; user-owned Appearances are excluded; approval leaves one verified managed copy and does not trigger replacement.

## AT-12: Cambodian wirehanger routing

Make verified identical Appearances available on local fast storage and S3-compatible storage.

**Pass:** ordinary Open chooses local; remote remains known; no canonical status changes; a remote-oriented operation may deliberately choose remote.

## AT-13: Editing when only a managed copy exists

Request Edit when only a managed Appearance is available.

**Pass:** SAMPO does not open the preservation copy as the default writable target; it creates or proposes a user-owned working copy and leaves managed bytes unchanged.

## AT-14: Unknown-provider prompt

Connect an unknown provider and exercise **Yes**, **No**, and **Ask me next time**.

**Pass:** Yes opens setup; No persists exclusion; Ask performs no enrollment and prompts on reconnect; SAMPO writes nothing before permission.

## AT-15: Provider-local memory travels

Enroll a writable drive on installation A, connect it to B, change ordinary files, and return it to A.

**Pass:** Each installation recognizes provider identity where evidence permits, compares home catalogue, provider ledger, and actual contents, trusts no ledger blindly, requires no consensus, and treats ambiguous custody as user-owned.

## AT-16: Provider-ledger loss

Delete or corrupt `.sampo/`.

**Pass:** Ordinary files remain readable; SAMPO can rescan; uncertain custody becomes user-owned; missing metadata grants no destructive authority.

## AT-17: Home-catalogue loss

Remove the home catalogue while leaving providers intact.

**Pass:** Ordinary files remain readable; rebuild recovers only proven evidence; SAMPO does not invent history or custody.

## AT-18: Project snapshot

Create a Project with 37 unique Content items, approve project protection, then add three more.

**Pass:** Contracts cover the original 37; additions do not inherit silently; the UI quietly reports them and offers batch protection.

## AT-19: Interrupted filesystem copy

Terminate Boatman during a filesystem transfer.

**Pass:** Partial staging does not count; no existing user file is truncated or replaced; Job safely retries or resumes; completion follows full verification only.

## AT-20: Interrupted S3 upload

Interrupt the local S3-compatible service during upload.

**Pass:** Contract is not fulfilled; multipart or partial state does not count; retry is idempotent; ETag alone is not accepted as SAMPO verification.

## AT-21: Local-only security

Attempt local and LAN access and malformed/state-changing requests.

**Pass:** Gateway binds to loopback by default; LAN clients cannot reach it; state-changing requests require selected local-session protections; body and time limits apply.

## AT-22: No metadata byte dependency

For committed filesystem and S3-compatible copies, remove SAMPO processes and metadata and retrieve bytes with provider-native tools.

**Pass:** Content is available without SAMPO decoding.

## Cross-cutting test requirements

- Run provider-contract tests against both provider classes.
- Inject process crashes before, during, and after staging, verification, provider commit, and catalogue finalization.
- Run Jobs under duplicate-claim and lease-expiry conditions.
- Test case-insensitive collisions, illegal names, exhausted destinations, disconnected providers, and changed source evidence.
- Verify that raw observations cannot directly authorize custody, deletion, or Contract fulfillment.
- Verify every state-changing UI action produces an explainable Plan, approved Job, or explicit rejection.
