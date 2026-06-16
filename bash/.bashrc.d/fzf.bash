command -v fzf &>/dev/null || return

# Use fd for faster file listing (respects .gitignore)
if command -v fd &>/dev/null; then
    export FZF_DEFAULT_COMMAND='fd --type f --hidden --exclude .git'
    export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
    export FZF_ALT_C_COMMAND='fd --type d --hidden --exclude .git'
fi

# Sensible default look + preview toggle (no hardcoded palette so terminal
# theme drives colors -- dark or light).
export FZF_DEFAULT_OPTS='--height=80% --layout=reverse --border --bind ctrl-/:toggle-preview'

# Pick the right clipboard command for this OS so the same FZF_*_OPTS work on Linux/macOS.
if command -v pbcopy &>/dev/null; then
    _FZF_CLIP_CMD='pbcopy'
elif command -v wl-copy &>/dev/null; then
    _FZF_CLIP_CMD='wl-copy'
elif command -v xclip &>/dev/null; then
    _FZF_CLIP_CMD='xclip -selection clipboard'
fi

# Ctrl-T: file picker with bat preview; Ctrl-Y copies file contents to the clipboard
if command -v bat &>/dev/null && [ -n "${_FZF_CLIP_CMD:-}" ]; then
    export FZF_CTRL_T_OPTS="
      --preview 'bat --color=always --style=numbers --line-range=:200 {}'
      --bind 'ctrl-y:execute-silent($_FZF_CLIP_CMD < {})+abort'
    "
fi

# Ctrl-R: history search; Ctrl-Y copies the command (fields 2..) without running it
if [ -n "${_FZF_CLIP_CMD:-}" ]; then
    export FZF_CTRL_R_OPTS="
      --bind 'ctrl-y:execute-silent(echo -n {2..} | $_FZF_CLIP_CMD)+abort'
    "
fi

if [ -n "${ZSH_VERSION:-}" ]; then
    eval "$(fzf --zsh)"
else
    eval "$(fzf --bash)"
fi

# ff: fuzzy find → enter=open in nvim, ctrl-y=copy path
ff() {
    local out key file
    out=$(fd --type f --hidden --exclude .git --absolute-path | fzf \
        --preview 'bat --color=always {}' \
        --header 'enter: open in nvim  |  ctrl-y: copy path' \
        --expect=ctrl-y)
    key=$(head -1 <<< "$out")
    file=$(tail -1 <<< "$out")
    [ -z "$file" ] && return
    if [ "$key" = "ctrl-y" ]; then
        echo "$file"
    else
        nvim "$file"
    fi
}

# rfv: live ripgrep through fzf, Enter opens nvim at the matched line.
# Usage: rfv [initial-query]
if command -v rg &>/dev/null && command -v bat &>/dev/null; then
    rfv() {
        local rg_prefix='rg --column --line-number --no-heading --color=always --smart-case'
        fzf --ansi --disabled --query "${1:-}" \
            --bind "start:reload:$rg_prefix {q} || :" \
            --bind "change:reload:sleep 0.1; $rg_prefix {q} || :" \
            --bind "enter:become(nvim {1} +{2})" \
            --delimiter=: \
            --preview 'bat --color=always --style=numbers --highlight-line {2} {1}' \
            --preview-window 'up,60%,border-bottom,+{2}+3/3'
    }
fi
