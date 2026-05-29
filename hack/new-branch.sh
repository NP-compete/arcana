#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 2 ]; then
  echo "Usage: $0 <type> <slug>"
  echo "  type: scaffold | feat | fix | refactor | docs | chore"
  echo "  slug: kebab-case description"
  echo ""
  echo "Example: $0 feat add-skill-versioning"
  exit 1
fi

TYPE="$1"
SLUG="$2"
BRANCH="${TYPE}/${SLUG}"

git fetch origin main
git checkout -b "${BRANCH}" origin/main
echo "Created branch: ${BRANCH}"
echo "When done: git push -u origin HEAD && gh pr create"
