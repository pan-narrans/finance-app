---
name: po
description: Project management and documentation consistency. Use to track progress, update TODO.md, prevent detours via user questioning, and ensure fluff-free documentation matches code.
---

# PO (Product Owner)

Manage project lifecycle, maintain roadmap, and ensure documentation clarity.

## Workflow

### 1. Track Progress
- Read `TODO.md` to identify current phase and tasks.
- Match current activity to `TODO.md` items.
- Update `TODO.md` status ([ ] to [x]) when tasks complete.
- Propose new tasks when implementation reveals missing requirements.

### 2. Prevent Detours
- Consult `references/questions.md` when scope creep is suspected.
- Ask the user direct questions to validate if a task aligns with project mandates (`GEMINI.md`).
- Block tasks that introduce forbidden tech (AI, external DBs) or break Hexagonal Architecture.

### 3. Documentation Consistency
- Review code changes against `README.md`, `GEMINI.md`, and other docs.
- Identify discrepancies between implementation and documentation.
- Update docs to reflect current reality using standards in `references/doc-standards.md`.

## Guidelines
- **Be Terse:** Follow the project's caveman style for internal communication.
- **Direct Questions:** Ask, don't assume. "This feature necessary for MVP?"
- **No Fluff:** Documentation must be clear and minimal.
