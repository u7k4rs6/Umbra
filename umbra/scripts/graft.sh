#!/usr/bin/env bash
# Graft umbra/ into a fork of entireio/entire-graph.
#
# THIS SCRIPT IS NOT RUN BY THE BUILD. It is written for a person to read
# first and run deliberately, because it forks a repository under their account
# and enables Entire in a new checkout. Run it with --dry-run to see every
# command it would run without running any of them.
#
#   ./scripts/graft.sh --dry-run
#   ./scripts/graft.sh
#
# The umbra/ layout was preserved for exactly this: the directory copies in
# unchanged and nothing inside it needs editing.

set -euo pipefail

UPSTREAM="entireio/entire-graph"
BRANCH="umbra"
WORKDIR="${GRAFT_WORKDIR:-$HOME/.cache/umbra-graft}"
DRY_RUN=0

for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=1 ;;
        -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
        *) echo "unknown argument: $arg" >&2; exit 2 ;;
    esac
done

say() { printf '\n== %s\n' "$1"; }

run() {
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '   would run: %s\n' "$*"
        return 0
    fi
    printf '   %s\n' "$*"
    "$@"
}

# The repository this script is being run from, which is where umbra/ lives.
SOURCE_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
if [ ! -d "$SOURCE_ROOT/umbra" ]; then
    echo "cannot find umbra/ next to this script; expected it at $SOURCE_ROOT/umbra" >&2
    exit 1
fi

say "checks"
for tool in gh git entire; do
    if command -v "$tool" >/dev/null 2>&1; then
        printf '   found %s\n' "$tool"
    else
        echo "   missing $tool" >&2
        [ "$DRY_RUN" -eq 1 ] || exit 1
    fi
done

say "fork $UPSTREAM under your account, without cloning"
run gh repo fork "$UPSTREAM" --clone=false --remote=false

say "clone the fork into $WORKDIR"
FORK="$(gh api user --jq .login 2>/dev/null || echo '<your-account>')/entire-graph"
run git clone "https://github.com/$FORK.git" "$WORKDIR"

say "add the upstream remote and branch from its default branch"
run git -C "$WORKDIR" remote add upstream "https://github.com/$UPSTREAM.git"
run git -C "$WORKDIR" fetch upstream
DEFAULT_BRANCH="$(gh repo view "$UPSTREAM" --json defaultBranchRef --jq .defaultBranchRef.name 2>/dev/null || echo main)"
run git -C "$WORKDIR" checkout -b "$BRANCH" "upstream/$DEFAULT_BRANCH"

say "copy umbra/ in unchanged"
# The build's own git metadata and the fixture virtualenv do not travel.
run rsync -a --exclude '.git' --exclude '.venv' --exclude '__pycache__' \
    "$SOURCE_ROOT/umbra/" "$WORKDIR/umbra/"

say "enable Entire in the new checkout"
# There is no region flag. `entire enable`, `entire login` and
# `entire configure` were all checked against CLI 0.10.5 and none of them
# accepts one, and no region appears anywhere in `entire --help`. The mirror a
# repository lands on follows the account you are logged in as, so the region
# is a property of the login and not of this checkout. SECURITY_AND_ACCESS.md
# says only that the mirror follows the fork's collaborator permissions.
#
# What this does is enable Entire with checkpoints as git refs, which is the
# default and is what makes the trail travel with a push. If a particular
# mirror is wanted, log in as the account on it before running this, and check
# the result in the status output at the end.
run env -C "$WORKDIR" entire enable --agent claude-code

if [ "$DRY_RUN" -eq 0 ]; then
    printf '   logged in as: %s\n' "$(entire auth status 2>/dev/null | head -1 || echo 'not logged in')"
fi

say "commit the documents first, then everything else"
# Two commits, in this order, so the design is in the history before the code
# that implements it, which is how this repository's own history reads.
run git -C "$WORKDIR" add umbra/docs
run git -C "$WORKDIR" commit -m "umbra: the four planning documents

Written before any code. The build that produced umbra/ happened in a
standalone repository with its own checkpoint trail; this fork's trail
begins at the graft."
run git -C "$WORKDIR" add umbra
run git -C "$WORKDIR" commit -m "umbra: the plugin

A single static binary, entire-umbra, dispatched by the Entire CLI as
entire umbra. It finds the code a change affects that the session never
looked at."

say "entire status in the grafted checkout"
if [ "$DRY_RUN" -eq 1 ]; then
    printf '   would run: entire status (in %s)\n' "$WORKDIR"
else
    env -C "$WORKDIR" entire status
fi

say "done"
printf '   the graft is in %s on branch %s\n' "$WORKDIR" "$BRANCH"
printf '   nothing has been pushed. review it, then: git -C %s push -u origin %s\n' "$WORKDIR" "$BRANCH"
printf '\n'
printf '   before pushing, decide what happens to the checkpoint trail:\n'
printf '   this fork starts a trail of its own at the graft, and the build trail\n'
printf '   stays in the standalone repository. scripts/checkpoint-refs-audit.sh\n'
printf '   reports what is in the existing one.\n' 
