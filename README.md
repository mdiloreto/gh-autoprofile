# gh-autoprofile

**Automatic GitHub profile switching per directory** — pin an account, `cd` in, done.

A [GitHub CLI](https://cli.github.com/) extension that pins a GitHub account to a directory using [direnv](https://direnv.net/). When you `cd` into a pinned directory, the correct `GH_TOKEN`, git identity, and SSH key are automatically exported — no manual switching, no global state mutation.

## Why

GitHub CLI [added multi-account support](https://github.com/cli/cli/blob/trunk/docs/multiple-accounts.md) in v2.40.0, but explicitly excluded automatic per-directory switching. The archived [gh-profile](https://github.com/gabe565/gh-profile) extension was the only tool that filled this gap — `gh-autoprofile` picks up where it left off.

### The problem

```bash
# You have 3 GitHub accounts...
gh auth status
# alice (personal), bob-work (company), alice-freelance (freelance)

# But when you cd into a project, you have to manually switch:
cd ~/work-project
gh auth switch --user bob-work   # tedious, mutates global state
```

### The solution

```bash
# Pin once:
gh autoprofile pin bob-work --dir ~/work-project --git-email bob@company.com

# Now just cd and work:
cd ~/work-project
gh api /user --jq .login    # -> bob-work (automatic!)
git config user.email        # -> bob@company.com
```

## How it works

1. `gh auth token --user <name>` reads a token from `gh`'s keyring without changing the active account
2. `direnv` scopes environment variables to the directory (and unloads them when you leave)
3. `gh-autoprofile` writes a managed `.envrc` block that calls `use_gh_autoprofile <user>`
4. The shell function exports `GH_TOKEN`, `GITHUB_TOKEN`, git identity vars, and optionally `GIT_SSH_COMMAND`

**Zero global state mutation.** Each terminal session gets its own isolated environment. Safe for multi-terminal workflows.

## Installation

### Prerequisites

- [GitHub CLI](https://cli.github.com/) v2.40.0+ with multiple accounts logged in
- [direnv](https://direnv.net/) with shell hook configured

### Install the extension

```bash
gh extension install mdiloreto/gh-autoprofile
```

### Run setup

```bash
gh autoprofile setup
```

This validates prerequisites and installs the direnv shell library.

## Usage

### Pin an account to a directory

```bash
# Basic — just the GitHub account
gh autoprofile pin alice --dir ~/personal-projects

# With git identity
gh autoprofile pin bob-work --dir ~/work \
  --git-email bob@company.com \
  --git-name "Bob Smith"

# With SSH key
gh autoprofile pin alice-freelance --dir ~/freelance \
  --git-email alice@freelance.com \
  --ssh-key ~/.ssh/id_freelance
```

### List all pins

```bash
gh autoprofile list
```

```
DIRECTORY            ACCOUNT          GIT EMAIL             GIT NAME    SSH KEY
---------            -------          ---------             --------    -------
  ~/personal         alice            -                     -           -
* ~/work             bob-work         bob@company.com       Bob Smith   -
  ~/freelance        alice-freelance  alice@freelance.com   -           ~/.ssh/id_freelance

3 pin(s) total. (* = current directory)
```

### Check current status

```bash
gh autoprofile status
```

```
Directory: /home/user/work

  Pinned account:   bob-work
  Pinned email:     bob@company.com
  Pinned name:      Bob Smith

  Environment:
    GH_TOKEN:             gho_****Xx4z
    GITHUB_TOKEN:         gho_****Xx4z
    GIT_AUTHOR_EMAIL:     bob@company.com
    GIT_AUTHOR_NAME:      Bob Smith

  Active gh user:   alice (github.com)

  Profile is active and GH_TOKEN is set.
```

### Remove a pin

```bash
gh autoprofile unpin ~/work-project
```

## What gets exported

| Variable | Purpose |
|---|---|
| `GH_TOKEN` | GitHub CLI authentication (highest precedence) |
| `GITHUB_TOKEN` | Third-party tools (Terraform, GitHub Actions, etc.) |
| `GIT_AUTHOR_EMAIL` | Git commit author email |
| `GIT_COMMITTER_EMAIL` | Git commit committer email |
| `GIT_AUTHOR_NAME` | Git commit author name |
| `GIT_COMMITTER_NAME` | Git commit committer name |
| `GIT_SSH_COMMAND` | Per-directory SSH key selection |

All variables are scoped to the directory via direnv and automatically unloaded when you `cd` out.

## How it compares

| Tool | Active | direnv | No global mutation | Git identity | SSH keys |
|---|---|---|---|---|---|
| **gh-autoprofile** | Yes | Yes | Yes | Yes | Yes |
| gh-profile (archived) | No | Yes | Yes | No | No |
| gh native `auth switch` | Yes | No | No (mutates global) | No | No |
| gh-switcher (npm) | Yes | No | No (mutates git config) | Partial | No |
| DIY `.envrc` | N/A | Yes | Yes | Manual | Manual |

## Architecture

```
~/.config/gh-autoprofile/pins.yml    # Pin registry (source of truth)
~/.config/direnv/lib/gh-autoprofile.sh  # Shell library (use_gh_autoprofile function)
~/your-project/.envrc                # Managed block between markers
```

The `.envrc` block is managed between `# gh-autoprofile:start` and `# gh-autoprofile:end` markers. Existing `.envrc` content is preserved.

## Development

```bash
# Clone
git clone https://github.com/mdiloreto/gh-autoprofile
cd gh-autoprofile

# Build
go build -o gh-autoprofile ./cmd/gh-autoprofile

# Test
go test ./... -v

# Install locally for testing
cp gh-autoprofile ~/.local/bin/
```

## License

MIT
