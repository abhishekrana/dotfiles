command -v direnv &>/dev/null || return

export DIRENV_LOG_FORMAT=""
eval "$(direnv hook bash)"

# Show direnv-activated venv in prompt
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
