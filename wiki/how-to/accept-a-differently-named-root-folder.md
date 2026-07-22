# Accept a differently named root folder

Goal: show a halo for a workspace whose name matches none of its root folder
names. Example: the saved workspace `my-tools.code-workspace` whose only root
folder is named `tools`. Without help, Workspace Halo finds no root to
inspect and stays inactive.

## Declare the accepted folder names

Add the folder name (or names) to the workspace settings, in the `settings`
block of the `.code-workspace` file:

```json
"workspaceHalo.rootSynonyms": ["tools"]
```

The extension then inspects the `tools` root folder as if it carried the
workspace name.

## Keep the logo named after the workspace

The logo file name never changes with synonyms: it always repeats the
workspace name, inside the matched root folder.

```text
tools/
`-- .vscode/
    `-- my-tools.logo.png
```

## Know which root wins in a multi-root workspace

Only one root folder is ever inspected:

1. a root folder whose name equals the workspace name exactly always wins;
2. otherwise the first folder in root order whose name appears in
   `workspaceHalo.rootSynonyms` is used;
3. other root folders are ignored entirely.

The complete matching rules, including how the workspace name itself is
derived, are in [activation conditions](../reference/activation-conditions.md).
