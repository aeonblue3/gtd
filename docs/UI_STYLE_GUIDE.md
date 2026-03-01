# UI Style Guide (v1 Baseline)

This is the official UI visual baseline for the GTD web interface.

## Direction Summary

- Palette source: **Option 2 (Terminal Executive)**
- Layout/interaction source: **Option 3 (Card-First GTD)**
- Tone: **powerful, approachable, warm, mobile-forward**
- Inspirations: OmniFocus dark aesthetic and Things-style visual comfort

## Core Principles

1. **Card-first information design**
   - Major content appears in cards/panels, not dense tables by default.
2. **Friendly dark UI**
   - Dark surfaces with readable contrast, avoiding harsh pure-black blocks.
3. **Generous spacing**
   - Priority on touch comfort and scanability.
4. **Rounded geometry**
   - Buttons, inputs, cards, chips all use softened corners.
5. **Fast inline workflow**
   - Keep frequent actions lightweight and visible.

## Color Tokens

- `--color-bg`: `#0B1220`
- `--color-surface`: `#111827`
- `--color-surface-elevated`: `#1F2937`
- `--color-text`: `#E5E7EB`
- `--color-text-muted`: `#94A3B8`
- `--color-border`: `#334155`
- `--color-accent`: `#22C55E`
- `--color-info`: `#06B6D4`
- `--color-warning`: `#F59E0B`
- `--color-danger`: `#F87171`

## Typography

Primary stack:

- `"Avenir Next", "Nunito", "Inter", "SF Pro Text", "Segoe UI", Roboto, Helvetica, Arial, sans-serif`

Usage:

- Body: 16px default
- Secondary/meta: 13-14px
- Section headers: 18-22px
- Use medium/semibold for hierarchy, avoid heavy bold saturation

Optional mono stack (metadata only):

- `"SFMono-Regular", Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace`

## Shape and Spacing

Radius:

- cards: `14px`
- inputs/buttons: `12px`
- chips: `9999px`

Spacing scale:

- `4, 8, 12, 16, 20, 24, 32, 40`

Touch targets:

- Minimum interactive height: `44px`

## Component Style Baselines

### App Shell

- Background uses `--color-bg`
- Primary regions rendered in `--color-surface` cards
- Navigation remains compact but touch-friendly

### Cards

- Solid surface with subtle border
- Soft elevation on hover/focus where pointer exists
- Large internal padding (`16-20px`)

### Buttons

- Primary: accent fill + dark text
- Secondary: surface-elevated fill + border
- Danger: danger fill + dark text

### Inputs/Selects

- Elevated dark surface, clear border, rounded corners
- Focus ring uses accent with visible contrast

### Chips/Badges

- Pill shape
- Status variants use muted tinted backgrounds and readable text

## Accessibility

- Maintain at least WCAG AA contrast for body text
- Visible keyboard focus for all interactive elements
- Do not rely on color alone for status meaning

## Responsive Intent

- Mobile first: card stack and simplified filter controls
- Desktop: denser multi-column layout allowed while keeping card visuals
- Keep line lengths readable; avoid wide text blocks

## Implementation Notes

- Single source of truth for tokens: `web/css/tokens.css`
- Components should consume tokens; avoid hardcoded hex values in component rules
- If future theming is needed, add theme classes that override token groups

