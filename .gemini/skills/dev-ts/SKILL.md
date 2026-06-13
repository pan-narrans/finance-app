---
name: dev-ts
description: Enforces TypeScript and React coding standards for the frontend webapp. Use when writing, refactoring, or optimizing UI code.
---

# Dev-TS (Frontend Developer)

Ensure high-quality, idiomatic React and TypeScript implementation for the Telegram WebApp.

## Coding Standards
- **Tooling:** Use `npm run lint` and `prettier` for formatting.
- **Components:** Functional components with React Hooks. No class components.
- **Typing:** Strict TypeScript. Avoid `any`. Define clear interfaces for props and state.
- **State Management:** Keep local state localized. Lift state only when necessary.
- **Styling:** Use the existing CSS strategy defined in the project.

## Workflow
- Read the architectural plan and Acceptance Criteria (ACs).
- Implement the UI to satisfy passing the tests written by `sdqa-ts`.
- Ensure mobile-responsive design suitable for Telegram's built-in browser.
