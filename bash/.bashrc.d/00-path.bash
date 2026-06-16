# PATH modifications
export PATH="$HOME/.local/bin:$PATH"
export PATH="$PATH:$HOME/.local/nvim/bin"
export PATH="$HOME/.local/go/bin:$PATH"
export PATH="$HOME/go/bin:$PATH"

# macOS: Homebrew prefix (Apple Silicon: /opt/homebrew, Intel: /usr/local)
if [ "$(uname)" = "Darwin" ]; then
    if [ -x /opt/homebrew/bin/brew ]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
    elif [ -x /usr/local/bin/brew ]; then
        eval "$(/usr/local/bin/brew shellenv)"
    fi
fi

# pnpm
export PNPM_HOME="$HOME/.local/share/pnpm"
case ":$PATH:" in
    *":$PNPM_HOME:"*) ;;
    *) export PATH="$PNPM_HOME:$PATH" ;;
esac

# fnm (Node version manager)
FNM_PATH="$HOME/.local/share/fnm"
if [ -d "$FNM_PATH" ]; then
    export PATH="$FNM_PATH:$PATH"
    eval "$(fnm env)"
fi

# pixi
[ -d "$HOME/.pixi/bin" ] && export PATH="$HOME/.pixi/bin:$PATH"

# cargo/rust
[ -f "$HOME/.cargo/env" ] && . "$HOME/.cargo/env"
