---
name: visualize
description: Use when the user wants an animation, chart, graph, diagram, simulation or any "show me" visual in the chat.
---

# Live visuals in the chat

Call the Artifact tool with `kind: "canvas"`. The visual runs live in the chat. `content` is plain JavaScript, with no HTML, libraries or network.

## The contract

- Define `function draw(ctx, w, h, t)`. It is called every frame. `t` = seconds since start; the canvas is cleared before each call. `w`, `h` = size in CSS pixels.
- Everything that moves must depend on `t`. Static drawings work too: just ignore `t`.
- Optional `function setup(ctx, w, h)` runs once first (precompute data there).
- Colours that match the app: `nox.colors.fg`, `.dim`, `.grid`, `.accent`, `.bg`, `nox.colors.palette[i]` (8 colours), `nox.hsl(hue)`.

## Helpers

| Call | Draws |
|---|---|
| `nox.bar(ctx, {labels, values, title, t})` | bar chart, bars grow in |
| `nox.line(ctx, {labels, series:[{name, values}], title, t})` | line chart, lines draw in |
| `nox.pie(ctx, {labels, values, title, t, donut: true})` | pie or donut |
| `nox.text(ctx, s, x, y, {size, color, align, bold})` | text |
| `nox.rand(seed)` | returns a repeatable `random()` |
| `nox.ease(p)`, `nox.lerp(a, b, p)`, `nox.clamp(v, a, b)` | smooth motion |

Pass `t` to the chart helpers to get the grow-in animation; leave it out for a still chart.

## Steps

1. Pick the closest example and Read it: `examples/bar-chart.js`, `examples/line-chart.js`, `examples/animation.js` (moving things), `examples/diagram.js` (boxes and arrows). They are in this skill's folder.
2. Adapt it to the user's data or idea. Keep it under ~80 lines. Use the user's language for labels.
3. Call Artifact: `{"id": "short-id", "title": "…", "kind": "canvas", "content": "<the JS>"}`.
4. Answer in one or two sentences: what it shows. Do not paste the code into the reply.
5. If the user reports an error (it is shown under the visual), fix it and call Artifact again with the same `id`.

## Rules

- Use `w` and `h`, never fixed sizes, so it fits any window.
- Never `alert`, `fetch`, `import`, `setInterval` or a `while(true)` loop. `draw` is the loop.
- For many particles create them once in `setup` with `nox.rand(1)`, move them in `draw` with `t`.
- A whole web page with buttons and forms is `kind: "html"`, not canvas.
