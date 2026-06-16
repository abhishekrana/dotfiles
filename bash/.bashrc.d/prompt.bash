# PS1 uses bash-only escape syntax (\u, \h, \w, \[\]). zsh has its own (%n/%m/%~)
# and would print this literally. Skip when sourced under zsh — the default
# zsh prompt stays in place.
[ -z "${BASH_VERSION:-}" ] && return

_git_branch() {
    git branch 2>/dev/null | sed -e '/^[^*]/d' -e 's/* \(.*\)/(\1)/'
}
PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\[\033[01;33m\]$(_git_branch)\[\033[00m\]\$ '
