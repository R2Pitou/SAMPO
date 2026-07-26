# Architectural Decisions

## Decision 1: Do we need a GUI?

### Status
Accepted

### Context
When building a storage hypervisor and metadata manager, developers often assume a graphical user interface (GUI) or dashboard is required for user interaction, configuration, and monitoring.

### Decision
**We do not need a built-in GUI inside the MAS-H core repository.**

MAS-H is designed as a headless control plane storage hypervisor and digital librarian. The core engine is decoupled from user interaction layers.

### Consequence & Rationale

1. **Alignment with Design Principles**:
   - *Users express intent, not implementation:* The system should work automatically in the background based on declarative policies defined in JSON configuration files (`config.json`), rather than requiring interactive manual clicking.
   - *Orchestrate, do not replace:* MAS-H orchestrates existing open-source tools (SMB, WebDAV, Git, Syncthing, Everything) rather than forcing users into a proprietary, custom interface.
   - *Feels like a librarian, not an admin panel:* A classic admin panel creates cognitive load. The system works as an autonomous coordinator in the background.

2. **Decoupled Architecture**:
   - The **Gateway** component provides standard interfaces like SMB, WebDAV, and HTTP/REST.
   - Any graphical interface (e.g., a dashboard for viewing replica health or configuring storage providers) can be built as a separate, optional client application. This client can consume the standard REST endpoints provided by the `Gateway` REST API (e.g., `/objects`, `/providers`).
   - Keeping the core headless keeps the binary extremely lightweight, cross-platform compatible, and easy to deploy in server, NAS, or headless environments.
