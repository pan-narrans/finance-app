---
name: llm-expert
description: Manages Gemini's context window, memory organization, and prompt efficiency for this project. Use when designing new skills, refactoring project-wide instructions (GEMINI.md), or optimizing token usage.
---

# LLM Expert

Ensure Gemini operates at peak efficiency by managing its limited context window and organizing project-wide memory using the **Progressive Disclosure** principle.

## Core Mandates

1.  **Context Efficiency**: Every token in the prompt must justify its presence. Remove redundant articles, pleasantries, and boilerplate from persistent instructions.
2.  **Progressive Disclosure**: Only load the context needed for the current task.
    - **Global rules** (Persona, Tone): Live in root `GEMINI.md`.
    - **Architectural rules**: Live in the `architect` skill.
    - **Coding standards**: Live in the `dev-go` skill.
    - **Domain-specific tools**: Live in their respective skills (e.g., `ledger-format`).
3.  **Memory Routing**:
    - Project-wide, long-term mandates -> Create or update a relevant **Skill**.
    - Machine-specific or private notes -> Private Project Memory (`MEMORY.md`).
    - Personal user preferences -> Global Personal Memory.

## Workflows

### 1. Optimizing Context
When the root `GEMINI.md` or a `SKILL.md` becomes too large (>500 lines):
1.  Identify logical clusters (e.g., "All rules about Go tests").
2.  Extract clusters into standalone Skills or Reference files.
3.  Replace the extracted section with a single-line pointer or trigger description in the original file.

### 2. Prompt Engineering
Design multi-step plans that use sub-agents to "compress" work. Avoid large, monolithic turns. Prefer surgical edits.
