command -v fzf &>/dev/null || return

# Use fd for faster file listing (respects .gitignore)
if command -v fd &>/dev/null; then
    export FZF_DEFAULT_COMMAND='fd --type f --hidden --exclude .git'
    export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
    export FZF_ALT_C_COMMAND='fd --type d --hidden --exclude .git'
fi

# Default bat theme for preview subprocesses (bat config isn't always picked up there).
# theme.bash loads after this and overrides BAT_THEME with the selected flavor.
export BAT_THEME="${BAT_THEME:-Solarized (light)}"

# Global look + preview toggle. The color block defaults to Solarized Light; the
# `theme` switcher overrides it via ~/.config/theme/fzf.sh so new shells follow the flavor.
_fzf_color='--color=light --color=fg:#586e75,bg:#fdf6e3,hl:#268bd2 --color=fg+:#073642,bg+:#eee8d5,hl+:#cb4b16 --color=info:#b58900,prompt:#859900,pointer:#dc322f --color=marker:#dc322f,spinner:#b58900,header:#2aa198 --color=border:#93a1a1,gutter:#fdf6e3'
[ -f ~/.config/theme/fzf.sh ] && . ~/.config/theme/fzf.sh
export FZF_DEFAULT_OPTS="
  --height=80% --layout=reverse --border
  --bind ctrl-/:toggle-preview
  $_fzf_color
"
unset _fzf_color

# Ctrl-T: file picker with bat preview; Ctrl-Y copies file contents to wl-clipboard
if command -v bat &>/dev/null && command -v wl-copy &>/dev/null; then
    export FZF_CTRL_T_OPTS="
      --preview 'bat --color=always --style=numbers --line-range=:200 {}'
      --bind 'ctrl-y:execute-silent(wl-copy < {})+abort'
    "
fi

# Ctrl-R: history search; Ctrl-Y copies the command (fields 2..) without running it
if command -v wl-copy &>/dev/null; then
    export FZF_CTRL_R_OPTS="
      --bind 'ctrl-y:execute-silent(echo -n {2..} | wl-copy)+abort'
    "
fi

eval "$(fzf --bash)"

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
