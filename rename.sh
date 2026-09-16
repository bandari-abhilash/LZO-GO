#!/bin/sh
# Set the module path and copyright holder in one shot.
#   ./rename.sh <github-user> <repo-name> "Your Name"
set -e
U=$1; R=$2; N=$3
[ -n "$U" ] && [ -n "$R" ] && [ -n "$N" ] || { echo "usage: $0 <github-user> <repo> \"Your Name\""; exit 1; }
grep -rl 'YOURGITHUB/lzo1z' . --include='*.go' --include='*.md' --include='go.mod' \
  | xargs sed -i '' "s|YOURGITHUB/lzo1z|$U/$R|g"
grep -rl 'YOUR NAME' . --include='*.go' --include='*.c' \
  | xargs sed -i '' "s|YOUR NAME|$N|g"
echo "module path -> github.com/$U/$R  (package name left as 'lzo1z')"
go test ./...
