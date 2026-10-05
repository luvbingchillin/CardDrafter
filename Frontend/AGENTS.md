# React & TypeScript Frontend Guidelines

## Architecture & Conventions
- **Strict Typing:** All components, hooks, and API responses must have explicit TypeScript types.
- **Event Handlers:** Type form events explicitly using `React.FormEvent<HTMLFormElement>`.
- **Error Handling:** In `try/catch` blocks, narrow caught errors using `if (err instanceof Error)` before accessing `.message`.
- **Styling:**
  - Use CSS Modules (`*.module.css`) for component-specific styles.
  - Rely on global theme tokens defined in `src/index.css` (e.g. `var(--primary)`, `var(--bg-base)`).
  - Avoid raw inline `style={{ ... }}` objects.
- **API Calls:** Use relative paths (e.g. `/api/packs`) so requests are seamlessly handled by Vite's dev proxy locally and Nginx in production.
