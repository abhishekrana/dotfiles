# uv/uvx shell completions
_dot_shell="bash"
[ -n "${ZSH_VERSION:-}" ] && _dot_shell="zsh"
command -v uv  &>/dev/null && eval "$(uv  generate-shell-completion "$_dot_shell")"
command -v uvx &>/dev/null && eval "$(uvx --generate-shell-completion "$_dot_shell")"
unset _dot_shell

# task (taskfile) completion
[ -f "$HOME/.bash-completion/completions/task.bash" ] && source "$HOME/.bash-completion/completions/task.bash"

# k3d completion
[ -f "$HOME/.local/share/bash-completion/completions/k3d/k3d_completion.sh" ] && source "$HOME/.local/share/bash-completion/completions/k3d/k3d_completion.sh"

# terraform completion
command -v terraform &>/dev/null && complete -C "$HOME/.local/bin/terraform" terraform

# yazi: run with y to cd to the chosen directory on quit
command -v yazi &>/dev/null && function y() {
    local tmp cwd
    tmp="$(mktemp -t yazi-cwd.XXXXXX)"
    yazi "$@" --cwd-file="$tmp"
    if cwd="$(cat -- "$tmp")" && [ -n "$cwd" ] && [ "$cwd" != "$PWD" ]; then
        cd -- "$cwd"
    fi
    rm -f -- "$tmp"
}
