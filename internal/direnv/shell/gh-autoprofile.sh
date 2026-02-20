#!/usr/bin/env bash
# gh-autoprofile direnv shell library
# Installed to ~/.config/direnv/lib/gh-autoprofile.sh by `gh autoprofile setup`
#
# Provides the use_gh_autoprofile function for .envrc files.
#
# Wrapper mode (default):
#   use_gh_autoprofile <user> [git-email] [git-name] [ssh-key-path]
#   Exports only GH_AUTOPROFILE_USER (non-sensitive marker). The shell hook
#   creates gh()/git() functions that inject the token per-invocation.
#
# Export mode (opt-in per pin with --export-token):
#   use_gh_autoprofile_export <user> [git-email] [git-name] [ssh-key-path]
#   Exports GH_TOKEN and GITHUB_TOKEN directly into the shell environment.

# --- wrapper mode (default) ---------------------------------------------------

use_gh_autoprofile() {
  local user="$1"
  local git_email="${2:-}"
  local git_name="${3:-}"
  local ssh_key="${4:-}"

  if [[ -z "$user" ]]; then
    log_error "gh-autoprofile: usage: use_gh_autoprofile <gh-username> [git-email] [git-name] [ssh-key-path]"
    return 1
  fi

  # Verify gh CLI is available
  if ! command -v gh &>/dev/null; then
    log_error "gh-autoprofile: gh CLI not found. Install from https://cli.github.com"
    return 1
  fi

  # Validate that the user has a token (fail fast), but do NOT export it.
  local token
  token=$(command gh auth token --user "$user" 2>/dev/null)
  if [[ -z "$token" ]]; then
    log_error "gh-autoprofile: no token found for user '$user'. Run: gh auth login"
    return 1
  fi

  # Export only the non-sensitive marker. The shell hook reads this to
  # create per-invocation wrapper functions.
  export GH_AUTOPROFILE_USER="$user"

  _gh_autoprofile_identity "$git_email" "$git_name" "$ssh_key"

  log_status "gh-autoprofile" "activated '$user' (wrapper mode)"
}

# --- export mode (opt-in) -----------------------------------------------------

use_gh_autoprofile_export() {
  local user="$1"
  local git_email="${2:-}"
  local git_name="${3:-}"
  local ssh_key="${4:-}"

  if [[ -z "$user" ]]; then
    log_error "gh-autoprofile: usage: use_gh_autoprofile_export <gh-username> [git-email] [git-name] [ssh-key-path]"
    return 1
  fi

  if ! command -v gh &>/dev/null; then
    log_error "gh-autoprofile: gh CLI not found. Install from https://cli.github.com"
    return 1
  fi

  local token
  token=$(command gh auth token --user "$user" 2>/dev/null)
  if [[ -z "$token" ]]; then
    log_error "gh-autoprofile: no token found for user '$user'. Run: gh auth login"
    return 1
  fi

  # Export tokens into environment — third-party tools can read them.
  export GH_TOKEN="$token"
  export GITHUB_TOKEN="$token"

  # Also set the marker so the shell hook knows NOT to create wrappers
  # (the env vars are already available).
  export GH_AUTOPROFILE_USER="$user"

  _gh_autoprofile_identity "$git_email" "$git_name" "$ssh_key"

  log_status "gh-autoprofile" "activated '$user' (export mode)"
}

# --- shared helpers -----------------------------------------------------------

_gh_autoprofile_identity() {
  local git_email="$1"
  local git_name="$2"
  local ssh_key="$3"

  # Git identity
  if [[ -n "$git_email" ]]; then
    export GIT_AUTHOR_EMAIL="$git_email"
    export GIT_COMMITTER_EMAIL="$git_email"
  fi

  if [[ -n "$git_name" ]]; then
    export GIT_AUTHOR_NAME="$git_name"
    export GIT_COMMITTER_NAME="$git_name"
  fi

  # SSH key selection (properly quoted to prevent command injection)
  if [[ -n "$ssh_key" ]]; then
    if [[ ! -f "$ssh_key" ]]; then
      log_error "gh-autoprofile: SSH key not found: $ssh_key"
      return 1
    fi
    export GIT_SSH_COMMAND="ssh -i '${ssh_key}' -o IdentitiesOnly=yes"
  fi
}
