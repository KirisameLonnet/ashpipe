# ashpipe

`cd` into a directory, get an SSH session — for humans and AI agents.

```
cd workspace/remote-space-prod/   →   terminal becomes a remote SSH session
```

ashpipe turns a local directory into a transparent portal to a remote host. File access goes through SSHFS (you see real remote files), and the terminal pty is handed directly to the remote shell. For AI agents (Claude Code, Codex), the same host is accessible via MCP tools.

## How it works

```
workspace/
├── .ashpipe/
│   └── config.yaml          ← host and portal definitions
├── remote-space-prod/       ← portal: cd here → SSH session on prod
└── remote-space-dev/        ← portal: cd here → SSH session on dev
```

**Human path** — shell hook detects portal directory on `cd`, mounts SSHFS, hands terminal to remote shell. On `exit`, SSHFS is automatically unmounted.

**Agent path** — `ashpipe mcp` starts an MCP server (stdio) that exposes `bash`, `read`, `write`, `edit`, `diff`, `ls`, `glob` tools backed by the same SSH connection.

Both paths share one SSH connection pool — no duplicate handshakes.

## Requirements

| Platform | Requirements |
|----------|-------------|
| macOS    | [macFUSE](https://osxfuse.github.io) + `brew install sshfs` |
| Linux    | `sudo apt install sshfs` (or equivalent) |

SSH host must already be in `~/.ssh/known_hosts`. Connect manually once if not:
```bash
ssh user@hostname
```

## Install

```bash
# From source
go install github.com/KirisameLonnet/ashpipe@latest

# With Nix
nix build github:KirisameLonnet/ashpipe
nix develop  # dev shell with go + sshfs + LSP
```

## Quick start

```bash
# 1. Initialize a workspace
mkdir my-workspace && cd my-workspace
ashpipe init

# 2. Add a portal
ashpipe add prod ubuntu@server.example.com:/opt/app

# 3. Add shell hook to ~/.zshrc (or ~/.bashrc / fish)
echo 'eval "$(ashpipe hook zsh)"' >> ~/.zshrc
source ~/.zshrc

# 4. Connect by entering the portal directory
cd prod/          # ← terminal is now a remote SSH session on ubuntu@server.example.com:/opt/app
pwd               # returns /opt/app
exit              # back to local shell, SSHFS unmounted automatically
```

## CLI reference

```
ashpipe init                              Initialize workspace in current directory
ashpipe add <name> <user@host:/path>      Add a portal
ashpipe remove <name> [--force]           Remove a portal (--force deletes non-empty dir)
ashpipe connect <name>                    Connect to portal (also triggered by cd)
ashpipe status                            Show portal mount status
ashpipe hook zsh|bash|fish               Print shell hook to eval
ashpipe mcp                               Start MCP server (stdio)
```

## SSH authentication

Priority order (highest first):

| Method | Config |
|--------|--------|
| Explicit key | `identity_file: ~/.ssh/id_ed25519` in config |
| SSH agent | Picked up from `$SSH_AUTH_SOCK` automatically |
| Default keys | `~/.ssh/id_ed25519`, `~/.ssh/id_ecdsa`, `~/.ssh/id_rsa` |
| No auth | Works when host allows passwordless access |
| Password | `password: "..."` in config — prints warning on every use |

```yaml
# .ashpipe/config.yaml
hosts:
  prod-host:
    hostname: server.example.com
    user: ubuntu
    identity_file: ~/.ssh/id_ed25519   # recommended

  legacy-host:
    hostname: old.example.com
    user: admin
    password: "s3cr3t"                 # allowed, but insecure — warning shown
```

## AI agent integration

### Claude Code

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "ssh": {
      "command": "ashpipe",
      "args": ["mcp"],
      "type": "stdio"
    }
  }
}
```

### OpenAI Codex CLI

Add to `~/.codex/config.yaml`:

```yaml
mcp_servers:
  - name: ssh
    command: ashpipe mcp
    type: stdio
```

### Available MCP tools

| Tool | Description |
|------|-------------|
| `ssh:bash` | Run a command on the remote host (persistent shell, maintains cwd) |
| `ssh:read` | Read a remote file |
| `ssh:write` | Write a remote file |
| `ssh:edit` | Edit a file with old_string → new_string (same API as Claude Code's Edit) |
| `ssh:diff` | Unified diff between two remote files |
| `ssh:ls` | List a remote directory |
| `ssh:glob` | Find files matching a pattern |

File paths are automatically translated: local portal paths → remote paths.

## Multiple portals

```bash
ashpipe add prod ubuntu@prod.example.com:/opt/app
ashpipe add dev developer@dev.example.com:/home/dev/project

# cd prod/   → SSH session on prod
# cd dev/    → SSH session on dev
# Agent can target a specific portal with the `host` parameter
```

## Security notes

- Host key verification uses `~/.ssh/known_hosts` — do not skip initial manual connection
- Passwords in `config.yaml` are stored in plaintext — use SSH keys or agent instead
- `config.yaml` should not be committed to version control if it contains passwords
- Add `.ashpipe/session.json` to `.gitignore` (generated at runtime)
