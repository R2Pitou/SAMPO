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

**Pass:** Search returns one Content item; USB Appearance is unavailable, not deleted; managed Appearance is available; Open routes to it; no replacement is scheduled merely because the USB is disconnected.

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

Fulfill a two-managed-copy Contract, then externally delete one and positively confirm its absence while exact source bytes remain accessible.

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

## AT-23: Providers are not Failure Domains

Register a filesystem Provider and a local S3-compatible Provider whose backing data resides on the same physical disk. Request unconstrained **Keep two copies**.

**Pass:**

- SAMPO represents or establishes the shared Failure Domain separately from the two Provider identities;
- the two Providers do not satisfy two-domain protection;
- the Contract remains unfulfilled and the Plan explains the shared loss boundary;
- moving or adding an eligible destination in an independently established Failure Domain allows fulfillment;
- unknown independence is not treated as proven independence.

## AT-24: Explicit weaker same-domain protection

With only one eligible Failure Domain, review a Plan for two distinct managed instances within that domain.

**Pass:**

- the ordinary unconstrained Plan remains unfulfilled;
- the weaker alternative is clearly labeled as reduced protection;
- it requires explicit approval as a material Contract term;
- after approval, two distinct verified committed managed Appearances can satisfy that weaker Contract;
- aliases or multiple locators for one underlying storage instance do not count as two.

## AT-25: Unavailable and surplus are not deletion targets

Start with a Contract requiring a minimum of two managed Appearances across independent Failure Domains and three qualifying managed Appearances present.

**Pass:**

- SAMPO reports two required and one surplus;
- it creates no automatic retirement Job;
- disconnecting one Provider reports that Appearance as known but currently unverifiable, not missing;
- disconnection alone creates no replacement Job;
- reconnect and verification restore current confidence;
- replacement becomes eligible only after positive missing or invalid evidence.

## AT-26: Readable managed layout survives renames and collisions

Approve a managed filesystem copy whose first relative path is `Project/Mix.wav`. Then rename the user-owned source and create a second different Content item that would map to the same destination name.

**Pass:**

- the committed copy is an ordinary file beneath `library/Project/Mix.wav` or a deterministic Provider-safe spelling;
- it opens with a native tool after SAMPO is stopped;
- the later source rename updates catalogue evidence without renaming the committed copy;
- the collision publishes the second file under a deterministic readable disambiguated name, preserving `.wav`, and never overwrites the first;
- machine staging and metadata remain under `.sampo/`.

## AT-27: A manually placed file is not silently adopted

Place a file manually beneath `library/` without a SAMPO creation Job.

**Pass:** SAMPO records it as user-owned and offers **Adopt**, **Leave it mine**, and **Ask later**. Leave and Ask preserve user custody. Adopt transfers custody only after explicit approval, stable-byte verification, applicable Contract handling, and an audit event. A file changed during adoption remains user-owned and requires a refreshed decision.

## AT-28: Windows control plane and Ubuntu S3 participation

Run the SAMPO control plane and loopback UI on Windows and enroll an S3-compatible service hosted on Ubuntu Server.

**Pass:** the complete MVP journey works without a Linux SAMPO control-plane process; ordinary S3 tools can retrieve committed bytes after SAMPO stops; Windows-specific integration remains outside the domain and Provider contracts.

## AT-29: Usage reporting is factual, not a billing authority

Perform filesystem and S3-compatible transfers, including a quota or billing-related Provider failure where the test service can simulate one.

**Pass:** SAMPO reports measured bytes, operations where exposed, dated transfer history, and factual Provider failures with provenance. It displays no currency estimate, claims no budget enforcement, and enrollment of a paid Provider warns the user to configure Provider-side budgets and alerts.

## Cross-cutting test requirements

- Run provider-contract tests against both provider classes.
- Inject process crashes before, during, and after staging, verification, provider commit, and catalogue finalization.
- Run Jobs under duplicate-claim and lease-expiry conditions.
- Test case-insensitive collisions, illegal names, exhausted destinations, disconnected providers, and changed source evidence.
- Verify that raw observations cannot directly authorize custody, deletion, or Contract fulfillment.
- Verify every state-changing UI action produces an explainable Plan, approved Job, or explicit rejection.
- Verify Provider count never substitutes for established Failure Domain independence.
- Verify Contract reports distinguish available, unavailable, confirmed missing/invalid, and surplus managed Appearances.
- Verify layout and adoption behavior cannot turn directory location into custody.
- Verify factual usage views never imply a complete bill or a SAMPO-enforced budget.
