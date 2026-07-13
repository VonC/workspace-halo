import * as esbuild from "esbuild";

await esbuild.build({
  entryPoints: ["src/extension.ts"],
  bundle: true,
  external: ["vscode"],
  format: "cjs",
  minify: process.argv.includes("--production"),
  outfile: "dist/extension.js",
  platform: "node",
  sourcemap: true,
  target: "node20"
});

