# Native host command line

`workspace-halo-host.exe` is launched by the extension with one flag per
resolved setting; it can also be run by hand for diagnostics. Invalid values
make it exit with code 1 and a `configuration error` message on stderr.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--name` | (required) | Workspace name displayed in the overlay |
| `--logo` | (required) | Path of the PNG logo |
| `--color` | `#ff2d55` | Shared border and text color, `#RGB` or `#RRGGBB` |
| `--border-width` | `12` | Border width in pixels, positive |
| `--border-style` | `solid` | `solid`, `double`, `dashed`, or `dotted` |
| `--border-segment` | `50` | Length of the alternating black border segments; 0 for a continuous border |
| `--font` | `Segoe UI` | Installed Windows font family |
| `--font-weight` | `700` | Windows font weight, 1 to 1000 |
| `--shadow` | `true` | Draw the dark text shadow |
| `--pill` | `true` | Draw the contrast pill behind the name |
| `--pill-opacity` | `100` | Pill opacity percentage, 0 to 100 |
| `--pill-margin` | `50` | Minimal left and right margin of the pill and name, in pixels |
| `--log` | `%TEMP%\workspace-halo-companion.log` | Log file path |
| `--window-mode` | `owned` | Overlay attachment: `owned` or `child` (the extension always passes `child`) |
| `--focus-handshake` | `false` | Require the launching extension to confirm the window over stdin |
| `--wait-for-vscode` | `1m` | Without a handshake, how long to wait for a stable VS Code window to become foreground |
| `--hwnd` | (none) | Bind to this HWND, decimal or `0x` hex; must belong to `Code.exe` unless `--allow-any-window` |
| `--allow-any-window` | `false` | Diagnostics: allow binding to a non-`Code.exe` window |
| `--startup-warning` | (none) | Warning text copied from the extension into the host log |

## Standard input commands

With `--focus-handshake`, the host reads line commands of the form
`<command> <token>`:

| Command | Sender meaning |
| --- | --- |
| `focus <token>` | The launching window believes it is focused |
| `blur <token>` | The launching window lost focus; the pending candidate is dropped |
| `confirm <token>` | The extension confirms the proposed candidate for that token |

## Standard output messages

Protocol lines start with `workspace-halo-`; every other output line is
relayed to the **Workspace Halo** output channel as plain log text.

| Message | Meaning |
| --- | --- |
| `workspace-halo-bound startup` | A unique title match bound the window without focus |
| `workspace-halo-ready` | Title match unavailable; the focus handshake is open |
| `workspace-halo-candidate <token>` | The foreground `Code.exe` window is proposed for confirmation |
| `workspace-halo-rejected <token>` | The proposal failed or became stale; a retry may follow |
| `workspace-halo-bound <token>` | The handshake confirmed the window; the overlay starts |

The handshake sequence and its rationale are described in
[how the host finds its window](../explanation/how-the-host-finds-its-window.md).

## Manual diagnostic run

From a repository checkout, the proof-of-concept flow still works: build the
companion and run it against the focused window, as recorded in
[docs/poc.md](../../docs/poc.md):

```powershell
.\bin\workspace-halo-companion.exe --name "my-project" --logo ".vscode\my-project.logo.png"
```

Without `--focus-handshake` the host waits up to `--wait-for-vscode` for a
stable VS Code window to become foreground, then binds to it.
