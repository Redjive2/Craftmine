#!/usr/bin/env bash
# Installs the Enforcer crew agent into the local pogo workspace.
#
# - Copies tools/enforcer-prompt.md -> ~/.pogo/agents/enforcer.md
# - Registers the review-cycle-enforcer schedule (idempotent via --id)
# - Prints next-step instructions for activating without waiting for pogod restart
#
# Safe to re-run: the file copy is idempotent (overwrites in place) and the
# schedule registration is keyed on --id so re-registering replaces the entry.

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
src="${repo_root}/tools/enforcer-prompt.md"
dst_dir="${HOME}/.pogo/agents"
dst="${dst_dir}/enforcer.md"

if [ ! -f "$src" ]; then
    echo "error: $src not found" >&2
    exit 1
fi

mkdir -p "$dst_dir"
cp "$src" "$dst"
echo "installed enforcer prompt at $dst"

pogo schedule enforcer --cron "0 * * * *" --id review-cycle-enforcer \
    --replay once \
    --message "Run a Craftmine style/architecture review cycle."
echo "registered review-cycle-enforcer schedule"

cat <<'EOF'

next steps:
  pogo agent start enforcer    # activate now without waiting for pogod restart
  pogo agent list              # confirm enforcer is running
  pogo schedule list --agent enforcer
EOF
