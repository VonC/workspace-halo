import assert from "node:assert/strict";
import test from "node:test";
import {
  branchFromHead,
  gitdirPointer,
  isHexColor,
  optionalWorkspaceScopedValue,
  randomHaloColor,
  resolveSharedColor,
  savedWorkspaceName,
  selectLogo,
  selectRootName,
  workspaceScopedValue
} from "../src/model";

test("the workspace name comes only from a saved code-workspace file", () => {
  assert.equal(savedWorkspaceName("llm-shared.code-workspace"), "llm-shared");
  assert.equal(savedWorkspaceName("My Folder"), undefined);
  assert.equal(savedWorkspaceName(".code-workspace"), undefined);
  assert.equal(savedWorkspaceName(undefined), undefined);
});

test("workspace-scoped values ignore global values by construction", () => {
  assert.equal(
    workspaceScopedValue({ workspaceFolderValue: 12, workspaceValue: 8 }, 4),
    12
  );
  assert.equal(workspaceScopedValue({ workspaceValue: 8 }, 4), 8);
  assert.equal(workspaceScopedValue(undefined, 4), 4);
  assert.equal(optionalWorkspaceScopedValue({}), undefined);
});

test("only the exact workspace-named logo activates", () => {
  const selection = selectLogo("my-project", [
    "other.logo.png",
    "my-project.logo.png",
    "readme.txt"
  ]);
  assert.equal(selection.exactLogo, "my-project.logo.png");
  assert.deepEqual(selection.logoFiles, ["my-project.logo.png", "other.logo.png"]);
  assert.equal(selectLogo("My-Project", ["my-project.logo.png"]).exactLogo, undefined);
});

test("a root synonym is accepted when no folder name matches", () => {
  assert.equal(selectRootName("setupsenv", ["custom", "senv"], ["senv"]), "senv");
  assert.equal(selectRootName("setupsenv", ["custom", "senv"], []), undefined);
  assert.equal(selectRootName("setupsenv", ["custom", "senv"], ["other"]), undefined);
  assert.equal(
    selectRootName("my-project", ["alias", "my-project"], ["alias"]),
    "my-project"
  );
  assert.equal(selectRootName("my-project", ["a", "b"], ["b", "a"]), "a");
});

test("a linked worktree is recognized by its gitdir pointer", () => {
  assert.equal(
    gitdirPointer("gitdir: C:/git/halo/.git/worktrees/feat\n"),
    "C:/git/halo/.git/worktrees/feat"
  );
  assert.equal(gitdirPointer("[core]\n"), undefined);
});

test("the worktree branch comes from its HEAD", () => {
  assert.equal(branchFromHead("ref: refs/heads/feat/two-lines\n"), "feat/two-lines");
  assert.equal(branchFromHead("0123456789abcdef0123456789abcdef01234567\n"), "0123456");
  assert.equal(branchFromHead("garbage"), undefined);
});

test("workspace Peacock color takes precedence over Workspace Halo color", () => {
  assert.equal(resolveSharedColor("#112233", "#445566", "#ff2d55"), "#112233");
  assert.equal(resolveSharedColor(undefined, "#445566", "#ff2d55"), "#445566");
  assert.equal(resolveSharedColor("blue", "invalid", "#ff2d55"), "#ff2d55");
  assert.equal(resolveSharedColor(undefined, undefined, "#eb2323"), "#eb2323");
  assert.equal(isHexColor("#0Fa2c4"), true);
  assert.equal(isHexColor("red"), false);
});

test("an assigned color is a vivid six-digit hex", () => {
  assert.equal(randomHaloColor(() => 0), "#eb2323");
  assert.equal(randomHaloColor(() => 0.5), "#23ebeb");
  assert.notEqual(randomHaloColor(() => 0.1), randomHaloColor(() => 0.7));
  for (let draw = 0; draw < 100; draw += 1) {
    assert.match(randomHaloColor(Math.random), /^#[0-9a-f]{6}$/);
  }
});
