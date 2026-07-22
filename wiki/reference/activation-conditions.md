# Activation conditions

Workspace Halo activates for a VS Code window only when every condition below
holds. While any of them is false, the extension is completely inert for that
window: no halo, no native host process, no command, no status-bar item.

## Environment gate

| Condition | Exact rule |
| --- | --- |
| Operating system | `process.platform` is `win32` (Windows 11 is the supported target) |
| VS Code flavor | Stable desktop VS Code; the host binds only to `Code.exe` windows |
| Extension kind | `ui` extension; web VS Code and remote UI are unsupported |
| Activation event | `onStartupFinished` |

## Workspace name derivation

- If the window has a workspace file on disk, the workspace name is the
  `.code-workspace` file name without its suffix: `my-tools.code-workspace`
  gives `my-tools`, wherever the file is saved.
- Otherwise (a plain opened folder), the workspace name is the display name
  of the workspace, which is the folder name.

## Root folder selection

Exactly one root folder is inspected:

1. the first folder whose name equals the workspace name, or
2. when none matches, the first folder in root order whose name appears in
   the `workspaceHalo.rootSynonyms` array.

If neither exists, or the matched root is not a local `file` folder, the
extension stays inactive. All name comparisons are case-sensitive and
preserve spaces.

## Logo file requirement

Inside the matched root folder, the `.vscode` directory must contain the file
`<workspace name>.logo.png`, matching the workspace name exactly (case
included), and it must be a decodable PNG. A missing `.vscode` directory, a
missing exact file, or an unreadable image keeps the extension inactive.

When several `*.logo.png` files exist in that directory, the exact one is
still used, and a warning listing the extra files is written to the developer
console, the **Workspace Halo** output channel, and the native-host log.

## Reactions to change

The extension watches its conditions and reconciles after a 150 millisecond
debounce:

| Watched change | Effect |
| --- | --- |
| `**/.vscode/*.logo.png` created, changed, or deleted | re-evaluate; restart or stop the host |
| Workspace folders added or removed | re-evaluate the root selection |
| `workspaceHalo.*` or `peacock.color` configuration | re-evaluate the settings fingerprint |
| Window gains focus | immediate re-evaluation |

A configuration fingerprint (workspace name, root, logo path with size and
modification time, resolved settings, warning state) decides whether the
running host is kept, restarted, or stopped. A host that exits on its own is
restarted after one second while the conditions still hold.
