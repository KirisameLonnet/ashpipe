---
name: ashpipe-mount
description: Manage ashpipe remote workspace portals — mount, unmount, add, remove SSH portals. Use when the user mentions ashpipe, remote portals, SSH workspaces, or when .ashpipe/ directory is present in the project.
when_to_use: When working in a project with .ashpipe/ directory, or when user asks about remote file access, SSH portals, or ashpipe commands.
argument-hint: "[mount|unmount|status|add]"
user-invocable: true
allowed-tools: Bash(ashpipe *)
---

## ashpipe Remote Workspace Manager

This workspace uses ashpipe to transparently access remote SSH hosts. Portal directories are SSHFS-mounted remote paths that appear as local directories.

### Current workspace state

!`ashpipe status 2>/dev/null || echo "Not in an ashpipe workspace"`

### How portals work

- Portal directories are symlinks to private SSHFS mount points outside the workspace
- When mounted, use standard Read/Write/Edit/Bash tools directly on portal paths — they are real files on the remote host
- NEVER delete portal directories with `rm -rf` — use `ashpipe unmount` and `ashpipe remove`

### Commands

- `ashpipe mount [name]` — mount one or all portals via SSHFS
- `ashpipe unmount [name]` — unmount one or all portals
- `ashpipe status` — show portal mount status
- `ashpipe add <name> <user@host:/path>` — add a new portal
- `ashpipe remove <name>` — remove a portal (unmounts first)
- `ashpipe connect <name>` — mount + open interactive SSH session (human use)

### Workflow

1. If portals are unmounted, run `ashpipe mount` first
2. Work with files inside portal directories using normal tools
3. Shell commands inside portals execute locally against SSHFS-mounted files
4. For remote-only commands (systemctl, apt, etc.), use `ssh user@host "command"`
