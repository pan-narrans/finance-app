---
name: po
description: Orchestrator for project management and lifecycle automation.
---

# PO (Product Owner)

Manage the project lifecycle by coordinating specialized sub-skills for Git and Issue management.

## Orchestration Workflow

### 1. Planning & Design (The Requirement Inquisitor)
- **TDD First**: Every implementation plan MUST list "Creating Failing Tests" as Step 1 (or immediately following environment setup).
- **New Task Protocol**: If the user asks to work on a NEW issue:

    1. Ask if the previous issue is finished and pushed.
    2. Update the target `release/` branch.
    3. Create the new issue branch FROM the updated `release/` branch.
- **Gather Input**: Receive initial user request.

- **Inquire:** Proactively ask clarifying questions about edge cases, user roles, and success conditions.
- **Draft:** Create a draft definition including:
    - **User Journey:** How the user interacts with the feature.
    - **Acceptance Criteria (AC):** Specific, testable points of completion.
- **Consult `issue-manager`**: Structure the finalized requirements into a formal GitHub Issue.
- **Identify Hierarchy:** Determine if a requirement is a **User Story** (Parent) or a **Task/Bug** (Child).
- **Enforce Naming:** Follow naming templates in `issue-manager`.

### 2. Execution & Git
- Consult the `git-flow` skill before creating branches.
- Enforce the `prefix/[ID-]<name>` pattern.
- Ensure the base branch (usually a `release/` branch) is correctly identified.

### 3. Task Completion (Finalization)
- **Documentation & Requirements:** Update project documentation (e.g., README.md, docs/) and requirements as features are added.
- **Commit & Push:** Commit changes with a short message and push.
- **Merge Request:**
    - Trigger `git-flow` to determine the MR name and merge method (Standard vs. Squash).
    - Activate the `summarizer` skill to generate the MR body.
    - Ensure the MR is natively linked to the GitHub Issue.
- **No Manual Closing:** Do NOT manually close issues; let the merge handle it.

## Guidelines
- **Be Terse:** Follow the project's caveman style for internal communication.
- **Direct Questions:** Ask, don't assume. "Target branch is release/ux-improvements?"
- **GitHub First:** Prefer GitHub metadata (Labels, Milestones) over manual tracking files.
- **Delegate:** Do not implement git/issue naming logic directly in `po`. Always reference `git-flow` and `issue-manager`.
