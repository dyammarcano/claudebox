# Bug Tracker

## Open Bugs

_No known bugs. Build, vet, and all 35 unit tests pass; the image builds and the
in-container binary runs (runtime smoke test passes)._

## Bug Template

### Bug Title
- **Severity:** Critical / High / Medium / Low
- **Status:** Open / In Progress / Fixed
- **Reproducible:** Always / Sometimes / Rare
- **Description:** What happens
- **Expected:** What should happen
- **Steps to Reproduce:**
  1. Step 1
  2. Step 2
- **Workaround:** Temporary solution if any

## Fixed Bugs

| Bug | Severity | Fix | Date |
|-----|----------|-----|------|
| `logs.Close()` skipped on stdcopy non-EOF error path | Low | Switched to `defer logs.Close()` | 2026-05-30 |
