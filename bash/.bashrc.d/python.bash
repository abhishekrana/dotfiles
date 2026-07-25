command -v python3 &>/dev/null || return

# Generate pyrightconfig.json for all .venv directories in a project
pyright-init() {
    find "${1:-.}" -maxdepth "${2:-3}" -name ".venv" -type d | while read -r venv; do
        local dir
        dir=$(dirname "$venv")
        if [ ! -f "$dir/pyrightconfig.json" ]; then
            echo '{"venvPath": ".", "venv": ".venv"}' >"$dir/pyrightconfig.json"
            echo "Created: $dir/pyrightconfig.json"
        else
            echo "Exists:  $dir/pyrightconfig.json"
        fi
    done
}
