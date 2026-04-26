#!/usr/bin/env sh
set -eu

git tag -d v1.3.0
git push origin :refs/tags/v1.3.0
git add .
git commit -m "Final Iron-Clad Go Module Refactor (v1.3.0)"
git tag v1.3.0
git push origin main --tags
