#!/usr/bin/env bash
# Installs .git/hooks/pre-commit so it calls tools/pre-commit-progress.sh.
# Idempotent: re-running replaces the hook content.
# Worktree-safe: hooks are installed in the current worktree's hook dir.

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
# Use --git-common-dir, not --git-dir: hooks live in the shared gitdir so they
# fire from any worktree. --git-dir returns the per-worktree path, which git
# does not consult for hooks.
hook_dir="$(git rev-parse --git-common-dir)/hooks"
hook_file="${hook_dir}/pre-commit"
script_rel="tools/pre-commit-progress.sh"
script_abs="${repo_root}/${script_rel}"

if [ ! -f "$script_abs" ]; then
    echo "error: $script_rel not found at $script_abs" >&2
    exit 1
fi

mkdir -p "$hook_dir"

cat > "$hook_file" <<'HOOK'
#!/usr/bin/env bash
# Installed by tools/install-hooks.sh. Delegates to the tracked auto-stamp script.
script="$(git rev-parse --show-toplevel)/tools/pre-commit-progress.sh"
if [ -x "$script" ]; then
    exec "$script"
fi
echo "pre-commit: $script not executable, skipping stamp" >&2
HOOK

chmod +x "$hook_file"
chmod +x "$script_abs"

echo "installed pre-commit hook at $hook_file"
echo "it calls $script_rel"
