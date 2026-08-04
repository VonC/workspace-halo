# Tutorial 02 - Style the halo

Starting from the `halo-demo` workspace of
[tutorial 01](01-halo-your-first-workspace.md), you now restyle its halo step
by step: color, border, name, pill, and logo. Every change below takes effect on
its own; the extension watches the settings and restarts its renderer, so
after each save the next halo already uses the new values.

## Step 1 - Open the folder settings

Create or open `.vscode\settings.json` inside the `halo-demo` folder.
Workspace Halo reads workspace and workspace-folder values only; the same
keys in your global user settings are intentionally ignored, so one
workspace's style never leaks into another.

## Step 2 - Change the identity color

```json
{
  "workspaceHalo.color": "#00b7ff"
}
```

Press double-Shift: the border and the name are now blue, and the pill
adapted its own tone automatically to keep the name readable (see
[how colors keep their contrast](../explanation/how-colors-keep-their-contrast.md)).

## Step 3 - Tune the border

```json
{
  "workspaceHalo.color": "#00b7ff",
  "workspaceHalo.borderWidth": 24,
  "workspaceHalo.borderSegment": 80
}
```

The border doubles in width and its alternating blue and black segments
lengthen. Now try a continuous motif instead: set
`"workspaceHalo.borderSegment": 0` and `"workspaceHalo.borderMotif": "double"`,
then compare with `"dashed"` and `"dotted"`.

## Step 4 - Shape the name and its pill

```json
{
  "workspaceHalo.fontFamily": "Cascadia Code",
  "workspaceHalo.fontWeight": 400,
  "workspaceHalo.pillOpacity": 40,
  "workspaceHalo.pillMargin": 120
}
```

The name switches to a lighter monospace, the pill becomes translucent (a
dithered 40 percent), and wider margins keep the pill further from the window
edges. To see the name without any pill, set
`"workspaceHalo.namePill": false`; without its shadow, set
`"workspaceHalo.textShadow": false`.

## Step 5 - Scale the logo

If you added the optional `.vscode\halo-demo.logo.png` in tutorial 01, make
it more present:

```json
{
  "workspaceHalo.logoScale": 60
}
```

The logo's longer side now takes 60 percent of the window height instead of
the default 33, still anchored in the lower right corner. Whatever the
value, the logo keeps its aspect ratio and stays inside the border padding,
so even 100 cannot push it over the border. Without a logo file this key
changes nothing.

## Step 6 - Check the result on every trigger

Minimize the window and hover its taskbar thumbnail, then hold Alt+Tab: the
restyled halo follows into every miniature, since thumbnails are composed
from the same overlay.

## Where this leaves you

You have styled one workspace without touching any other. The full list of
keys, ranges, and defaults is in [settings](../reference/settings.md), and
letting Peacock provide the color instead of `workspaceHalo.color` is the
subject of
[take the halo color from Peacock](../how-to/take-the-halo-color-from-peacock.md).
