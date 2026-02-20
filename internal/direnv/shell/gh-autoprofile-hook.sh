#!/usr/bin/env bash
# gh-autoprofile shell hook — process-scoped token injection
# Installed to ~/.config/gh-autoprofile/hook.sh by `gh autoprofile setup`
# Sourced from ~/.zshrc or ~/.bashrc.
#
# When GH_AUTOPROFILE_USER is set (by direnv), this hook creates gh() and
# git() wrapper functions that inject GH_TOKEN only into the child process
# for ~30 ms — the token never lives in the parent shell's environment.
#
# When GH_AUTOPROFILE_USER is unset (leaving the directory), the wrapper
# functions are removed and the original commands are restored.

# Guard: only load once per shell session.
[[ -n "$_GH_AUTOPROFILE_HOOK_LOADED" ]] && return 0
_GH_AUTOPROFILE_HOOK_LOADED=1

# Track the last user we created wrappers for, so we don't recreate them
# on every prompt.
_gh_autoprofile_last_user=""
_gh_autoprofile_last_has_token=""

_gh_autoprofile_hook() {
  local current_user="${GH_AUTOPROFILE_USER:-}"
  local has_token="${GH_TOKEN:+1}"

  # Nothing changed — skip.
  if [[ "$current_user" == "$_gh_autoprofile_last_user" && "$has_token" == "$_gh_autoprofile_last_has_token" ]]; then
    return 0
  fi

  if [[ -n "$current_user" && -z "${GH_TOKEN:-}" ]]; then
    # Wrapper mode: GH_AUTOPROFILE_USER is set but GH_TOKEN is NOT.
    # Create wrapper functions that inject token per-invocation.
    # `command` bypasses the function and calls the real binary.

    gh() {
      local _token
      _token=$(command gh auth token --user "$GH_AUTOPROFILE_USER" 2>/dev/null)
      if [[ -n "$_token" ]]; then
        GH_TOKEN="$_token" command gh "$@"
      else
        # Fallback: run without injection (will use default gh account)
        echo "gh-autoprofile: warning: could not resolve token for '$GH_AUTOPROFILE_USER'" >&2
        command gh "$@"
      fi
    }

    git() {
      # Only inject token for HTTPS credential operations; SSH uses
      # GIT_SSH_COMMAND which direnv already sets.
      local _token
      _token=$(command gh auth token --user "$GH_AUTOPROFILE_USER" 2>/dev/null)
      if [[ -n "$_token" ]]; then
        GH_TOKEN="$_token" command git "$@"
      else
        command git "$@"
      fi
    }

    _gh_autoprofile_last_user="$current_user"
    _gh_autoprofile_last_has_token=""

  elif [[ -n "$current_user" && -n "${GH_TOKEN:-}" ]]; then
    # Export mode: GH_AUTOPROFILE_USER is set AND GH_TOKEN is present.
    # Tokens are already in the environment — no wrapper functions needed.
    # Remove any stale wrappers from a previous wrapper-mode directory.
    unset -f gh 2>/dev/null
    unset -f git 2>/dev/null
    _gh_autoprofile_last_user="$current_user"
    _gh_autoprofile_last_has_token="1"

  else
    # User left a pinned directory — remove wrapper functions.
    unset -f gh 2>/dev/null
    unset -f git 2>/dev/null
    _gh_autoprofile_last_user=""
    _gh_autoprofile_last_has_token=""
  fi
}

# Install the hook into the shell's prompt cycle.
if [[ -n "$ZSH_VERSION" ]]; then
  # zsh: use precmd hook array (works alongside oh-my-zsh).
  autoload -Uz add-zsh-hook
  add-zsh-hook precmd _gh_autoprofile_hook
elif [[ -n "$BASH_VERSION" ]]; then
  # bash: append to PROMPT_COMMAND.
  if [[ -z "$PROMPT_COMMAND" ]]; then
    PROMPT_COMMAND="_gh_autoprofile_hook"
  else
    PROMPT_COMMAND="_gh_autoprofile_hook;${PROMPT_COMMAND}"
  fi
fi
