# Manifesto

> *"They're all my children."*

- Every filesystem.
- Every storage provider.
- Every forgotten USB stick.
- Every piece of your digital life.

SAMPO does not judge existing storage.

It accepts the user's digital life as it exists today, not as it should have been organised ten years ago.

The user should never need to remember where a file lives.

Storage is an implementation detail.

---

# Principles and Laws

## First Law

**SAMPO must never make user data less accessible than it was before SAMPO existed.**

If SAMPO disappears, the user's data remains intact on ordinary filesystems using ordinary tools.

---

## Second Law

**The user expresses intent. The Librarian determines implementation.**

Examples:

* User: *Keep three copies.*
  Librarian: Chooses where those copies live.

* User: *Archive this project.*
  Librarian: Selects the appropriate archive tier.

* User: *Keep this hot.*
  Librarian: Places the object on fast storage.

* User: [*Find project files for Sunday Short Mix 5.*](https://r2pitou.github.io/butthurtsplits/index.html#music)
  Librarian: Finds the object without requiring the user to remember where it lives.

---

## Third Law

**Existing tools should be orchestrated before new ones are invented.**

SAMPO prefers to build upon proven open-source software rather than replace it.

If an excellent tool already exists, SAMPO should integrate it.

---

## Fourth Law

**Optimise human time before machine time.**

Disks are cheap.

CPU cycles are cheap.

Human attention is expensive.

SAMPO exists to reduce cognitive load, not benchmark scores.

---

## Fifth Law

**Storage providers are implementation details, not identities.**

NTFS.

ext4.

APFS.

exFAT.

Git.

Cloud object storage.

They're all my children.

The user should never have to care which one currently holds an object.

---

## SAMPO

SAMPO came first.

The name existed before the architecture and before anyone knew what the letters were supposed to mean.

Later, it was retroactively expanded into:

**Storage Abstraction Management & Policy Orchestrator**

> **What exactly is SAMPO?**
>
> “Your guess is as good as mine. I was high. I liked the name.”
>
> — Arttu Pitou, Founder

---

SAMPO is a **Storage Hypervisor with a Digital Librarian control plane**.

The hypervisor provides the infrastructure.

The Librarian provides the judgement.

The user simply asks for their information.
