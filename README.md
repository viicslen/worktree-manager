# Worktree Manager (wtm)

A powerful CLI tool for managing git worktrees with a streamlined workflow designed for developers who work with multiple branches simultaneously.

## Overview

Worktree Manager (`wtm`) simplifies the git worktree workflow by providing an intuitive interface for cloning repositories as bare repositories, creating worktrees, switching between branches, and persisting files across worktrees.

**Note:** This is a very opinionated implementation of how to work with git worktrees. It enforces a specific directory structure and workflow that may not suit all use cases. The tool is designed for developers who frequently context-switch between multiple branches and want a consistent, IDE-friendly approach to managing worktrees.

## Table of Contents

- [Installation](#installation)
- [Core Concepts](#core-concepts)
- [Commands](#commands)
  - [clone](#clone)
  - [checkout](#checkout)
  - [switch](#switch)
  - [persist](#persist)
  - [restore](#restore)
- [Workflow Examples](#workflow-examples)
- [Directory Structure](#directory-structure)
- [Use Cases](#use-cases)

## Installation

### Using Nix (Recommended)

If you have [Nix](https://nixos.org/) with flakes enabled, you can run the tool directly without installation:

```bash
# Run directly from GitHub
nix run github:viicslen/worktree-manager -- --help

# Or run from a local clone
nix run . -- --help
```

To install permanently, add it to your system configuration:

**NixOS (configuration.nix or flake.nix):**

```nix
{
  environment.systemPackages = [
    (pkgs.callPackage (pkgs.fetchFromGitHub {
      owner = "viicslen";
      repo = "worktrees";
      rev = "main";  # or a specific commit/tag
      hash = "";     # nix will tell you the correct hash
    }) { })
  ];
}
```

**Home Manager:**

```nix
{
  home.packages = [
    (pkgs.callPackage (pkgs.fetchFromGitHub {
      owner = "viicslen";
      repo = "worktrees";
      rev = "main";  # or a specific commit/tag
      hash = "";     # nix will tell you the correct hash
    }) { })
  ];
}
```

**Flake-based NixOS/Home Manager:**

Add to your flake inputs:

```nix
{
  inputs = {
    worktree-manager.url = "github:viicslen/worktree-manager";
  };
}
```

Then reference it in your packages:

```nix
# NixOS
environment.systemPackages = [ inputs.worktrees.packages.${system}.default ];

# Home Manager
home.packages = [ inputs.worktrees.packages.${system}.default ];
```

### Pre-built Binaries

Download pre-built binaries from the [releases page](https://github.com/viicslen/worktrees/releases):

**Linux / macOS:**

```bash
# Download and extract (replace VERSION and PLATFORM with your target)
curl -L https://github.com/viicslen/worktrees/releases/latest/download/wtm-linux-amd64.tar.gz | tar xz

# Move to PATH
sudo mv wtm /usr/local/bin/
```

Available builds:

- `wtm-linux-amd64.tar.gz` - Linux x86_64
- `wtm-linux-arm64.tar.gz` - Linux ARM64
- `wtm-darwin-amd64.tar.gz` - macOS Intel
- `wtm-darwin-arm64.tar.gz` - macOS Apple Silicon
- `wtm-windows-amd64.zip` - Windows x86_64

### Build from Source

```bash
git clone https://github.com/viicslen/worktrees.git
cd worktrees
go build -o wtm
```

Move the binary to your PATH:

```bash
sudo mv wtm /usr/local/bin/
```

### Verify Installation

```bash
wtm --help
```

## Core Concepts

### Bare Repository

Worktree Manager uses a bare repository as the central hub for all your worktrees. A bare repository contains only the git metadata without a working directory, making it ideal for managing multiple worktrees.

### Worktree Structure

The tool organizes worktrees in a specific structure:

```
<bare-repo>/
├── tree/
│   ├── main/           # Inactive branch worktrees
│   ├── feature-1/
│   └── develop/
├── workspace/          # Currently active worktree
└── shared/             # Persisted files shared across worktrees
    ├── .env
    ├── node_modules/
    └── config.json
```

### Workspace Directory

The `workspace` directory is your primary working directory where your IDE should be opened. The `switch` command seamlessly moves branches in and out of this directory without requiring you to close your IDE or change directories.

## Commands

### clone

Clone a repository as a bare repository configured for worktree usage.

**Usage:**

```bash
wtm clone <repo-url> [directory]
```

**What it does:**

1. Clones the repository as a bare repository
2. Configures the fetch refspec to fetch all remote branches
3. Sets up the repository for optimal worktree management

**Examples:**

```bash
# Clone to inferred directory name
wtm clone https://github.com/user/repo.git

# Clone to specific directory
wtm clone https://github.com/user/repo.git my-project
```

**Output:**

- Creates a bare repository in `<directory>` or `<repo-name>` if directory is not specified
- Repository is ready for worktree operations

---

### checkout

Create a new worktree for a branch or commit in the `tree/<name>` directory.

**Usage:**

```bash
wtm checkout <commitish>
```

**What it does:**

1. Verifies you're in a bare repository
2. Creates a `tree` directory if it doesn't exist
3. Creates a new worktree at `tree/<sanitized-name>`
4. Sanitizes branch names by replacing slashes with dashes

**Examples:**

```bash
# Checkout main branch
wtm checkout main
# Creates: tree/main

# Checkout feature branch
wtm checkout feature/new-ui
# Creates: tree/feature-new-ui

# Checkout specific commit
wtm checkout abc123
# Creates: tree/abc123
```

**Notes:**

- Must be run from the bare repository root
- Will fail if worktree already exists
- Branch names with slashes are converted to use dashes for directory names

---

### switch

Switch the active workspace to a different branch or commit without closing your IDE.

**Usage:**

```bash
wtm switch <branch|commit>
```

**What it does:**

1. Moves current `workspace` to `tree/<current-branch>`
2. Either moves `tree/<target-branch>` to `workspace` or creates new worktree at `workspace`
3. Maintains git worktree metadata throughout the operation

**Examples:**

```bash
# Switch to develop branch
wtm switch develop

# Switch to feature branch
wtm switch feature/api-refactor

# Switch to specific commit
wtm switch abc123
```

**Can be run from:**

- The bare repository root
- The `workspace` directory

**Cannot be run from:**

- Inside `tree/<branch>` subdirectories (these are for storage only)

**Workflow:**

1. Current workspace (e.g., `main`) is moved to `tree/main`
2. Target branch (e.g., `develop`) is:
   - Moved from `tree/develop` to `workspace` if it exists
   - Created as a new worktree at `workspace` if it doesn't exist
3. Your IDE continues to see the `workspace` directory with updated content

**Notes:**

- If running from `workspace`, you may need to reload files in your IDE
- The command handles both filesystem moves and git metadata updates
- Automatically creates `tree` directory if needed

---

### persist

Manage files that should be shared across all worktrees.

#### persist add

Copy a file or directory to shared storage.

**Usage:**

```bash
wtm persist add <file|dir>
```

**What it does:**

1. Copies the specified file/directory to `shared/<path>`
2. Preserves the relative path structure
3. Maintains file permissions

**Examples:**

```bash
# Persist environment file
wtm persist add .env

# Persist configuration
wtm persist add src/config.json

# Persist node_modules to avoid reinstalling
wtm persist add node_modules
```

**Notes:**

- Must be run from within a worktree (not the bare repository)
- Will fail if file already exists in shared storage
- Use relative or absolute paths

#### persist list

List all persisted files and directories.

**Usage:**

```bash
wtm persist list
```

**Output:**

```
Persisted files in shared/:

  📁 node_modules/
  📄 .env (245 B)
  📄 src/config.json (1.2 KB)
```

#### persist remove

Remove a file or directory from shared storage.

**Usage:**

```bash
wtm persist remove <file|dir>
```

**Examples:**

```bash
# Remove persisted environment file
wtm persist remove .env

# Remove persisted directory
wtm persist remove node_modules
```

---

### restore

Restore persisted files from shared storage to the current worktree.

**Usage:**

```bash
wtm restore <file|dir> [flags]
```

**Flags:**

- `--link`: Create a symlink instead of copying (saves disk space)
- `--to <path>`: Restore to a different path than the original
- `--force`: Overwrite existing files
- `--all`: Restore all persisted files at once

**What it does:**

1. Copies or links files from `shared/<path>` to the worktree
2. Restores to original relative path by default
3. Creates parent directories as needed

**Examples:**

```bash
# Copy .env to current worktree
wtm restore .env

# Symlink node_modules (recommended for large directories)
wtm restore node_modules --link

# Restore to custom location
wtm restore config.json --to custom/path/config.json

# Overwrite existing file
wtm restore .env --force

# Restore everything at once
wtm restore --all

# Restore all and overwrite existing
wtm restore --all --force
```

**Copy vs. Link:**

- **Copy** (default): Creates an independent copy; changes won't affect other worktrees
- **Link** (`--link`): Creates a symlink; all worktrees share the same file/directory

**Use symlinks for:**

- Large directories like `node_modules`, `vendor`, or build artifacts
- Files that should be truly shared (e.g., shared cache)

**Use copies for:**

- Files you might modify per-branch (e.g., `.env` with different values)
- Small files where disk space isn't a concern

**Docker Development Caveat:**

If you are using Docker for local development (e.g., mounting your workspace directory into a container), avoid using the `--link` option. Symlinks created on the host may not resolve correctly inside the container, especially when the symlink points to paths outside the mounted volume. In Docker development workflows, always use the default copy behavior instead of `--link`.

---

## Workflow Examples

### Initial Setup

```bash
# Clone your repository
wtm clone https://github.com/user/my-app.git

# Navigate into the repository
cd my-app

# Create your first workspace
wtm checkout main

# Switch to workspace directory
cd workspace

# Install dependencies
npm install

# Persist node_modules to avoid reinstalling
wtm persist add node_modules

# Persist environment file
wtm persist add .env
```

### Daily Development Workflow

```bash
# Working on main in workspace/
cd /path/to/my-app/workspace

# Need to switch to feature branch
wtm switch feature/new-feature

# If branch didn't exist, restore shared dependencies
wtm restore node_modules --link
wtm restore .env

# Work on feature...
git add .
git commit -m "Add new feature"

# Switch back to main
wtm switch main

# Your IDE still points to workspace/, just reload files
```

### Working with Multiple Features

```bash
# In bare repository root
cd /path/to/my-app

# Checkout multiple branches for parallel work
wtm checkout feature/api-changes
wtm checkout feature/ui-updates
wtm checkout bugfix/critical-issue

# Now you have:
# tree/main/
# tree/feature-api-changes/
# tree/feature-ui-updates/
# tree/bugfix-critical-issue/

# Switch between them as needed
wtm switch feature/api-changes    # Workspace now has api-changes
wtm switch feature/ui-updates     # Workspace now has ui-updates
```

### Managing Shared Files

```bash
# Create a configuration all branches need
cd /path/to/my-app/workspace
echo "API_KEY=abc123" > .env

# Persist it
wtm persist add .env

# Switch to another branch
wtm switch develop

# Restore the shared config
wtm restore .env

# Or restore everything at once
wtm restore --all
```

## Directory Structure

After setting up and using Worktree Manager, your repository structure will look like:

```
my-app/                          # Bare repository root
├── branches/                    # Git metadata
├── config                       # Git configuration
├── description                  # Repository description
├── HEAD                         # Current HEAD reference
├── hooks/                       # Git hooks
├── info/                        # Git info
├── objects/                     # Git objects
├── packed-refs                  # Packed references
├── refs/                        # Git references
├── worktrees/                   # Git worktree metadata
├── tree/                        # Inactive worktrees
│   ├── main/                    # Branch: main
│   │   ├── src/
│   │   ├── package.json
│   │   └── ...
│   ├── feature-api-changes/     # Branch: feature/api-changes
│   │   ├── src/
│   │   └── ...
│   └── develop/                 # Branch: develop
│       └── ...
├── workspace/                   # Active worktree (open in IDE)
│   ├── src/
│   ├── package.json
│   ├── .env                     # Restored from shared/
│   └── node_modules/            # Symlinked to shared/
└── shared/                      # Persisted files
    ├── .env                     # Shared environment file
    ├── node_modules/            # Shared dependencies
    └── config.json              # Shared configuration
```

## Use Cases

### 1. Code Review Workflow

```bash
# Working on feature branch in workspace
# PR comes in needing review

wtm switch pr-123
wtm restore --all --link
# Review code, test changes

# Switch back to your work
wtm switch feature/my-work
```

### 2. Hotfix on Production

```bash
# Working on develop branch
# Production issue reported

wtm switch main
wtm restore --all
# Fix issue, test, commit

# Back to development
wtm switch develop
```

### 3. Dependency Sharing

```bash
# Large node_modules or vendor directories
wtm persist add node_modules

# In any worktree
wtm restore node_modules --link
# All worktrees share same dependencies, saving disk space
```

### 4. Environment Configuration

```bash
# Different .env per branch but same structure
wtm persist add .env.example

# In each worktree
wtm restore .env.example --to .env
# Customize per branch as needed
```

### 5. Multiple Simultaneous Branches

```bash
# Keep several branches ready
wtm checkout main
wtm checkout develop
wtm checkout staging
wtm checkout feature/big-refactor

# Instant switching without checkout delays
wtm switch staging        # Test staging
wtm switch develop        # Continue development
wtm switch feature/big-refactor  # Work on long-term feature
```

## Benefits

1. **IDE Persistence**: Your IDE stays open in the `workspace` directory while branches change underneath
2. **Fast Branch Switching**: No git checkout delays; instant filesystem swaps
3. **Parallel Work**: Keep multiple branches ready simultaneously in `tree/`
4. **Shared Dependencies**: Avoid reinstalling dependencies with `persist` and `restore --link`
5. **Disk Space Optimization**: Symlink large directories across worktrees
6. **Clean Workflow**: Structured approach to managing multiple concurrent tasks

## Requirements

- Git 2.5+ (for worktree support)
- Go 1.25.1+ (for building from source)

## License

This project is licensed under the terms specified in the LICENSE file.
