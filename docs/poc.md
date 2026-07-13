# Windows 11 Alt+Tab proof of concept

## Acceptance criteria

The proof of concept passes only when all of the following are true:

- the overlay is bound to one specific stable VS Code window;
- it is above that VS Code window but below unrelated covering windows;
- its border, workspace name, and PNG logo appear in that VS Code window's
  Windows 11 Alt+Tab miniature;
- pressing and releasing Shift twice within 400 milliseconds while VS Code is
  focused displays the same overlay over the live window;
- the first subsequent input directed to VS Code hides the manual overlay.

## Build

From PowerShell:

```powershell
.\scripts\build-companion.ps1
```

The script uses the installed Go toolchain and keeps its build cache inside the
repository. It does not require Visual Studio or third-party Go modules.

## Manual experiment

1. Open a VS Code integrated terminal in the window under test.
2. Build the companion.
3. Run this command while that VS Code window is focused:

   ```powershell
   .\bin\workspace-halo-companion.exe --name "my-project" --logo ".vscode\my-project.logo.png"
   ```

4. Press Alt+Tab and inspect the VS Code miniature.
5. Return to VS Code and press and release Shift twice quickly.
6. Move the mouse or press a key to confirm that the manual overlay disappears.

The companion log defaults to `%TEMP%\workspace-halo-companion.log`.

