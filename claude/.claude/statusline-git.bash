# Sourced by the Claude Code status line scripts.
#
# git_place <dir> sets $place_root and $place_branch for the checkout holding
# <dir>, and returns 1 when there is none. Pure bash: the status line re-runs on
# a timer and again per subagent row, so it reads .git rather than forking git.

# place_root / place_branch are this fragment's output, set for its callers.
# shellcheck disable=SC2034
git_place() {
    local dir=${1%/} gitdir head
    place_root=
    place_branch=
    [ -n "$dir" ] && [ -d "$dir" ] || return 1

    while [ ! -e "$dir/.git" ] && [ "$dir" != / ]; do
        dir=${dir%/*}
        [ -n "$dir" ] || dir=/
    done
    [ -e "$dir/.git" ] || return 1
    place_root=$dir

    if [ -d "$dir/.git" ]; then
        gitdir=$dir/.git
    else
        # A linked worktree's .git is a file naming its gitdir.
        gitdir=$(<"$dir/.git")
        gitdir=${gitdir%%$'\n'*}
        gitdir=${gitdir#gitdir: }
        [ "${gitdir#/}" = "$gitdir" ] && gitdir=$dir/$gitdir
    fi

    [ -r "$gitdir/HEAD" ] || return 0
    head=$(<"$gitdir/HEAD")
    head=${head%%$'\n'*}
    case $head in
        'ref: refs/heads/'*) place_branch=${head#ref: refs/heads/} ;;
        'ref: '*) place_branch=${head#ref: } ;;
        *) place_branch=${head:0:7} ;; # detached
    esac
}
