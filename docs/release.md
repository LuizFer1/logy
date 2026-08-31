# Releasing Logy

1. Merge the release branch to `main`.
2. Write release notes at `docs/releases/vX.Y.Z.md` (must match the tag name).
3. Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z` (repo: https://github.com/LuizFer1/logy)
4. GitHub Actions (`.github/workflows/release.yml`) runs GoReleaser with that notes file and publishes a GitHub Release with archives + `checksums.txt`.
5. Users install or upgrade with:
   - `scripts/install.ps1` / `scripts/install.sh`

Pin installs with `LOGY_VERSION=v0.1.0`. Forks can set `LOGY_GITHUB_REPO=owner/name`.
