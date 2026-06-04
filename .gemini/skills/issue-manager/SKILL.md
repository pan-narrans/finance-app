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

### Tasks
Format: `<Verb> <Subject>`
- Example: `Implement isTriggered logic`
- Example: `Update dependencies`

### Bugs
Format: `<Feature> should <expected behavior> but <actual result>`
- Example: `Auth middleware should whitelist Group IDs but currently only allows User IDs.`

### New Features & Improvements
- **Features:** `Implement <component/logic>`
- **Improvements:** `Improve <feature> performance` or `<component> > also <additional functionality>`

## Guidelines
- Use Labels (e.g., `enhancement`, `bug`, `ux`) to categorize issues.
- Assign Issues to the relevant Milestone (Project Phase).
- Use `mcp_github_sub_issue_write` to maintain the Parent/Child relationship.
- Keep descriptions concise and focused on the technical requirements.
