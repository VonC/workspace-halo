# How colors keep their contrast

One color identifies a workspace: the border and the workspace name always
share it. This page explains where that color comes from and how the renderer
guarantees the name stays readable over any content.

## One identity color, Peacock first

The shared color is resolved in priority order:

1. a workspace-scoped `peacock.color` value in `#rrggbb` form;
2. the `workspaceHalo.color` setting;
3. the built-in default `#ff2d55`.

Peacock wins by design: if the
[Peacock](https://marketplace.visualstudio.com/items?itemName=johnpapa.vscode-peacock)
extension already colors the inside of the window (title bar, activity bar,
status bar), the halo extends that same identity outside the window instead of
introducing a second hue. Only exact six-digit hex values are honored; Peacock
color names such as `red` are ignored, as detailed in
[take the halo color from Peacock](../how-to/take-the-halo-color-from-peacock.md).

## The name pill picks its pole by WCAG luminance

The workspace name is drawn on a rounded pill so it never depends on what
happens to be behind it. The pill color is computed from the text color:

- the WCAG relative luminance of the text picks the higher-contrast pole,
  white under dark text and black under light text (the break-even luminance
  is about 0.179);
- one eighth of the text color tints that pole, so the pill visibly carries
  the workspace hue instead of being flat white or black;
- if that tint would drop the contrast ratio under 3:1, the WCAG minimum for
  large text (and the halo name is always large), the pure pole wins; the
  pole itself always reaches at least 4.58:1.

An optional dark shadow behind the letters adds a second safety margin, and
both the pill and the shadow can be turned off in the
[settings](../reference/settings.md).

## Dithered opacity on a binary overlay

The child overlay uses color-key transparency, where each pixel is either
fully opaque or fully absent. A pill opacity percentage is therefore rendered
as the matching share of pill pixels, spread evenly by a 4x4 Bayer
ordered-dither matrix: 40 percent opacity keeps 40 percent of the pixels. At
normal viewing distance the dither reads as a uniform translucency.

## A border that survives any background

By default the border alternates segments of the halo color with segments of
opaque black (50 pixels each). Whatever sits behind the window, light
wallpaper or dark editor, at least one of the two tones stays visible, and
the black runs are painted rather than transparent so the frame always reads
as one unbroken rectangle. Setting the segment length to zero switches to the
continuous `solid`, `double`, `dashed`, or `dotted` motifs.
