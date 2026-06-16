---
name: ashpipe-mount
description: Manage ashpipe remote workspace portals. Use when the user mentions ashpipe, SSHFS portals, remote workspace files, or when a project contains .ashpipe/config.yaml.
argument-hint: "[mount|unmount|status|add]"
allowed-tools: Bash(ashpipe *)
---

## ashpipe Remote Workspace Manager

This workspace uses ashpipe to transparently access remote SSH hosts. Portal directories are SSHFS-mounted remote paths that appear as local directories.

### Current workspace state

!`ashpipe status 2>/dev/null || echo "Not in an ashpipe workspace"`

### How portals work

- Portal directories are symlinks to private SSHFS mount points outside the workspace
- When mounted, use standard Read/Write/Edit/Bash tools directly on portal paths; they are real files on the remote host
- Never delete portal directories with `rm -rf`; use `ashpipe unmount` and `ashpipe remove`
- Do not fall back to MCP or SSH file tools for normal portal file operations

### Commands

- `ashpipe mount [name]` mounts one or all portals via SSHFS
- `ashpipe unmount [name]` unmounts one or all portals
- `ashpipe status` shows portal mount status
- `ashpipe add <name> <user@host:/path>` adds a new portal
- `ashpipe remove <name>` removes a portal after unmounting it
- `ashpipe connect <name>` mounts and opens an interactive SSH session for human use

### Workflow

1. Find the ashpipe workspace root by walking upward until `.ashpipe/config.yaml` exists.
2. If portals are unmounted, run `ashpipe mount` from the workspace root.
3. Work with files inside portal directories using normal Claude Code tools.
4. Shell commands inside portals execute locally against SSHFS-mounted files.
5. For remote-only commands such as `systemctl` or package managers, use `ssh user@host "command"`.
