---
name: sync-upstream
description: Manually synchronize gh-aw-plus with the latest stable github/gh-aw release while preserving Claude OAuth behavior and upstream security.
---

Read `FORK.md` first, then follow the canonical procedure in `skills/sync-upstream/SKILL.md` from the repository root. Target the latest published stable release; exclude prereleases and do not silently follow `main`. Preserve the functional invariant, not today's implementation. The canonical skill contains the complete workflow, validation requirements, security checks, and reporting contract. Do not implement scheduled synchronization.
