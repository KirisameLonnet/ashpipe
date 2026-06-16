# ashpipe

`cd` into a directory, get an SSH session — for humans and AI agents.

```
cd workspace/remote-space-prod/   →   terminal becomes a remote SSH session
```

ashpipe turns a local directory into a transparent portal to a remote host. File access goes through SSHFS (you see real remote files), and the terminal pty is handed directly to the remote shell. For AI agents (Claude Code, Codex), mounted portal directories are ordinary local directories, so built-in file tools can read, edit, diff, and search without switching to ashpipe-specific tools.

## Safety warning

Do **not** manually delete ashpipe portal paths or mount directories with
`rm -rf`. ashpipe manages them. Use `ashpipe unmount` and `ashpipe remove`
instead.

If you must clean anything manually, unmount first and verify it is no longer a
mount point:

```bash
ashpipe unmount <portal>
mount | grep ashpipe
```

Deleting a live SSHFS/FUSE mount can delete files on the remote host.

## How it works

```
workspace/
├── .ashpipe/
│   └── config.yaml          ← host and portal definitions
├── remote-space-prod -> /Volumes/ashpipe/<workspace-id>/remote-space-prod      # macOS
└── remote-space-dev  -> /media/$USER/ashpipe/<workspace-id>/remote-space-dev   # Linux
```

The real SSHFS mount points live outside the workspace (for example under
`/Volumes/ashpipe` on macOS or `/media/$USER/ashpipe` on Linux). This mirrors
where external drives normally appear, making it clear that the portal target is
a mounted filesystem. It also keeps `rm -rf remote-space-prod` and
`rm -rf .ashpipe` from recursing into remote files.

**User path** — shell hook detects portal directory on `cd`, mounts SSHFS, hands terminal to remote shell. On `exit`, SSHFS is automatically unmounted.

**Agent path** — `ashpipe mount` mounts portal directories via SSHFS without starting an interactive SSH shell. Agents then use their default built-in file tools directly against the mounted directory.

Both paths are system-native: humans get a remote pty, agents get real mounted directories.

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

### Homebrew

```bash
brew install KirisameLonnet/tap/ashpipe
```

### Arch Linux

The repository includes an Arch `PKGBUILD` under `packaging/aur/` for
`ashpipe-bin`. Clone the repository and build it locally:

```bash
git clone https://github.com/KirisameLonnet/ashpipe.git
cd ashpipe/packaging/aur
makepkg -si
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

ashpipe is workspace-based. The first decision is where the stable local
workspace should live. That directory is the durable relationship between your
machine, the portal names, and the remote paths. Do not initialize ashpipe in a
temporary download folder or in a random directory you do not plan to keep.

```bash
# 1. Pick or create a stable workspace directory
mkdir -p ~/workspaces/prod-servers
cd ~/workspaces/prod-servers

# 2. Initialize ashpipe metadata in this workspace
ashpipe init

# 3. Follow the init guidance and add a portal
ashpipe add prod ubuntu@server.example.com:/opt/app

# 4. Add the shell hook once, in your shell startup config
ashpipe hook zsh   # or bash / fish

# 5. Mount portals for agent-native file access
ashpipe mount       # mounted portal dirs behave like normal local dirs

# 6. Connect by entering the portal directory
cd prod/          # ← terminal is now a remote SSH session on ubuntu@server.example.com:/opt/app
pwd               # returns /opt/app
exit              # back to local shell, SSHFS unmounted automatically
```

After `ashpipe init`, the workspace owns `.ashpipe/config.yaml`, generated
agent instructions, and portal symlinks. Return to this same workspace whenever
you want to work with the same remote directories.

## CLI reference

```
ashpipe init                              Initialize workspace in current directory
ashpipe add <name> <user@host:/path>      Add a portal
ashpipe remove <name> [--force]           Remove a portal (--force deletes non-empty dir)
ashpipe mount [name]                      Mount one/all portals via SSHFS
ashpipe unmount [name]                    Unmount one/all portals
ashpipe connect <name>                    Connect to portal (also triggered by cd)
ashpipe status                            Show portal mount status
ashpipe hook zsh|bash|fish               Print shell hook to eval
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

### Claude Code plugin

ashpipe ships as a Claude Code plugin. Once installed, portals auto-mount on session start — no manual setup needed.

**Install the plugin (in Claude Code):**

```
# Step 1: Add the ashpipe marketplace
/plugin marketplace add KirisameLonnet/ashpipe

# Step 2: Install the plugin
/plugin install ashpipe@ashpipe
```

**Or pre-configure for your team** — add to your project's `.claude/settings.json` so collaborators get prompted automatically:

```
/plugin marketplace add --scope project KirisameLonnet/ashpipe
/plugin install ashpipe@ashpipe
```

If Claude Code has cached an older marketplace definition, refresh it first:

```
/plugin marketplace remove ashpipe
/plugin marketplace add KirisameLonnet/ashpipe
/plugin install ashpipe@ashpipe
```

**Verify installation:**

```
# In Claude Code, type:
/ashpipe-mount
# Should show portal status or "Not in an ashpipe workspace"
```

**What the plugin does:**

- **SessionStart hook** — auto-detects `.ashpipe/` in the workspace and runs `ashpipe mount` before you interact with the agent
- **`/ashpipe-mount` skill** — invokable by user or auto-triggered by Claude when portal operations are needed; shows current portal status and provides workflow guidance
- **CLAUDE.md** — generated by `ashpipe add`, tells Claude to use built-in Read/Write/Edit/Bash tools directly on portal directories

Claude Code loads the plugin skill from `.claude/skills/ashpipe-mount` and the
hook from `hooks/hooks.json`.

**Agent workflow after plugin install:**

1. Open Claude Code in an ashpipe workspace
2. Portals are already mounted (hook ran on session start)
3. Agent uses built-in tools on portal directories — no special commands needed
4. If a portal is not mounted, agent knows to run `ashpipe mount`

### Codex plugin

ashpipe also ships a Codex plugin manifest and skill in this repository:

- `.agents/plugins/marketplace.json`
- `plugins/ashpipe/.codex-plugin/plugin.json`
- `plugins/ashpipe/skills/ashpipe-mount/SKILL.md`

**Install the plugin:**

```bash
codex plugin marketplace add KirisameLonnet/ashpipe --ref main
codex plugin add ashpipe@ashpipe
```

**Verify installation:**

```bash
codex plugin marketplace list
codex plugin list --marketplace ashpipe
```

**Update or reinstall:**

```bash
codex plugin marketplace upgrade ashpipe
codex plugin remove ashpipe
codex plugin add ashpipe@ashpipe
```

**What the Codex plugin does:**

- Activates when `.ashpipe/config.yaml` or ashpipe portal work is detected
- Tells Codex to run `ashpipe mount` before reading or editing portal contents
- Tells Codex to use native file tools on mounted portal directories, not MCP or SSH file fallbacks
- Repeats the safety rule: never manually `rm -rf` portal symlinks or mount directories

### Other agent tools

For agent tools not listed above, prefer asking the agent to install or import
the ashpipe skill from this repository instead of manually copying workflow
notes into every session. The agent should learn the workflow itself:

- Detect an ashpipe workspace by `.ashpipe/config.yaml`
- Run `ashpipe mount` from the workspace root before touching portal contents
- Use its native read, edit, write, search, and diff tools on portal directories
- Avoid MCP or SSH file-tool fallbacks for normal file operations
- Never manually delete portal symlinks or mount directories; use `ashpipe unmount`
  and `ashpipe remove`

If an agent has no plugin or skill mechanism, use the manual setup below and
rely on the generated `AGENTS.md` guidance in the workspace.

### Manual agent setup (without plugin)

If you prefer not to install the plugin:

```bash
cd my-workspace
ashpipe mount
```

After that, portal directories are real SSHFS-mounted directories. Claude Code and Codex use their default built-in tools for reading, editing, writing, searching, and diffing files. No ashpipe-specific tool choice is required.

Generated `CLAUDE.md` and `AGENTS.md` files tell agents to treat portal directories as ordinary workspace directories and to ask for `ashpipe mount` if the mount is missing.

## Multiple portals

```bash
ashpipe add prod ubuntu@prod.example.com:/opt/app
ashpipe add dev developer@dev.example.com:/home/dev/project

# portal directories are created automatically: prod/ and dev/

# cd prod/   → SSH session on prod
# cd dev/    → SSH session on dev
# Agent-native file access:
ashpipe mount
```

## Security notes

- Host key verification uses `~/.ssh/known_hosts` — do not skip initial manual connection
- Passwords in `config.yaml` are stored in plaintext — use SSH keys or agent instead
- `config.yaml` should not be committed to version control if it contains passwords
- Add `.ashpipe/session.json` to `.gitignore` (generated at runtime)
