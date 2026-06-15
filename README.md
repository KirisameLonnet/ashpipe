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

**User path** — shell hook detects portal directory on `cd`, mounts SSHFS, hands terminal to remote shell. On `exit`, SSHFS is automatically unmounted.

**Agent path** — `ashpipe mcp` starts an MCP server (stdio) that exposes `bash`, `read`, `write`, `edit`, `diff`, `ls`, `glob` tools backed by the same SSH connection.

Both paths share one SSH connection pool — no duplicate handshakes.

## Requirements

| Platform | SSHFS | Notes |
|----------|-------|-------|
| macOS    | [macFUSE](https://osxfuse.github.io) + `brew install sshfs` | macFUSE requires manual install (kernel extension) |
| Linux    | `sudo apt install sshfs` | or `dnf install fuse-sshfs` / `pacman -S sshfs` |
| NixOS    | provided by `flake.nix` dev shell | see below |

SSH host must already be in `~/.ssh/known_hosts`. Connect manually once if not:
```bash
ssh user@hostname
```

### Nix / NixOS

ashpipe ships a `flake.nix` with full cross-platform support (Linux x86_64/aarch64, macOS aarch64/x86_64).

```bash
# Enter dev shell (provides go, sshfs, gopls, gotools)
nix develop

# Build the binary
nix build
./result/bin/ashpipe --help
```

**Shell hook via home-manager** (do not echo directly into `~/.zshrc` on managed systems):

```nix
programs.zsh.initExtra = ''
  eval "$(ashpipe hook zsh)"
'';
# or for bash:
programs.bash.initExtra = ''
  eval "$(ashpipe hook bash)"
'';
```

**Package** — add to your NixOS `configuration.nix` or `home.nix`:

```nix
# flake input
inputs.ashpipe.url = "github:KirisameLonnet/ashpipe";

# package
environment.systemPackages = [ inputs.ashpipe.packages.${system}.default ];
# or with home-manager
home.packages = [ inputs.ashpipe.packages.${system}.default ];
```

On **macOS with Nix**, macFUSE cannot be managed by Nix (kernel extension). Install it separately:
1. Download from [osxfuse.github.io](https://osxfuse.github.io) and install
2. `brew install sshfs`

On **NixOS**, sshfs is included in the dev shell and available as `pkgs.sshfs-fuse`.

## Install

Once installed via any method below, the hook just works — no full path needed:
```bash
eval "$(ashpipe hook zsh)"   # or bash / fish
```

### Homebrew (macOS + Linux)
```bash
brew install KirisameLonnet/tap/ashpipe
```

### AUR (Arch Linux)
```bash
yay -S ashpipe-bin
# or: paru -S ashpipe-bin
```

### Nix / home-manager
```nix
# flake.nix
inputs.ashpipe.url = "github:KirisameLonnet/ashpipe";

# home.nix
home.packages = [ inputs.ashpipe.packages.${system}.default ];
programs.zsh.initExtra = ''
  eval "$(ashpipe hook zsh)"
'';
```

### Debian / Ubuntu (.deb)
```bash
# Download from https://github.com/KirisameLonnet/ashpipe/releases
sudo dpkg -i ashpipe_*_amd64.deb
# or for arm64:
sudo dpkg -i ashpipe_*_arm64.deb
```

### Fedora / RHEL / openSUSE (.rpm)
```bash
sudo rpm -i ashpipe_*_amd64.rpm
# or for arm64:
sudo rpm -i ashpipe_*_arm64.rpm
```

### Go
```bash
go install github.com/KirisameLonnet/ashpipe@latest
```

### Build from source
```bash
git clone https://github.com/KirisameLonnet/ashpipe
cd ashpipe
go build -o ashpipe .
# Move to somewhere in your PATH, e.g.:
sudo mv ashpipe /usr/local/bin/
```

## Quick start

```bash
# 1. Initialize a workspace
mkdir my-workspace && cd my-workspace
ashpipe init

# 2. Add a portal
ashpipe add prod ubuntu@server.example.com:/opt/app

# 3. Add shell hook — run this to see the line to add to your config:
ashpipe hook zsh   # or bash / fish

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

# portal directories are created automatically: prod/ and dev/

# cd prod/   → SSH session on prod
# cd dev/    → SSH session on dev
# Agent can target a specific portal with the `host` parameter
```

## Security notes

- Host key verification uses `~/.ssh/known_hosts` — do not skip initial manual connection
- Passwords in `config.yaml` are stored in plaintext — use SSH keys or agent instead
- `config.yaml` should not be committed to version control if it contains passwords
- Add `.ashpipe/session.json` to `.gitignore` (generated at runtime)
