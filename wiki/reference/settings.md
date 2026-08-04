# Settings

All keys live under `workspaceHalo.*`, plus the honored external
`peacock.color`. For every key, the workspace-folder value wins over the
workspace value, and global user values are intentionally ignored: a halo
style always belongs to one workspace.

Workspace values sit in the `settings` block of the `.code-workspace` file;
workspace-folder values sit in the matched root folder's
`.vscode\settings.json`. Every change is picked up automatically and restarts
the renderer.

## Halo settings

| Setting | Type | Default | Range | Meaning |
| --- | --- | --- | --- | --- |
| `workspaceHalo.color` | string | (assigned) | `#rrggbb` | Shared border and workspace-name color; when unset, a random color is assigned to the workspace and remembered |
| `workspaceHalo.borderWidth` | integer | `12` | 1 to 64 | Border width in pixels |
| `workspaceHalo.borderMotif` | string | `solid` | `solid`, `double`, `dashed`, `dotted` | Continuous border motif, drawn when `borderSegment` is 0 |
| `workspaceHalo.borderSegment` | integer | `50` | 0 or more | Length in pixels of the alternating opaque black border segments; 0 draws the continuous motif instead |
| `workspaceHalo.logoScale` | integer | `33` | 1 to 100 | Length of the logo's longer side as a percentage of the window height, when a logo file exists; aspect ratio is kept and the logo stays inside the border padding |
| `workspaceHalo.fontFamily` | string | `Segoe UI` | installed family | Windows font family of the workspace name |
| `workspaceHalo.fontWeight` | integer | `700` | 1 to 1000 | Windows font weight of the workspace name |
| `workspaceHalo.textShadow` | boolean | `true` | | Draw a dark shadow behind the name |
| `workspaceHalo.namePill` | boolean | `true` | | Draw the contrast pill behind the name |
| `workspaceHalo.pillOpacity` | integer | `100` | 0 to 100 | Pill opacity percentage, rendered by ordered dithering |
| `workspaceHalo.pillMargin` | integer | `50` | 0 or more | Minimal left and right margin, in pixels, between the window edges and the pill with its name |
| `workspaceHalo.rootSynonyms` | string array | `[]` | | Extra root folder names accepted when none matches the workspace name; window-scoped, so it belongs in the workspace settings |

## Peacock color priority

A workspace-scoped `peacock.color` matching the regular expression
`^#[0-9a-fA-F]{6}$` takes priority over `workspaceHalo.color`. Any other form
(named colors, three-digit hex, missing value) leaves `workspaceHalo.color`
in charge, with the workspace's remembered randomly assigned color as the
final fallback. The border and the name
always share the resulting color; the pill derives its own tone from it, as
explained in
[how colors keep their contrast](../explanation/how-colors-keep-their-contrast.md).

## Scope summary

Every `workspaceHalo.*` key is resource-scoped and read against the matched
root folder, except `workspaceHalo.rootSynonyms`, which is window-scoped
because it decides which root folder is matched in the first place.
