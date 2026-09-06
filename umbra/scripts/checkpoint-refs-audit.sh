#!/usr/bin/env bash
# Report what the checkpoint refs on a remote contain.
#
# THIS SCRIPT ONLY READS. It never deletes a ref, never pushes, and never
# rewrites history. It exists so the owner of this repository can decide what
# to do about the checkpoint trail before making the repository public, with
# numbers rather than a guess.
#
#   ./scripts/checkpoint-refs-audit.sh
#   ./scripts/checkpoint-refs-audit.sh --remote origin
#
# Every hook-written checkpoint stores the whole session transcript. On a
# public repository those refs are public with it. The three choices are:
#
#   1. delete the refs from the remote and stop syncing them,
#   2. read them and accept what they contain, or
#   3. rebuild the history in a fresh repository with checkpoint sync off.
#
# This script gives the sizes and the counts the second choice needs.

set -euo pipefail

REMOTE="${1:-origin}"
if [ "${1:-}" = "--remote" ]; then REMOTE="${2:-origin}"; fi

command -v git >/dev/null || { echo "git is not on PATH" >&2; exit 1; }

printf 'checkpoint refs on %s\n' "$REMOTE"
printf '%s\n' "-------------------------------------------------------------------"

REFS=$(git ls-remote "$REMOTE" 'refs/entire/checkpoints/*' 2>/dev/null | awk '{print $2}' | sort)
if [ -z "$REFS" ]; then
    printf 'none. nothing to decide.\n'
    exit 0
fi

TOTAL=0
COUNT=0
HOMES=0
MAILS=0

printf '%-52s %10s %7s %7s\n' 'ref' 'bytes' 'paths' 'emails'

while IFS= read -r ref; do
    [ -n "$ref" ] || continue
    COUNT=$((COUNT + 1))
    name=$(basename "$ref")

    # A ref that was pushed from here is already local. Resolving it is a read;
    # fetching one that is not would change this repository, so it is reported
    # as unknown instead.
    if ! git rev-parse --quiet --verify "$ref" >/dev/null 2>&1; then
        printf '%-52s %10s %7s %7s\n' "$name" "not local" "-" "-"
        continue
    fi

    # Walk the tree rather than the whole archive. A checkpoint holds the
    # session transcript, which runs to megabytes, and putting that through a
    # shell variable is what made an earlier version of this script report
    # zero for every ref. awk does the arithmetic and the selection, so there
    # is no nested read loop and no dependence on the value of IFS.
    # --full-tree, because ls-tree is scoped to the working directory by
    # default and this script is run from anywhere in the repository. Without
    # it, running from a subdirectory lists nothing and every count reads zero.
    tree=$(git ls-tree -r --long --full-tree "$ref" 2>/dev/null || true)

    bytes=$(printf '%s\n' "$tree" | awk '{ total += $4 } END { print total + 0 }')

    homes=0
    mails=0
    for blob in $(printf '%s\n' "$tree" | awk '$5 ~ /\.(jsonl|json|txt)$/ { print $3 }'); do
        h=$(git cat-file blob "$blob" 2>/dev/null | { grep -coE '/home/[A-Za-z0-9._-]+' || true; })
        m=$(git cat-file blob "$blob" 2>/dev/null | { grep -coE '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}' || true; })
        homes=$((homes + ${h:-0}))
        mails=$((mails + ${m:-0}))
    done

    TOTAL=$((TOTAL + bytes))
    HOMES=$((HOMES + homes))
    MAILS=$((MAILS + mails))

    printf '%-52s %10s %7s %7s\n' "$name" "$bytes" "$homes" "$mails"
done <<EOF
$REFS
EOF

printf '%s\n' "-------------------------------------------------------------------"
printf '%-52s %10s %7s %7s\n' "$COUNT refs" "$TOTAL" "$HOMES" "$MAILS"
printf '\n'
printf 'bytes  is the size of everything the ref holds, transcripts included.\n'
printf 'paths  is the number of lines carrying an absolute home path.\n'
printf 'emails is the number of lines carrying an email address.\n'
printf 'Counts are per line, so a line with two matches counts once.\n'
printf '\n'
printf 'Nothing was deleted and nothing was pushed. To act on this:\n'
printf '  delete one:  git push %s --delete <ref>\n' "$REMOTE"
printf '  stop syncing: entire configure, and turn checkpoint sync off\n'
printf '  rebuild:      graft into a fork with scripts/graft.sh, sync off\n'
