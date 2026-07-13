import assert from "node:assert/strict";
import test from "node:test";
import {
  optionalWorkspaceScopedValue,
  resolveSharedColor,
  selectLogo,
  workspaceScopedValue
} from "../src/model";

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

test("workspace Peacock color takes precedence over Workspace Halo color", () => {
  assert.equal(resolveSharedColor("#112233", "#445566", "#ff2d55"), "#112233");
  assert.equal(resolveSharedColor(undefined, "#445566", "#ff2d55"), "#445566");
  assert.equal(resolveSharedColor("blue", "invalid", "#ff2d55"), "#ff2d55");
});

