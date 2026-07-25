# FilmMash — Visual Identity

This document specifies FilmMash's visual identity so that any contributor — human or AI agent — can build new UI that looks native to the app. It is derived from (and must stay in sync with) the single source of truth for styling: the `<style>` block in [internal/view/template/base.html](../internal/view/template/base.html).

**When adding or changing UI, follow this spec. When changing this spec, change `base.html` too, and vice versa.**

## 1. Concept

**Golden Age of Hollywood / film noir.** The app should feel like a dark movie theater and a 1940s title card at the same time:

- A near-black room lit by a faint golden marquee glow.
- Typewriter (screenplay) lettering in aged ivory.
- Film posters that start as sepia stills and "bloom" into full color when the spotlight (hover) hits them.
- Ornamental gold hairlines framing every surface, like a poster mat or a film frame.
- A whisper of film grain and vignette over everything.

There is **one theme only: dark**. Never add a light mode, and never use pure white or saturated modern hues (blues, greens, neon).

## 2. Color tokens

All colors are CSS custom properties declared on `:root` in `base.html`. **Never hardcode new colors; always use these tokens.** If a new need genuinely can't be met, extend the palette here first, staying inside the noir/gold/ivory family.

| Token | Value | Role |
|---|---|---|
| `--noir-black` | `#0b0b0c` | Page background; darkest ink. Also used as text color on gold fills. |
| `--noir-panel` | `#16130d` | Bottom stop of panel gradients. |
| `--noir-panel-2` | `#1f1a12` | Top stop of panel gradients (slightly warmer/lighter). |
| `--ivory` | `#f3ead3` | Primary text. |
| `--ivory-dim` | `#c9bfa6` | Secondary text: subtitles, credits, metadata, empty states. |
| `--gold` | `#c9a24b` | Accent base: small labels, scrollbar, "VS"/"X" separators. |
| `--gold-bright` | `#e7c66a` | Accent highlight: headings, links on hover, interactive borders, icons, focus rings. |
| `--crimson` | `#7a1818` | Deep red. Atmosphere only — the faint glow at the bottom of the page background. (The token is declared but currently consumed only as its literal `rgba(122, 24, 24, 0.18)` in the body gradient.) Do not use for text, buttons, or error states without extending this spec. |
| `--line` | `rgba(201, 162, 75, 0.35)` | **The universal hairline.** Every *gold* border, divider, and rule is `1px solid var(--line)`. The only non-gold borders are the black inset keylines (§4) and poster edges (§5). |

### Color rules

- **Text** is ivory (`--ivory` primary, `--ivory-dim` secondary). Lower emphasis further with `opacity: 0.6–0.7` on `--ivory-dim`, not with new grays.
- **Gold means interactive or emphasized.** Gold is for headings, labels, borders of hoverable things, and hover states — never for body copy.
- **Glows** are gold at low alpha: `rgba(201, 162, 75, 0.25–0.35)` in `box-shadow`. Shadows are black at high alpha: `rgba(0, 0, 0, 0.5–0.7)`.
- **No pure `#fff`**, no cool grays, no blue links. Links are ivory (`--ivory`, or `--ivory-dim` for metadata links) and turn `--gold-bright` on hover. (Pure `#000` does appear, but only in text-shadows and inset keylines.)

## 3. Typography

**One typeface: [Courier Prime](https://fonts.google.com/specimen/Courier+Prime)** (Google Fonts), fallback `monospace`. It is the screenplay/typewriter voice of the whole app — headings, body, buttons, tables, everything. Do not introduce a second typeface.

- Weights: 400 and 700, plus italics of both. The `<body>` carries class `courier-prime-regular`; new elements inherit it (`font-family: inherit` on form controls).
- **Fluid sizing**: use `clamp()` for anything that must scale, e.g. `clamp(0.85rem, 0.75rem + 0.5vw, 1rem)`. Inside cards (which set `container-type: inline-size`), use container-query units: `clamp(0.85rem, 9cqi, 1.5rem)`.
- **Section labels / eyebrows** (nav heading, table headers, "vote history"): small, uppercase, widely tracked gold — `font-size: 0.65–0.8rem; text-transform: uppercase; letter-spacing: 0.15–0.25em; color: var(--gold-bright)` (or `--gold` for the quietest ones), usually with a `--line` bottom border.
- **Headings**: `h1` is centered `--gold-bright` with `text-shadow: 0 2px 0 #000` (marquee). Card titles (`h2`) are `--ivory-dim` with `text-shadow: 0 1px 0 #000`.

## 4. Surfaces & framing

Every raised surface (card, input, drawer, table, modal, menu) uses the same **"framed panel"** recipe — this is the most recognizable element of the identity:

```css
background: linear-gradient(180deg, var(--noir-panel-2), var(--noir-panel));
border: 1px solid var(--line);              /* outer gold hairline   */
outline: 1px solid rgba(0, 0, 0, 0.6);      /* inner black keyline…  */
outline-offset: -5px;                        /* …inset like a mat     */
box-shadow: 0 8px 22px rgba(0, 0, 0, 0.5);  /* heavy black drop      */
```

- The **double frame** (gold hairline outside, inset black keyline) evokes a film frame / poster mat. Larger panels (duel cards, modals) use `outline-offset: -7px`; smaller ones `-5px`.
- **Corners are square.** `border-radius: 0` for cards and panels. Only tiny utilitarian controls get a radius: `2px` for the rating badge, `4px` for buttons/menus (burger, user widget, modal close). The only larger radii in the app are the **VS medallion** (a full circle, `50%`) and the modal **scrollbar thumb** (`6px`); don't add others.
- Full-height surfaces (side nav, dropdown menus) may deepen the gradient to `linear-gradient(180deg, var(--noir-panel-2), var(--noir-black))`.

### Page atmosphere

The `<body>` builds the theater. Don't recreate these on components; they are global:

- Background: two radial glows over black — gold from the top (`rgba(201,162,75,0.10)`), crimson from below the fold (`rgba(122,24,24,0.18)`).
- A fixed **vignette** (`body::before`, inset box-shadow, `z-index: 1`).
- A fixed **film-grain** overlay (`body::after`, inline-SVG `feTurbulence` at `opacity: 0.05`, `mix-blend-mode: overlay`, `z-index: 2`).
- All content renders above them (`body > *` gets `z-index: 3`).

### z-index registry

| Layer | z-index |
|---|---|
| Vignette | 1 |
| Film grain | 2 |
| Page content | 3 |
| Nav overlay (scrim) | 18 |
| Side nav drawer | 19 |
| Burger button, user widget | 20 |
| Nav toggle (invisible click target) | 21 |
| Modal overlay | 30 |

(The VS medallion uses `z-index: 5`, but inside the duel arena's own stacking context, not this global scale.)

New floating UI must slot into this table — update it when you do.

## 5. Imagery (posters)

Film posters are the only imagery in the app, always loaded from TMDB (`https://image.tmdb.org/t/p/w300…`).

- **At rest, posters are aged**: `filter: sepia(0.55) contrast(1.05) brightness(0.92) saturate(0.85);`
- **On hover/focus of their card, they bloom to color**: `filter: sepia(0) contrast(1.05) brightness(1) saturate(1.05);` with `transition: filter 0.35s ease`. This sepia-to-color bloom is a signature interaction — keep it on any new poster context.
- Inside modals (the "spotlight" view), posters are full color at rest.
- Posters get a thin dark border (`1px solid rgba(0,0,0,0.8)`) and a black drop shadow. Never round poster corners.

## 6. Motion

Motion is subtle and mechanical — a stage light moving, not a bouncy app:

- **Durations**: 0.2s (small color/border changes) to 0.35s (poster color bloom). Easing is always plain `ease`. No springs, no bounces, no delays.
- **Hover on cards**: lift with `transform: translateY(-4px)` (list items) to `-8px` (duel cards), border turns `--gold-bright`, and a gold glow joins the black shadow: `box-shadow: 0 0 28px rgba(201,162,75,0.35), 0 16px 40px rgba(0,0,0,0.7)`.
- **Modals**: overlay fades in (0.2s), dialog rises 16px while fading (0.25s).
- Drawers/menus slide (`transform: translateX(-100%)` → `0`, 0.3s ease).

## 7. Layout

- Centered single-column pages under a fixed chrome (burger top-left, user widget top-right). Body padding: `2.5rem 1.25rem 1rem`.
- **Layout tokens**: `--card-max: 360px` (one duel card), `--gap-max: 5rem` (duel gap), `--arena-max: calc(var(--card-max) * 2 + var(--gap-max))`. Content that should align with the duel arena uses `max-width: min(95vw, var(--arena-max))`.
- Other content widths: film list `min(95vw, 1100px)`, my-votes `min(95vw, 760px)`, admin table `min(95vw, 960px)`.
- Grids: `repeat(auto-fill, minmax(260px, 1fr))` for film lists. Gaps use `clamp()` (e.g. `clamp(0.6rem, 2vw, 1.1rem)`).
- **Responsive**: a single `640px` boundary with rules on both sides. At `max-width: 640px` the VS medallion hides, modal columns stack, and modal padding tightens; at `min-width: 641px` the duel gap widens to `clamp(3.5rem, 6vw, var(--gap-max))` so the medallion fits between the cards. Prefer fluid `clamp()`/`min()` sizing over extra breakpoints.

## 8. Component recipes

Reference implementations all live in `base.html`; match them exactly.

- **Duel arena** (`#duel`): two framed cards side by side, `flex` with a fluid gap; a **VS medallion** floats between them — a gold radial-gradient circle (`3.2rem`), black text, ringed by `box-shadow: 0 0 0 4px var(--noir-black), 0 0 0 5px var(--line)`.
- **Rating badge** (`.film-rating`): inline-block, `--gold-bright` text, `--line` hairline border, `2px` radius, padding `0.25em 0.9em`. This is the app's "chip"; reuse it for any small numeric/score display.
- **Text input** (`.film-search input`): framed-panel recipe; on focus, border `--gold-bright` + gold glow `0 0 16px rgba(201,162,75,0.25)`.
- **Buttons / small controls**: `--line` hairline border, `4px` radius, `--gold-bright` content; hover = gold border + glow. The burger and user widget are `2.6rem`-tall chips carrying the panel gradient; the modal close is a `2rem` square with a *transparent* background that inverts on hover (gold fill, black glyph).
- **Duel CTA** (`.duel-cta`): the button recipe as a link — gold "⚔ Duel this film" label, transparent background. On the film page it centers under the rating (`main > .duel-cta`); in the modal it sits under the rating in `.modal-film`; on film-list items a compact glyph-only variant (`.film-list-duel`) sits at the item's right edge with an `aria-label`.
- **Tables** (`.admin-user-table`): framed panel; uppercase gold letterspaced `thead`; `--line` row separators; row hover tints the row `rgba(201,162,75,0.06)` and lifts text from `--ivory-dim` to `--ivory`.
- **Modal**: scrim `rgba(0,0,0,0.7)`; dialog is a framed panel (`outline-offset: -7px`) with a gold aura `0 0 30px rgba(201,162,75,0.25)`; widths `min(92vw, 460px)` or `min(95vw, 800px)` (`.modal-dialog-wide`). Custom scrollbar: thin, gold thumb on a black track.
- **Navigation drawer** (`.side-nav`): CSS-only (checkbox hack, no JS); links separated by `--line` hairlines; hover turns a link gold and indents it (`padding-left: 1rem`).

## 9. Interaction & accessibility

- **Hover and focus should be twins**: pair every `:hover` rule with `:focus-within` (cards) or `:focus-visible` (controls). The reference implementation does this on duel cards and film-list items (`:hover, :focus-within`), the search input (`:focus`), the duel CTA (`:hover, :focus-visible`), and the nav toggle (`:focus-visible` ring: `outline: 2px solid var(--gold-bright); outline-offset: 2px`). Nav links currently lack focus styles — that's a known gap, not a pattern to copy.
- State changes never rely on color alone — pair them with the lift, glow, indent, or fill described above.
- Decorative icons (burger bars via `aria-hidden` label, the SVG user icon) are hidden from assistive tech; the VS medallion is CSS generated content (`#duel::before`), not markup. Invisible interactive stand-ins (the nav checkbox) get `aria-label`s.
- Prefer CSS-only mechanisms (checkbox drawer, `<details>` menu, `:has()` state switches) over custom JavaScript. Interactivity comes from HTMX; the only inline script in the chrome reads the `user_name` cookie.

## 10. Voice (copy on screen)

- UI labels are terse English ("Films list", "My votes", "Login", "Logout", "Console").
- Structural labels are the uppercase letterspaced gold eyebrows (§3).
- Editorial copy (intro text) is conversational and first-person, in dimmed ivory, justified, and never wider than the duel arena.
- **Brand mark**: the favicon is an inline-SVG noir-black rounded square with a bold Courier "F" in `--gold-bright` — the identity compressed to one glyph. Reuse it wherever a tiny mark is needed.

## 11. Do / Don't checklist

Quick gate before shipping UI. Everything here restates a rule above.

**Do**

- Use only the `:root` tokens for color; `var(--line)` for every gold border and divider.
- Use the framed-panel recipe (§4) for any new surface, and Courier Prime for all text.
- Keep posters sepia-at-rest with color bloom on hover (§5).
- Pair every hover state with a focus state; use `ease` transitions of 0.2–0.35s.
- Add new floating UI to the z-index registry (§4).

**Don't**

- Don't introduce new colors, fonts, or a light theme.
- Don't use pure white, blues, or saturated modern hues; don't use crimson outside the page-background glow.
- Don't round corners beyond `4px` (sole existing exceptions, §4: the circular VS medallion and the `6px` scrollbar thumb), and never on cards or posters.
- Don't add bouncy/spring animations or transition durations over 0.35s.
- Don't pull in CSS frameworks or component libraries; styling lives in `base.html` and follows these recipes.
