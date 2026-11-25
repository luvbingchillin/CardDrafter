# React & TypeScript Frontend Guidelines

## 1. TypeScript & State Architecture
- **Strict Contracts:** All components, hooks, WebSocket envelopes (`WSMessage<T>`), and API payloads must be explicitly typed. Match Go struct JSON field casing exactly (e.g. `Seat.IsConnected`, `Seat.UserID`, `Seat.IsReady`) and never bind UI logic to backend fields omitted via `json:"-"`.
- **Event & Error Typing:** Type form events with `React.FormEvent<HTMLFormElement>` and narrow caught errors with `if (err instanceof Error)` before reading `.message`.
- **WebSocket Lifecycle (`useRef` vs. `useState`):**
  - Always store imperative `WebSocket` instances in `useRef<WebSocket | null>(null)`—never in `useState`—to avoid dependency-array reconnection loops and stale closures.
  - Guard outgoing messages with `if (wsRef.current?.readyState === WebSocket.OPEN)`.
  - Always close the socket inside a synchronous `useEffect` cleanup return (`() => { ws.close(); }`) to survive React 18 StrictMode double-mounting.
- **High-Frequency Tick Slicing:**
  - Keep 1-second `TIMER_TICK` countdown state in an independent primitive slice (`const [timeRemaining, setTimeRemaining] = useState(0)`) separate from `draftState` so active pack and drafted pool card grids do not re-render every second.

## 2. Accessibility (A11y) & Craft Floor Standards
- **Keyboard & Screen-Reader Parity:**
  - Interactive non-button containers (`CardUi`, `PackUi`) must include `role="button"`, `tabIndex={0}`, `aria-label`, and `onKeyDown` handling `Enter` and `Space`.
  - Form controls (`Input.tsx`) must connect `<label htmlFor={id}>` and `<input id={id}>` via React 18 `useId()`.
  - Modals (`LoginModal.tsx`) must enforce `role="dialog"`, `aria-modal="true"`, focus trapping, and `Escape` key dismissal.
- **Anti-Slop Visual Craft Floor:**
  - Use CSS Modules (`*.module.css`) and global theme tokens from `src/index.css`.
  - Use unified SVG vector icons (`src/components/ui/Icons.tsx`) instead of Unicode emojis.
  - Use physical directional drop shadows with crisp `1px` borders instead of zero-offset neon halos.
  - Apply `font-variant-numeric: tabular-nums` to all timers and counters, and use smooth exponential deceleration (`cubic-bezier(0.16, 1, 0.3, 1)`) instead of overshooting bounce curves.
