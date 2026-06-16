command -v zoxide &>/dev/null || return
if [ -n "${ZSH_VERSION:-}" ]; then
    eval "$(zoxide init --cmd cd zsh)"
else
    eval "$(zoxide init --cmd cd bash)"
fi
