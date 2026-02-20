#!/usr/bin/env bash
# gh-autoprofile direnv shell library
# Installed to ~/.config/direnv/lib/gh-autoprofile.sh by `gh autoprofile setup`
#
# Provides the use_gh_autoprofile function for .envrc files.
# Usage in .envrc:
#   use_gh_autoprofile <gh-username> [git-email] [git-name] [ssh-key-path]

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

  # Resolve token from gh keyring (does NOT change active account)
  local token
  token=$(gh auth token --user "$user" 2>/dev/null)
  if [[ -z "$token" ]]; then
    log_error "gh-autoprofile: no token found for user '$user'. Run: gh auth login"
    return 1
  fi

  # Export tokens — covers gh CLI, GitHub Actions, Terraform, etc.
  export GH_TOKEN="$token"
  export GITHUB_TOKEN="$token"

  # Git identity
  if [[ -n "$git_email" ]]; then
    export GIT_AUTHOR_EMAIL="$git_email"
    export GIT_COMMITTER_EMAIL="$git_email"
  fi

  if [[ -n "$git_name" ]]; then
    export GIT_AUTHOR_NAME="$git_name"
    export GIT_COMMITTER_NAME="$git_name"
  fi

  # SSH key selection
  if [[ -n "$ssh_key" ]]; then
    if [[ ! -f "$ssh_key" ]]; then
      log_error "gh-autoprofile: SSH key not found: $ssh_key"
      return 1
    fi
    export GIT_SSH_COMMAND="ssh -i $ssh_key -o IdentitiesOnly=yes"
  fi

  log_status "gh-autoprofile" "activated '$user'"
}
