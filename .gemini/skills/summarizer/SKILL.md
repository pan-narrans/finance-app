---
name: Git changes summarizer
description: Generate project summaries from git diffs. Use when the user needs a high-level overview of changes between the current branch and a target branch (default 'main').
---

# Git changes summarizer

## Overview

The Git changes summarizer skill enables the creation of concise, high-signal project summaries by analyzing git diffs. It helps stakeholders understand the impact of changes without reading through thousands of lines of code.

## Workflow

1. **Identify Target Branch**: Default to `main` unless the user specifies otherwise.
2. **Analyze Diffs**: Execute `git diff <target_branch> --stat` to get a list of modified files and change volume.
3. **Generate Summary**: Create a structured report following the [Summary Pattern](#summary-pattern).

## Summary Pattern

Follow this structure exactly for every summary:

### Summary
1-2 sentences explaining the ultimate goal and high-level impact of the changes.

### Changes
Group bullet points by architectural layer or functional category (e.g., Domain, App Layer, Adapters, Config).
- Use bold labels for categories.
- Focus on "why" and "impact" rather than just listing file names.
- Keep bullet points concise and technical.

## Examples

### User: "Summarize changes against main"
1. Run `git diff main --stat`.
2. Generate summary:
    > ## Summary
    > Built robust Excel-to-Ledger import pipeline with deterministic identity, case-insensitive description
    > cleaning, and partial success reporting.
    > ## Changes
    > - **Domain**: Improved `Transaction` identity logic. Switched `GenerateCode` to truncated SHA-256 (16 chars) using pipe-delimited header fields and postings. Metadata in `Format` now sorted alphabetically for stable file diffs.
    > - **App Layer**: Implemented `ImportService` orchestrator. Added `ImportSummary` and partial success logic to handle row-level failures without stopping the entire process. Refactored ports to `primary.go` and `secondary.go` for cleaner architecture.
    > - **Adapters**:
    >     - **Excel**: Created `OpenBankParser` with HTML-to-Transaction mapping. Added regex-based case-insensitive prefix stripping (e.g., "Apple pay:") and card-to-owner resolution.
    >     - **Security**: Obfuscated bank `Balance` into an 8-char MD5 `ID` in metadata to discourage user manual tampering.
    >     - **Factory**: Added `ParserFactory` to dynamically select parser by file type.
    > - **Configuration**: Added `mappings.json` for centralized account, card, and prefix configuration.
    > - **Infrastructure**: Updated `.gitignore` for Go binaries and added `.env` support for local configuration.

