# GitHub account workflow for this project

## Account

- GitHub user/org: `Devin-gits`
- Contact email: `DevinonGits@proton.me`
- Repository: `https://github.com/Devin-gits/osint-lead-platform`

## Authentication

This project uses a GitHub Personal Access Token (PAT) stored in the system `gh` credential store (`keyring`).

### Check current login

```bash
gh auth status
```

Expected active account: `Devin-gits`.

### If the token expires or login is lost

1. Obtain a new PAT from `DevinonGits@proton.me` or the GitHub account settings.
2. Do **not** paste the PAT into repository files, commits, or skill files.
3. Run:

```bash
gh auth login --with-token --hostname github.com
# paste the PAT on the single line and press Enter
gh auth setup-git
```

### Using git with the active account

After `gh auth setup-git`, git operations against `origin` use the `gh` credential helper and do not need the token in the remote URL.

```bash
git remote set-url origin https://github.com/Devin-gits/osint-lead-platform.git
git fetch origin
git push origin <branch>
```

Avoid embedding the PAT in the remote URL or in `.git/config`.

## Commit identity

Before authoring commits, ensure local git identity matches the new account if required by the user:

```bash
git config user.email "DevinonGits@proton.me"
git config user.name "Devin-gits"
```

Only run this when explicitly requested; do not change global git config without user confirmation.

## Repository operations

- Create a new repo under `Devin-gits`:

```bash
gh repo create Devin-gits/<repo-name> --public --description "..." --source=. --remote=origin --push
```

- Open a PR:

```bash
gh pr create --title "..." --body-file /tmp/pr-body.md
```

- Push all local branches:

```bash
git push --all origin
```

## Troubleshooting

- `remote: Your account is suspended` or 403: the old account token is active or the token is invalid. Run `gh auth logout -h github.com` for the old account, then re-authenticate with `Devin-gits`.
- `gh` cannot see the new repo: ensure you are logged in as `Devin-gits` and the repo URL is correct.
