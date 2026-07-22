# Workspace Halo wiki

Workspace Halo identifies each Visual Studio Code window on Windows 11 with a
halo: a colored border, the workspace name, and its logo, drawn over the window
exactly when identification matters and hidden the rest of the time. The
[project README](../README.md) shows the two-file quick start; this wiki holds
everything else: the reasoning, the guided first runs, the task recipes, and
the exact formats and conditions.

The halo pairs naturally with the
[Peacock](https://marketplace.visualstudio.com/items?itemName=johnpapa.vscode-peacock)
extension: when a workspace has a Peacock color, the halo takes that same
color, so one hue identifies the workspace both inside and around the window.
See [take the halo color from Peacock](how-to/take-the-halo-color-from-peacock.md).

The wiki follows [Diátaxis](https://diataxis.fr/) and always presents its four
purposes in this order: explanation, tutorials, how-to guides, then reference.

## 💡 Explanation

Background and reasoning: understand why the extension is built this way.

- [Why a native host runs beside the extension](explanation/why-a-native-host-beside-the-extension.md)
- [Why the halo shows only on triggers](explanation/why-the-halo-shows-only-on-triggers.md)
- [How the host finds its VS Code window](explanation/how-the-host-finds-its-window.md)
- [How the overlay stays inside its window](explanation/how-the-overlay-stays-inside-its-window.md)
- [How colors keep their contrast](explanation/how-colors-keep-their-contrast.md)

## 🎓 Tutorials

Learning by doing: follow the steps in order and inspect the result.

- [01 - Halo your first workspace](tutorials/01-halo-your-first-workspace.md)
- [02 - Style the halo](tutorials/02-style-the-halo.md)

## 🧭 How-to guides

Recipes for one precise goal each, for readers who already know the basics.

- [Take the halo color from Peacock](how-to/take-the-halo-color-from-peacock.md)
- [Accept a differently named root folder](how-to/accept-a-differently-named-root-folder.md)
- [Troubleshoot a missing halo](how-to/troubleshoot-a-missing-halo.md)
- [Install or update the extension](how-to/install-or-update-the-extension.md)
- [Set up the build toolchain](how-to/set-up-the-build-toolchain.md)
- [Build and package the VSIX](how-to/build-and-package-the-vsix.md)
- [Publish to the Visual Studio Marketplace](how-to/publish-to-the-marketplace.md)
- [Sign the native host for free](how-to/sign-the-native-host-for-free.md)

## 📖 Reference

Exact descriptions of conditions, settings, triggers, commands, and files.

- [Activation conditions](reference/activation-conditions.md)
- [Settings](reference/settings.md)
- [Display triggers](reference/display-triggers.md)
- [Native host command line](reference/native-host-command-line.md)
- [Logs and processes](reference/logs-and-processes.md)
- [Repository layout](reference/repository-layout.md)
