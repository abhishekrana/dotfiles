# Programmable completion
if ! shopt -oq posix; then
    if [ -f /usr/share/bash-completion/bash_completion ]; then
        . /usr/share/bash-completion/bash_completion
    elif [ -f /etc/bash_completion ]; then
        . /etc/bash_completion
    fi
fi

# Tool-specific completions
[ -f "$HOME/.bash-completion/completions/task.bash" ] && source "$HOME/.bash-completion/completions/task.bash"
[ -f "$HOME/.local/share/bash-completion/completions/k3d/k3d_completion.sh" ] && source "$HOME/.local/share/bash-completion/completions/k3d/k3d_completion.sh"
