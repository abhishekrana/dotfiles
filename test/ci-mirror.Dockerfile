# Runner mirror for `task check-ci`: tmux 3.4, no LANG (tmux then rewrites tabs
# in -F output to "_"), CI set (termenv reads that as "not a TTY", dropping
# colour). Tools come from install.sh, so it cannot drift; Docker caches it.
FROM ubuntu:24.04

RUN apt-get update -qq && apt-get install -y -qq \
    curl git make sudo ca-certificates >/dev/null

COPY install.sh /repo/install.sh
WORKDIR /repo
RUN ./install.sh gate-tools install_go

ENV PATH=/root/.local/bin:$PATH
# Deliberately no LANG, and CI is set by `task check-ci` at run time.
