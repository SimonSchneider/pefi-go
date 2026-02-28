---
name: github-issue-solver
description: Solve GitHub issues end-to-end using the gh CLI. Use whenever the user references a GitHub issue by number (e.g. #123, issue 42), asks to fix an issue, work on an issue, or implement something for an issue. Create a branch for the issue, implement the fix, wait for user approval before committing, then push and open a PR linked to the issue. Always use the `gh` CLI for all GitHub interaction — never use the GitHub API directly or manual git remote URLs when gh can do it.
---

# GitHub Issue Solver

Work through GitHub issues from reference to merged PR using the **`gh` CLI** for all GitHub operations. Issue references look like `#123` or "issue 42" — treat these as the same.

## Prerequisites

- **gh CLI** must be installed and authenticated (`gh auth status`). If not, tell the user to install and run `gh auth login`.
- Run commands from the repository root (the repo that contains the issue, or the repo the user is working in).

## Workflow

### 1. Resolve the issue number

From the user's message, get the issue number (e.g. `123` from `#123` or "issue 123"). If the repo isn't obvious, use the current workspace repo or ask.

### 2. Fetch issue details

Use `gh` to load the issue so you know what to implement:

```bash
gh issue view <NUMBER>
```

Use `--json title,body,labels` if you need machine-readable output. Read the title and body to understand scope, acceptance criteria, and constraints.

### 3. Create a branch for the issue

Create and checkout a branch linked to the issue so the PR will be associated correctly:

```bash
gh issue develop <NUMBER> --checkout
```

This creates a branch (e.g. `username/issue-123-description`) and checks it out. If you need a specific base branch (e.g. `main`), use `--base main`. Do **not** create branches with `git checkout -b`; use `gh issue develop` so the branch is linked to the issue.

### 4. Implement the fix

- Implement the changes required by the issue (code, tests, docs, etc.).
- Follow project conventions (see CLAUDE.md, existing code, linters).
- Run relevant tests and fix any failures before asking for approval.

### 5. Wait for user approval before committing

**Do not commit or push until the user explicitly approves the changes.**

- Summarize what you changed and where.
- Ask the user to review (e.g. "Please review the changes above. Say when you're happy for me to commit and open a PR.").
- If they request edits, make them and ask again. Only after they confirm (e.g. "looks good", "approved", "go ahead") proceed to the next step.

### 6. Commit the changes

After approval, create a single logical commit (or a few clear commits if the fix is large). Use a concise message that references the issue:

```bash
git add -A
git commit -m "Fix <short description> (fixes #<NUMBER>)"
```

Including "fixes #&lt;NUMBER&gt;" in the commit message helps link the commit to the issue; the PR body will link it as well.

### 7. Push the branch and create a PR

Push the branch and open a PR that **closes** the issue:

```bash
git push -u origin HEAD
gh pr create --fill --body "Fixes #<NUMBER>"
```

- `--fill` uses the commit message(s) for title and body; ensure the first commit or your edit includes "Fixes #&lt;NUMBER&gt;" so GitHub links and will close the issue on merge.
- If you need a custom title: `gh pr create --title "Your title" --body "Fixes #<NUMBER>"`.
- If the repo has multiple remotes or you're in a fork, `gh pr create` will target the default remote; use `--repo OWNER/REPO` if you need to specify.

### 8. Confirm with the user

Tell the user the PR URL (e.g. from `gh pr view --web` or the URL printed by `gh pr create`) and that the PR is linked to the issue and will close it when merged.

## Using the gh CLI

Use **`gh`** for all GitHub interaction in this workflow:

| Task | Command |
|------|--------|
| View issue | `gh issue view <NUMBER>` |
| Create/checkout branch for issue | `gh issue develop <NUMBER> --checkout` |
| Create PR (from current branch) | `gh pr create --fill --body "Fixes #<NUMBER>"` |
| Open PR in browser | `gh pr view --web` |
| Check auth | `gh auth status` |
| Repo context | `gh repo view` or `-R OWNER/REPO` on any command |

Do not use raw GitHub API calls or manual `git remote`/PR creation when these commands suffice.

## Edge cases

- **Issue in another repo:** Use `-R OWNER/REPO` with `gh issue view` and `gh issue develop`. Push and `gh pr create` from a clone of that repo (or the fork you're using).
- **Branch already exists:** If `gh issue develop` says the branch exists, list with `gh issue develop --list <NUMBER>`, then `git fetch` and `git checkout <branch-name>`.
- **User wants multiple commits:** After approval, make multiple logical commits, then push once and create one PR with body "Fixes #&lt;NUMBER&gt;".
- **Draft PR:** If the user wants a draft, use `gh pr create --draft --fill --body "Fixes #<NUMBER>"`.

## Summary

1. Get issue number from user (e.g. #123).
2. `gh issue view <NUMBER>` to read the issue.
3. `gh issue develop <NUMBER> --checkout` to create and switch to the branch.
4. Implement the fix and run tests.
5. **Wait for explicit user approval** — do not commit before that.
6. After approval: commit (message can include "fixes #&lt;NUMBER&gt;"), then `git push -u origin HEAD` and `gh pr create --fill --body "Fixes #<NUMBER>"`.
7. Share the PR link and confirm it’s linked to the issue.
