# Tutorial 01 - Halo your first workspace

In this tutorial you create a fresh workspace, mark it with a logo, and watch
its halo appear on every display trigger. At the end you will recognize each
trigger and know where the extension logs what it does.

## What you need before starting

- Windows 11 with stable desktop VS Code (`Code.exe`; Insiders and web VS
  Code are not supported).
- The `workspace-halo-win32-x64.vsix` file, built from this repository (see
  [build and package the VSIX](../how-to/build-and-package-the-vsix.md)) and
  installed as described in
  [install or update the extension](../how-to/install-or-update-the-extension.md).
- Any square-ish PNG image to use as a logo.

## Step 1 - Create and save the workspace

1. Create an empty folder named `halo-demo` and open it in VS Code with
   **File > Open Folder...**.
2. Choose **File > Save Workspace As...**, navigate into `halo-demo\.vscode`
   (create the `.vscode` folder in the dialog), and save the workspace as
   `halo-demo.code-workspace`.

VS Code reopens the window on the saved workspace and its title now shows
`halo-demo (Workspace)`. The workspace is named `halo-demo`, after its file
name, and it matches its folder name: that match is the first activation
condition.

## Step 2 - Add the logo

Copy your PNG into the workspace as `.vscode\halo-demo.logo.png`. The file
name must repeat the workspace name exactly, case included.

```text
halo-demo/
`-- .vscode/
    |-- halo-demo.code-workspace
    `-- halo-demo.logo.png
```

Within a second or two the activation halo appears: a pink `#ff2d55` border
around the window, the name `halo-demo` on its pill in the center, and your
logo in the lower right corner. No reload is needed; the extension watches
`.vscode` for logo files.

## Step 3 - Acknowledge the activation halo

Click anywhere inside the VS Code window. The halo disappears: the click
tells Workspace Halo you have identified the window. From now on the focused
window stays clean.

## Step 4 - Call the halo back with double-Shift

Press and release **Shift** twice within 400 milliseconds. The halo comes
back over the live window, and the very next input (a key, a mouse move)
hides it again. This is the on-demand trigger: use it whenever you want to
double-check which workspace you are in.

## Step 5 - See the halo where it matters

Try each ambient trigger:

1. Hold **Alt** and press **Tab**: the Alt+Tab miniature and the window both
   show the halo for as long as Alt stays down.
2. Move the mouse onto the Windows taskbar: the halo shows while the cursor
   rests there, where thumbnail previews appear.
3. Minimize the window: its taskbar and Alt+Tab thumbnails keep the halo.
4. Cover the window partially with another application while some other
   window has focus: the halo appears on the covered VS Code window.

## Step 6 - Read what happened

Open **View > Output** and select **Workspace Halo** in the dropdown. You
will see the tracking line with the root and logo paths, the native host
process id, and the path of its `native-host.log`, where each visibility
change is logged with its reason.

## Where to go next

- Style the border, name, and pill in
  [tutorial 02 - Style the halo](02-style-the-halo.md).
- Read the exact rules in [activation conditions](../reference/activation-conditions.md)
  and [display triggers](../reference/display-triggers.md).
