# Show the branch of a Git worktree

Goal: tell apart several VS Code windows opened on the same project through
Git worktrees, with each halo showing its branch under the project name.

## Open the worktree through its own workspace file

The `.vscode` directory is versioned with the project, so a linked worktree
already carries the workspace file and the logo:

```bat
git worktree add ..\my-project-feat -b feat/two-lines
code ..\my-project-feat\.vscode\my-project.code-workspace
```

The halo shows `my-project` with its logo, and below it, in italic with the
same font and size, `feat/two-lines`. The two lines are centered together.

The worktree folder name does not matter (`my-project-feat` here): a folder
that is a linked worktree qualifies as soon as its main working tree also
holds `.vscode\my-project.code-workspace`. The logo is read from the
worktree's own `.vscode\my-project.logo.png`.

## Switch branches

Run `git switch` in the worktree: the extension watches that worktree's
HEAD and redraws the second line with the new branch. A detached HEAD shows
the abbreviated commit instead.

## Show the branch outside a worktree

The main working tree shows the name only. To add its branch too, set in
its workspace settings:

```json
"workspaceHalo.showBranch": true
```

## Know what is remembered

The Git detection (where HEAD lives, and whether the root is a linked
worktree of the same workspace) runs once and is memorized in the workspace
state; later refreshes only re-read HEAD. The exact rules are in
[activation conditions](../reference/activation-conditions.md).
