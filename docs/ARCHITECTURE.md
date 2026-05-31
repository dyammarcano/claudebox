# Architecture

`claudebox` is a single Go binary that drives Docker to run Claude Code headless
in a throwaway container. The Dockerfile and entrypoint are embedded via
`go:embed`, so there are no runtime file dependencies.

## System Overview

```mermaid
flowchart TB
    user([User]) -->|"claudebox run -p ... -w ..."| cli

    subgraph binary["claudebox binary"]
        cli["cmd/claudebox<br/>Cobra run command"]
        cfg["internal/config<br/>Normalize + Validate"]
        sb["internal/sandbox<br/>lifecycle orchestrator"]
        ver["internal/version<br/>detect host claude"]
        creds["internal/credentials<br/>seed prepare + scrub"]
        eng["internal/engine<br/>Docker SDK wrapper"]
        assets["assets<br/>embedded Dockerfile + entrypoint.sh"]
        cli --> cfg --> sb
        sb --> ver
        sb --> creds
        sb --> eng
        eng --> assets
    end

    subgraph host["Host"]
        claudebin["claude (host binary)<br/>version source only"]
        home["~/.claude<br/>.credentials.json, settings.json"]
        ws["workspace dir"]
        seed["temp seed dir<br/>~/.claudebox/seeds (0700)"]
    end

    subgraph docker["Docker engine"]
        img["image claudebox:&lt;version&gt;"]
        ctr["throwaway container<br/>non-root agent user"]
    end

    ver -.reads --version.-> claudebin
    creds -.copies creds.-> home
    creds -->|writes scrubbed copy| seed
    eng -->|ImageBuild CLAUDE_VERSION| img
    eng -->|ContainerCreate/Start| ctr
    img --> ctr
    ws -->|bind rw /workspace| ctr
    seed -->|bind ro /seed| ctr
    ctr -->|stdout/stderr stream| cli
    eng -->|ContainerRemove force| ctr
```

## Run lifecycle (sequence)

```mermaid
sequenceDiagram
    actor U as User
    participant C as cmd/claudebox
    participant S as sandbox
    participant V as version
    participant Cr as credentials
    participant E as engine (Docker SDK)
    participant D as Docker engine
    participant K as container

    U->>C: run -p "..." -w ./proj
    C->>C: Normalize + Validate config
    C->>S: Run(ctx)
    S->>E: Ping (daemon reachable?)
    alt --claude-version unset
        S->>V: Detect(host claude --version)
        V-->>S: "2.1.158"
    end
    S->>E: ImageExists(claudebox:2.1.158)?
    alt missing or --rebuild/--no-cache
        S->>E: BuildImage(CLAUDE_VERSION=2.1.158)
        E->>D: ImageBuild(tar of embedded assets)
        D-->>E: build log stream
    end
    S->>Cr: PrepareSeed(~/.claude, base=~/.claudebox/seeds)
    Cr-->>S: seed dir (creds [+settings]) + cleanup
    alt no API key and no OAuth creds
        S-->>U: error: log in or pass --api-key
    end
    S->>E: Run(image, mounts, env)
    E->>D: ContainerCreate + ContainerWait + ContainerStart
    D->>K: start (entrypoint seeds ~/.claude, exec claude -p)
    K-->>E: stdout/stderr (stdcopy de-mux)
    E-->>U: streamed output
    K-->>D: exit code
    D-->>E: status
    E->>D: ContainerRemove(force)  %% unless --keep
    S->>Cr: cleanup() (rm seed)  %% deferred
    E-->>S: exit code
    S-->>C: exit code
    C-->>U: process exits with claude's code
```

## Credential isolation (why a seed copy)

```mermaid
flowchart LR
    subgraph host["Host (never written)"]
        h["~/.claude/.credentials.json<br/>settings.json"]
    end
    subgraph temp["Ephemeral seed (0700, deleted on exit)"]
        s["/seed (RO mount)<br/>credentials [+settings]"]
    end
    subgraph ctr["Container (throwaway)"]
        w["~/.claude (writable)<br/>tokens refresh here, history here"]
    end
    h -->|copy scrubbed subset| s
    s -->|entrypoint cp -a| w
    w -.dies with container.-> x([discarded])
```

OAuth tokens refresh at runtime (Claude rewrites `.credentials.json`), so the
in-container `~/.claude` must be writable. Mounting the host dir read-only would
break refresh and history; mounting it read-write would pollute the host. The
seed-copy pattern keeps the host untouched and read-only while giving the
container a writable, disposable copy.

## Component responsibilities

| Package | Responsibility | Key types/functions |
|---------|----------------|---------------------|
| `cmd/claudebox` | CLI surface, flag parsing, exit-code propagation | `runCmd`, `runE`, `exitCodeError` |
| `internal/config` | Config struct, defaults, validation | `Config`, `Normalize`, `Validate`, `ResolvedImageTag` |
| `internal/version` | Detect/parse host Claude version | `Detect`, `Parse` |
| `internal/credentials` | Prepare scrubbed ephemeral seed | `PrepareSeed`, `Seed`, `DefaultClaudeHome` |
| `internal/engine` | Docker build/run/stream/remove | `Engine`, `BuildImage`, `Run`, `ImageExists` |
| `internal/sandbox` | Orchestrate the lifecycle | `Sandbox`, `Run`, `defaultSeedBase` |
| `assets` | Embedded build context | `Dockerfile`, `Entrypoint` |
