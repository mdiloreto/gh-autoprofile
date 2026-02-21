# gh-autoprofile

**Automatic GitHub profile switching per directory** — pin an account, `cd` in, done.

A [GitHub CLI](https://cli.github.com/) extension that pins a GitHub account to a directory using [direnv](https://direnv.net/). When you `cd` into a pinned directory, the correct GitHub token, git identity, and SSH key are automatically activated — no manual switching, no global state mutation.

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

gh-autoprofile supports two token injection modes:

### Wrapper mode (default — secure)

1. `direnv` exports only a non-sensitive marker (`GH_AUTOPROFILE_USER=<name>`) into the shell
2. A shell hook creates `gh()` and `git()` wrapper functions on each prompt
3. When you run `gh` or `git`, the wrapper calls `gh auth token --user <name>` (~30ms), injects `GH_TOKEN` **only into the child process**, then runs the real command
4. The token **never sits in the parent shell's environment** — similar to how `aws-vault exec` works

### Export mode (opt-in — for third-party tools)

1. `direnv` exports `GH_TOKEN` and `GITHUB_TOKEN` directly into the shell environment
2. Third-party tools that read these env vars (Terraform, `act`, CI tools) work out of the box
3. Use `--export-token` flag when pinning directories that need this

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

This validates prerequisites, installs the direnv shell library, installs the shell hook for wrapper mode, and configures your shell RC file (`~/.zshrc` or `~/.bashrc`).

**Restart your shell** (or `source ~/.zshrc`) after setup.

### Upgrade (v0.2+)

After upgrading, run migration once to apply security defaults to existing pins:

```bash
gh extension upgrade mdiloreto/gh-autoprofile
gh autoprofile setup --migrate
# restart shell (or source your rc file)
```

Migration does this automatically:

- refreshes installed shell library and wrapper hook
- rewrites managed `.envrc` blocks with current templates and `0600` permissions
- backfills missing pin mode to `wrapper`
- runs `direnv allow` for updated pinned directories

## Usage

### Pin an account to a directory

```bash
# Basic — just the GitHub account (wrapper mode, token never in env)
gh autoprofile pin alice --dir ~/personal-projects

# With git identity
gh autoprofile pin bob-work --dir ~/work \
  --git-email bob@company.com \
  --git-name "Bob Smith"

# With SSH key
gh autoprofile pin alice-freelance --dir ~/freelance \
  --git-email alice@freelance.com \
  --ssh-key ~/.ssh/id_freelance

# Export mode — for directories where Terraform, act, or other tools need GH_TOKEN
gh autoprofile pin bob-work --dir ~/infra --export-token
```

### List all pins

```bash
gh autoprofile list
```

```
DIRECTORY            ACCOUNT          MODE      GIT EMAIL             GIT NAME    SSH KEY
---------            -------          ----      ---------             --------    -------
  ~/personal         alice            wrapper   -                     -           -
* ~/work             bob-work         wrapper   bob@company.com       Bob Smith   -
  ~/freelance        alice-freelance  wrapper   alice@freelance.com   -           ~/.ssh/id_freelance
  ~/infra            bob-work         export    bob@company.com       -           -

4 pin(s) total. (* = current directory)
```

### Check current status

```bash
gh autoprofile status
```

```
Directory: /home/user/work

  Pinned account:   bob-work
  Token mode:       wrapper
  Pinned email:     bob@company.com
  Pinned name:      Bob Smith

  Environment:
    GH_AUTOPROFILE_USER:  bob-work
    GH_TOKEN:             (not set)
    GITHUB_TOKEN:         (not set)
    GIT_AUTHOR_EMAIL:     bob@company.com
    GIT_AUTHOR_NAME:      Bob Smith

  Active gh user:   alice (github.com)

  Profile is active (wrapper mode). Token injected per-command only.
```

### Validate installation health

```bash
gh autoprofile doctor
# or auto-fix:
gh autoprofile doctor --fix
```

### Remove a pin

```bash
gh autoprofile unpin ~/work-project
```

## What gets set

### Wrapper mode (default)

| Variable | Purpose |
|---|---|
| `GH_AUTOPROFILE_USER` | Non-sensitive marker (read by shell hook) |
| `GIT_AUTHOR_EMAIL` | Git commit author email |
| `GIT_COMMITTER_EMAIL` | Git commit committer email |
| `GIT_AUTHOR_NAME` | Git commit author name |
| `GIT_COMMITTER_NAME` | Git commit committer name |
| `GIT_SSH_COMMAND` | Per-directory SSH key selection |

`GH_TOKEN` is injected **only into `gh` and `git` child processes** by the wrapper functions — it never appears in `env` or `printenv`.

### Export mode (`--export-token`)

All of the above, plus:

| Variable | Purpose |
|---|---|
| `GH_TOKEN` | GitHub CLI authentication (highest precedence) |
| `GITHUB_TOKEN` | Third-party tools (Terraform, GitHub Actions, etc.) |

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
~/.config/gh-autoprofile/pins.yml       # Pin registry (source of truth)
~/.config/gh-autoprofile/hook.sh        # Shell hook (wrapper mode — creates gh()/git() functions)
~/.config/direnv/lib/gh-autoprofile.sh  # Direnv library (use_gh_autoprofile functions)
~/your-project/.envrc                   # Managed block between markers
```

The `.envrc` block is managed between `# gh-autoprofile:start` and `# gh-autoprofile:end` markers. Existing `.envrc` content is preserved.

### Security model

- **Wrapper mode** (default): Tokens are read from the keyring on each `gh`/`git` invocation and exist only for the lifetime of that child process. A compromised child process cannot leak the token to siblings. This is the same pattern used by `aws-vault exec`.
- **Export mode**: Tokens live in the shell environment for the duration of the directory session. Any child process can read them. Use only when third-party tools require `GH_TOKEN`/`GITHUB_TOKEN` as env vars.
- **File permissions**: `pins.yml` is written `0600`, config directory is `0700`.
- **Shell quoting**: All values written to `.envrc` use POSIX single-quote escaping to prevent injection.

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
