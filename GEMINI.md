# GEMINI.md — Project & Design Rules for GoRoute

## UI & Typography Standards

1. **Mandatory Typeface Hierarchy**:
   - **Primary UI Font**: Geometric, rounded, high-craft sans (`Quicksand`, `Plus Jakarta Sans`). Never default to generic system fonts (`-apple-system`, `Inter`, `Arial`).
   - **Code & Numeric Font**: `JetBrains Mono` for latency badges, token rates, model identifiers, and config snippets.

2. **Visual Aesthetics & Craft**:
   - Zero childish emojis in any part of the UI or notifications. Use crisp Lucide SVGs exclusively.
   - High-contrast Devin Hacks Kochi palette (15:1+ contrast ratios, warm light tones, deep navy/slate dark mode).
   - Subtle animated background layers (Aurora mesh + film grain noise).

3. **Interaction & Feedback**:
   - Global interactive cursor pointer (`cursor: pointer !important`) on all clickable elements.
   - Smooth reactive cursor ring (`CustomCursor.tsx`) with hover expansion and active press physics.

4. **CSS Preflight Hygiene**:
   - Never attach layout constraints (`min-height: 100vh`, `overflow-x: hidden`, or background colors) to multi-element selectors (`body, button, input`). Isolate layout dimensions strictly to `body`.
