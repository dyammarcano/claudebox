# Backlog

## Priority Levels

| Priority | Timeline |
|----------|----------|
| P1 | This sprint |
| P2 | This quarter |
| P3 | Future |

## Items

### Security hardening — container runtime lockdown
- **Priority:** P1
- **Category:** Feature / Security
- **Description:** v1 runs the container on the default bridge with full
  capabilities and a writable rootfs. Add opt-in (then default-on) hardening:
  - `--cap-drop ALL` (+ minimal `--cap-add` if needed)
  - read-only root filesystem (`ReadonlyRootfs`) + `tmpfs` for `/tmp` and
    writable Claude home
  - `--security-opt no-new-privileges`, seccomp/AppArmor profile
  - resource limits: `--memory`, `--cpus`, `--pids-limit`
- **Effort:** Medium

### API key visible via `docker inspect`
- **Priority:** P2
- **Category:** Security
- **Description:** When `--api-key`/`ANTHROPIC_API_KEY` is used, the key is passed
  as a container env var and is readable via `docker inspect` for the (brief)
  life of the container. The default OAuth path is unaffected (creds are a
  read-only file mount). Mitigate with a credential-helper file mount or Docker
  secrets so the key never lands in the container config. Documented as a v1
  limitation in the `--api-key` flag help and README.
- **Effort:** Medium

### Network egress allowlist
- **Priority:** P1
- **Category:** Feature / Security
- **Description:** Restrict outbound traffic to the Anthropic API
  (`api.anthropic.com`) via an egress proxy/firewall sidecar so a compromised
  agent cannot exfiltrate data or reach arbitrary hosts. `--network none` exists
  today but blocks the API too.
- **Effort:** Large

### BuildKit-based image build
- **Priority:** P2
- **Category:** Tech Debt
- **Description:** The classic engine builder ignores the `# syntax=` directive.
  Move to a BuildKit session build for cache mounts and faster rebuilds.
- **Effort:** Medium

### Session resume across runs
- **Priority:** P3
- **Category:** Feature
- **Description:** Optionally persist a named Claude session volume so a follow-up
  `claudebox run --resume <id>` can continue a prior conversation, while keeping
  the default "clean instance" behavior.
- **Effort:** Medium

### Alpine/musl image variant
- **Priority:** P3
- **Category:** Feature
- **Description:** Offer a smaller musl-based image (`install.sh` already detects
  musl) behind a `--base` flag, after validating the native binary on musl.
- **Effort:** Small
