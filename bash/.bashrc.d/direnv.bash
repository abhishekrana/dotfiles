command -v direnv &>/dev/null || return

export DIRENV_LOG_FORMAT=""
if [ -n "${ZSH_VERSION:-}" ]; then
    eval "$(direnv hook zsh)"
else
    eval "$(direnv hook bash)"
fi

# Show direnv-activated venv as a prompt prefix. PROMPT_COMMAND is bash-only;
# zsh has its own venv display via VIRTUAL_ENV_PROMPT, so skip there.
if [ -n "${BASH_VERSION:-}" ]; then
    _ORIG_PS1="${PS1}"
    _direnv_ps1_update() {
        local venv=""
        if [[ -n "${VIRTUAL_ENV:-}" ]]; then
            venv="($(basename "$(dirname "$VIRTUAL_ENV")")) "
        fi
        PS1="${venv}${_ORIG_PS1}"
    }
    if [[ -n "$PROMPT_COMMAND" ]]; then
        PROMPT_COMMAND="$PROMPT_COMMAND; _direnv_ps1_update"
    else
        PROMPT_COMMAND="_direnv_ps1_update"
    fi
fi
