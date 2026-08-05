# Runner mirror for `task check-ci`: gawk as awk, no LANG (tmux then rewrites
# tabs in -F output to "_"), CI set (termenv reads that as "not a TTY", dropping
# colour). Tools come from install.sh, so it cannot drift; Docker caches it.
#
# gawk is the one a runner has and Ubuntu does not: on `exit` it closes the pipe
# and SIGPIPEs the producer, where mawk drains the input first. Under pipefail
# that is the difference between a green gate here and a red one on CI.
FROM ubuntu:24.04

RUN apt-get update -qq && apt-get install -y -qq \
    ca-certificates curl gawk git make sudo >/dev/null \
    && update-alternatives --set awk /usr/bin/gawk >/dev/null

COPY install.sh /repo/install.sh
WORKDIR /repo
RUN ./install.sh gate-tools install_go

ENV PATH=/root/.local/bin:$PATH
# Deliberately no LANG, and CI is set by `task check-ci` at run time.
