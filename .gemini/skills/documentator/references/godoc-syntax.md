# Godoc Syntax Rules

Follow these rules for Go 1.19+ documentation.

## Syntax
- **Paragraphs**: Separate with a blank line.
- **Lists**: Use `-` or `*` indented by two spaces.
- **Headings**: Use `#` at start of line (Go 1.19+).
- **Links**: URLs are auto-linked. Doc links use `[Name]`.
- **Preformatted**: Indent lines by one or more spaces (for code blocks).

## Standards
- Use **Block Comments** (`/* ... */`) for complex structs to consolidate field info.
- Use **Line Comments** (`// ...`) for simple functions or constants.
- First sentence should be a summary of the item.
- Use imperative mood for function descriptions ("Execute task" not "Executes task").
- No fluff. Be direct.
