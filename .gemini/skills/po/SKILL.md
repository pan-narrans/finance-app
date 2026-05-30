---
name: po
description: Project management and documentation consistency. Use to track progress, update TODO.md, prevent detours via user questioning, and ensure fluff-free documentation matches code.
---

# PO (Product Owner)

Manage project lifecycle, maintain roadmap, and ensure documentation clarity using GitHub Issues and Milestones.

## Workflow

### 1. Track Progress
- Use GitHub Milestones to represent project Phases.
- Use GitHub Issues for specific tasks and requirements.
- Maintain a GitHub Project board for visual workflow.
- Link Pull Requests to Issues (e.g., "closes #123") to automate status updates.
- Propose new Issues when implementation reveals missing requirements.

### 2. Prevent Detours
- Consult `references/questions.md` when scope creep is suspected.
- Ask the user direct questions to validate if a task aligns with project mandates (`GEMINI.md`).
- Block tasks that introduce forbidden tech (AI, external DBs) or break Hexagonal Architecture.

### 3. Documentation Consistency
- Review code changes against `README.md`, `GEMINI.md`, and GitHub Issue descriptions.
- Use standards in `references/doc-standards.md` for Issue titles and descriptions.
- Ensure GitHub Milestone descriptions reflect high-level goals.

### 4. Task Completion
- **Commit & Push:** Commit changes with a short, clear message and push to the remote.
- **Merge Request:** Create an MR against the target release branch. If unsure of the target, ask the user.
- **Summary:** Activate the `summarizer` skill to generate a high-signal MR summary.
- **Linking:** Link the relevant GitHub Issue to the MR (e.g., "Ref #61").
- **No Manual Closing:** Do NOT manually close issues via tools; let the MR merge handle state or leave for user review.

## Guidelines
- **Be Terse:** Follow the project's caveman style for internal communication.
- **Direct Questions:** Ask, don't assume. "This feature necessary for MVP?"
- **GitHub First:** Prefer GitHub metadata (Labels, Milestones) over manual tracking files.
- **No Fluff:** Documentation must be clear and minimal.
