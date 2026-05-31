---
name: git-flow
description: Branch, Git, and Merge Request (MR) Automation standards.
---

# Git Flow

Enforce branch naming, MR creation, and merging standards to maintain a clean and traceable repository history.

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
