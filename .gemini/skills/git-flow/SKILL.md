---
name: git-flow
description: Branch, Git, and Merge Request (MR) Automation standards.
---

# Git Flow

Enforce branch naming, MR creation, and worktree-based development to maintain a clean and traceable repository history.

## Worktree-First Workflow
All development tasks MUST be performed in a dedicated **Git Worktree**. The `main/` directory is reserved for syncing, status checks, and orchestration.

### 1. Creation
When starting a task:
1. Create a new branch: `git branch feature/[ID-]<name>` (or `bugfix/`, etc).
2. Create a worktree sibling to `main/`: `git worktree add ../feature-[name] feature/[ID-]<name>`
3. Move to the new worktree directory to perform all work.

### 2. Synchronization
- Pull updates in `main/`: `git pull origin main`.
- Rebase your feature branch as needed from within its worktree.

### 3. Cleanup
Once the MR is merged:
1. Remove the worktree: `git worktree remove ../feature-[name]`
2. Delete the local branch: `git branch -d feature/[ID-]<name>`
3. Run `git worktree prune` to clean up references.

## Branch Naming
Follow the structure `prefix/[ID-]<description>`. Use lowercase and hyphens.

- **Releases:** `release/<name>`
  - Used to group related tasks.
  - Merges into `main` as a unit.
  - Example: `release/ux-improvements`
- **Features:** `feature/[ID-]<name>`
  - Used for single atomic functionalities.
  - Example: `feature/61-group-chat`
- **Bugfixes:** `bugfix/[ID-]<name>`
  - Used for fixes discovered later in development.
  - Example: `bugfix/61-group-chat`
- **No-Task:** `notask/<name>`
  - Used for small improvements with no associated GitHub Issue.
  - Example: `notask/refactor-imports`

## Merge Requests (MR)
- **Targeting:**
  - Feature/Bugfix branches target the active `release/` branch.
  - Release branches target `main`.
- **Naming:**
  - **Release MR:** Name it `Release/<release-name>`.
  - **Feature/Bug MR:** Use the exact **GitHub Issue Title**.
- **Merge Method:**
  - **Release MR to main:** Use **Standard Merge** (preserve history).
  - **Feature/Bug MR to release:** Always pre-select **Squash Merge**.
- **Linking:**
  - Link the MR to the relevant GitHub Issue using the native "Linked Issues" functionality (do not rely solely on description mentions).

## Workflow Integration
- Use `git status`, `git log`, and `git diff` to verify state before any operation.
- Always propose a draft commit message and MR title/body for review.
