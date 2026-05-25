#!/usr/bin/env bash
# Pre-commit hook: bump version + UTC timestamp in Progress.md's auto-stamp block.
#
# Invoked by .git/hooks/pre-commit (see tools/install-hooks.sh).
# Never blocks a commit — any failure logs a warning and exits 0.
#
# Skip rules:
#   - Commit message starts with "WIP:" (read from .git/COMMIT_EDITMSG).
#   - Caller passed --no-verify to git (git itself skips the hook).

set -u

warn() { echo "pre-commit-progress: $*" >&2; }

repo_root=$(git rev-parse --show-toplevel 2>/dev/null) || { warn "not in a git repo"; exit 0; }
cd "$repo_root" || { warn "cannot cd to repo root"; exit 0; }

progress="Progress.md"
git_dir=$(git rev-parse --git-dir 2>/dev/null) || git_dir=".git"
commit_msg_file="${git_dir}/COMMIT_EDITMSG"

if [ -f "$commit_msg_file" ]; then
    first_line=$(sed -n '1p' "$commit_msg_file" 2>/dev/null || true)
    case "$first_line" in
        WIP:*) exit 0 ;;
    esac
fi

if [ ! -f "$progress" ]; then
    warn "Progress.md not found, skipping stamp"
    exit 0
fi

current=$(grep -oE '^Version: [0-9]+\.[0-9]+\.[0-9]+' "$progress" | tail -n 1 | sed -E 's/^Version: //')
[ -z "$current" ] && current="0.0.0"

IFS=. read -r major minor patch <<EOF
$current
EOF
patch=$((patch + 1))
new_version="${major}.${minor}.${patch}"
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

tmp=$(mktemp "${progress}.XXXXXX") || { warn "mktemp failed"; exit 0; }
trap 'rm -f "$tmp"' EXIT

if grep -q '<!-- auto-stamp:start -->' "$progress"; then
    awk -v ver="$new_version" -v ts="$timestamp" '
        /<!-- auto-stamp:start -->/ {
            print "<!-- auto-stamp:start -->"
            print "Version: " ver
            print "Last commit: " ts
            print "<!-- auto-stamp:end -->"
            in_block = 1
            next
        }
        /<!-- auto-stamp:end -->/ {
            in_block = 0
            next
        }
        !in_block { print }
    ' "$progress" > "$tmp" || { warn "awk replace failed"; exit 0; }
else
    awk -v ver="$new_version" -v ts="$timestamp" '
        BEGIN { inserted = 0 }
        {
            print
            if (!inserted && /^# /) {
                print ""
                print "<!-- auto-stamp:start -->"
                print "Version: " ver
                print "Last commit: " ts
                print "<!-- auto-stamp:end -->"
                inserted = 1
            }
        }
    ' "$progress" > "$tmp" || { warn "awk insert failed"; exit 0; }
    if ! grep -q '<!-- auto-stamp:start -->' "$tmp"; then
        {
            printf '<!-- auto-stamp:start -->\nVersion: %s\nLast commit: %s\n<!-- auto-stamp:end -->\n\n' "$new_version" "$timestamp"
            cat "$progress"
        } > "$tmp" || { warn "prepend failed"; exit 0; }
    fi
fi

mv "$tmp" "$progress" || { warn "mv failed"; exit 0; }
trap - EXIT

git add "$progress" 2>/dev/null || warn "git add failed"
exit 0
