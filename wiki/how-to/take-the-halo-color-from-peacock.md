# Take the halo color from Peacock

Goal: one color per workspace, applied by
[Peacock](https://marketplace.visualstudio.com/items?itemName=johnpapa.vscode-peacock)
inside the window (title bar, activity bar, status bar) and by Workspace Halo
around it. A valid workspace-scoped `peacock.color` always takes priority over
`workspaceHalo.color`.

## Set the Peacock color for the workspace

1. Install the Peacock extension.
2. In the workspace, run **Peacock: Change to a Favorite Color** (or
   **Surprise Me With a Random Color**) from the command palette.

Peacock writes `peacock.color` into the workspace settings and recolors the
VS Code chrome. Workspace Halo detects the change, restarts its renderer, and
the next halo uses the same color for its border and name.

## Verify the value is a six-digit hex

Workspace Halo only honors `peacock.color` values matching `#rrggbb`, for
example `#832561`. Peacock's **Enter a Specific Color** command also accepts
named colors such as `red` and stores them as-is; those are ignored by the
halo, which then falls back to `workspaceHalo.color` or the workspace's
remembered randomly assigned color.

If your halo does not follow Peacock, open the workspace settings and replace
the named color with its hex form:

```json
"peacock.color": "#ff0000"
```

## Keep the value workspace-scoped

The color must live in the workspace settings (the `settings` block of the
`.code-workspace` file) or in the matching folder's `.vscode\settings.json`.
A `peacock.color` in your global user settings is ignored, deliberately: a
user-level color could not identify one workspace among many.

## Choose which setting owns the color

- To let Peacock drive: set `peacock.color` and remove `workspaceHalo.color`;
  the halo follows every Peacock change.
- To pin the halo independently: remove the workspace-scoped `peacock.color`,
  or set `workspaceHalo.color` and accept that it only applies while no valid
  Peacock color exists.

The full resolution order is described in
[how colors keep their contrast](../explanation/how-colors-keep-their-contrast.md).
