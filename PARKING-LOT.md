# SAMPO Parking Lot

These capabilities are not rejected forever. They are excluded from the MVP and must not enter through incidental implementation choices.

## Storage behavior

- Automatic migration or source retirement.
- Cache eviction and automatic hot/cold tiering.
- Deduplication that removes Appearances.
- RAID-like redundancy without an explicit Protection Contract.
- Block-level virtualization.
- Proprietary containers as the sole data representation.
- Automatic deletion of user-owned data.

## Synchronization and version control

- Dropbox-style synchronization and write-through propagation.
- Automatic conflict resolution, merges, or reverts.
- Git-like commits or revision DAGs.
- Inferred ancestry beyond minimal directly witnessed facts.
- Lineage-wide protection policies.

## Interfaces

- SMB and WebDAV.
- Virtual drives and filesystem overlays.
- Explorer/Finder shell extensions.
- Public APIs and remote administration.
- Native .NET GUI and Electron packaging requirements.

## Distribution and identity

- Multi-node control plane or distributed catalogue writes.
- Consensus or simultaneous multi-installation provider mutation.
- Multi-user accounts, roles, shared libraries, and ACL mapping.
- Automatic cloned-provider identity resolution beyond safe detection and user review.

## Providers

- Dropbox, Google Drive, R2, and public S3 accounts.
- Git or GitHub providers.
- Another SAMPO node as a provider.
- Cloud billing optimization.

## Intelligence and search

- Semantic search or AI-required classification.
- AI authority over correctness, custody, identity, deletion, or Contract safety.
- Generic knowledge graphs and arbitrary relationship ontologies.

## Projects and automation

- Living Project Contracts and automatic policy inheritance.
- Automatic draft cleanup or garbage detection.
- Interrupting the user for each newly added Project item.
- Opportunistic scheduling beyond work already authorized by a Contract.

## Metadata

- One sidecar per file as the default.
- Extended attributes or alternate streams as sole identity evidence.
- Claims of full reconstruction where surviving evidence cannot prove it.

## Re-entry rule

A parked capability returns only through an explicit product decision and architecture review covering:

1. The user problem it solves.
2. First-Law safety consequences.
3. New domain semantics and authority.
4. Failure and recovery behavior.
5. Provider capability requirements.
6. Acceptance tests.
7. Build-versus-integrate evidence.
