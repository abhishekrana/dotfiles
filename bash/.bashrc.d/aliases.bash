# Git
alias gs='git status'
alias gd='git diff'
alias gl='git log'
alias gb='git branch --sort=committerdate --format="%(refname:short) %(committerdate:relative)" | tail -20 | awk -F" " "{name=\$1; \$1=\"\"; printf \"%-50s (%s)\\n\", name, substr(\$0,2)}" && echo "" && echo "* $(git branch --show-current)"'
alias gdd='nvim -c "DiffviewOpen"'
alias mrd='nvim -c "DiffviewOpen origin/main...HEAD"'

# Docker
alias docker-stop-all='docker stop $(docker ps -a -q)'
alias docker-rm-all='docker rm $(docker ps -a -q)'

# Editor
alias vim='nvim'
