---
name: issue-manager
description: GitHub Issue standards, naming templates, and hierarchy.
---

# Issue Manager

Standardize how requirements are captured and organized within GitHub Issues and Milestones.

## Hierarchy
- **User Stories (Parent Issues):** High-level features or epics. They represent the "What" and "Why".
- **Tasks (Child Issues):** Atomic units of work linked to a Parent User Story. They represent the "How".

## Naming Templates

### User Stories
Format: `As a <user>, I want <goal> so that <reason>`
- Example: `As a user, I want to track expenses in a Telegram group so that my partner and I can manage shared finances.`

**Required Sections:**
1. **User Journey:** Step-by-step description of the user interaction.
2. **Acceptance Criteria (AC):** Bulleted list of testable conditions that must be met.
3. **Technical Notes:** Any specific constraints or integration points.

### Tasks
Format: `<Verb> <Subject>`
- Example: `Implement isTriggered logic`
- Example: `Update dependencies`

**Required Sections:**
1. **Goal:** Brief description of what this task achieves.
2. **Acceptance Criteria (AC):** How to verify the task is complete.

### Bugs
Format: `<Feature> should <expected behavior> but <actual result>`
- Example: `Auth middleware should whitelist Group IDs but currently only allows User IDs.`

**Required Sections:**
1. **Reproduction Steps:** List of steps to trigger the bug.
2. **Expected vs Actual:** Clear contrast of behavior.
3. **Acceptance Criteria (AC):** Verification that the fix works and prevents regression.

## Guidelines
- Use Labels (e.g., `enhancement`, `bug`, `ux`) to categorize issues.
- Assign Issues to the relevant Milestone (Project Phase).
- Use `mcp_github_sub_issue_write` to maintain the Parent/Child relationship.
- Keep descriptions concise and focused on the technical requirements.
