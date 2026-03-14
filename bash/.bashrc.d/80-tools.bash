# uv/uvx shell completions
command -v uv &>/dev/null && eval "$(uv generate-shell-completion bash)"
command -v uvx &>/dev/null && eval "$(uvx --generate-shell-completion bash)"
