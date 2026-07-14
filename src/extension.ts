import { ChildProcess, spawn } from "node:child_process";
import * as path from "node:path";
import * as vscode from "vscode";
import {
  logicalWorkspaceName,
  optionalWorkspaceScopedValue,
  resolveSharedColor,
  selectLogo,
  selectRootName,
  workspaceScopedValue
} from "./model";

interface HaloSettings {
  readonly color: string;
  readonly borderWidth: number;
  readonly borderMotif: "solid" | "double" | "dashed" | "dotted";
  readonly fontFamily: string;
  readonly fontWeight: number;
  readonly textShadow: boolean;
}

interface Registration {
  readonly workspaceName: string;
  readonly root: vscode.WorkspaceFolder;
  readonly logo: vscode.Uri;
  readonly settings: HaloSettings;
  readonly warning?: string;
  readonly fingerprint: string;
}

class WorkspaceHaloController implements vscode.Disposable {
  private readonly output = vscode.window.createOutputChannel("Workspace Halo", { log: true });
  private readonly disposables: vscode.Disposable[] = [];
  private host: ChildProcess | undefined;
  private registration: Registration | undefined;
  private refreshTimer: NodeJS.Timeout | undefined;
  private refreshGeneration = 0;
  private lastWarning: string | undefined;
  private stopping = false;
  private disposed = false;

  public constructor(private readonly context: vscode.ExtensionContext) {}

  public async start(): Promise<void> {
    if (process.platform !== "win32") {
      return;
    }

    const logoWatcher = vscode.workspace.createFileSystemWatcher("**/.vscode/*.logo.png");
    this.disposables.push(
      logoWatcher,
      logoWatcher.onDidCreate(() => this.scheduleRefresh()),
      logoWatcher.onDidChange(() => this.scheduleRefresh()),
      logoWatcher.onDidDelete(() => this.scheduleRefresh()),
      vscode.workspace.onDidChangeWorkspaceFolders(() => this.scheduleRefresh()),
      vscode.workspace.onDidChangeConfiguration((event) => {
        if (
          event.affectsConfiguration("workspaceHalo")
          || event.affectsConfiguration("peacock.color")
        ) {
          this.scheduleRefresh();
        }
      }),
      vscode.window.onDidChangeWindowState((state) => {
        if (state.focused) {
          this.scheduleRefresh(0);
        }
      })
    );

    await this.refresh();
  }

  public dispose(): void {
    this.disposed = true;
    if (this.refreshTimer !== undefined) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = undefined;
    }
    for (const disposable of this.disposables) {
      disposable.dispose();
    }
    this.disposables.length = 0;
    void this.stopHost();
    this.output.dispose();
  }

  public async shutdown(): Promise<void> {
    this.disposed = true;
    await this.stopHost();
  }

  private scheduleRefresh(delay = 150): void {
    if (this.disposed) {
      return;
    }
    if (this.refreshTimer !== undefined) {
      clearTimeout(this.refreshTimer);
    }
    this.refreshTimer = setTimeout(() => {
      this.refreshTimer = undefined;
      void this.refresh();
    }, delay);
  }

  private async refresh(): Promise<void> {
    const generation = ++this.refreshGeneration;
    const next = await this.resolveRegistration();
    if (generation !== this.refreshGeneration || this.disposed) {
      return;
    }

    this.reportWarning(next?.warning);
    const unchanged = next?.fingerprint === this.registration?.fingerprint;
    this.registration = next;

    if (next === undefined) {
      await this.stopHost();
      return;
    }
    if (unchanged && this.host !== undefined) {
      return;
    }
    if (!unchanged) {
      await this.stopHost();
    }
    if (vscode.window.state.focused && this.host === undefined) {
      await this.startHost(next);
    }
  }

  private async resolveRegistration(): Promise<Registration | undefined> {
    const workspaceFileName = vscode.workspace.workspaceFile?.scheme === "file"
      ? path.basename(vscode.workspace.workspaceFile.fsPath)
      : undefined;
    const workspaceName = logicalWorkspaceName(vscode.workspace.name, workspaceFileName);
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (workspaceName === undefined || workspaceFolders === undefined) {
      return undefined;
    }

    const inspectedSynonyms = vscode.workspace
      .getConfiguration("workspaceHalo")
      .inspect<readonly string[]>("rootSynonyms");
    const rootSynonyms = optionalWorkspaceScopedValue(inspectedSynonyms) ?? [];
    const rootName = selectRootName(
      workspaceName,
      workspaceFolders.map((folder) => folder.name),
      rootSynonyms
    );
    const root = workspaceFolders.find((folder) => folder.name === rootName);
    if (root === undefined || root.uri.scheme !== "file") {
      return undefined;
    }

    const vscodeDirectory = vscode.Uri.joinPath(root.uri, ".vscode");
    let directoryEntries: readonly [string, vscode.FileType][];
    try {
      directoryEntries = await vscode.workspace.fs.readDirectory(vscodeDirectory);
    } catch {
      return undefined;
    }

    const fileNames = directoryEntries
      .filter(([, type]) => type === vscode.FileType.File)
      .map(([name]) => name);
    const selection = selectLogo(workspaceName, fileNames);
    if (selection.exactLogo === undefined) {
      return undefined;
    }

    const logo = vscode.Uri.joinPath(vscodeDirectory, selection.exactLogo);
    const logoStat = await vscode.workspace.fs.stat(logo);
    const settings = this.resolveSettings(root);
    const warning = selection.logoFiles.length > 1
      ? `Workspace Halo found multiple logo files in ${vscodeDirectory.fsPath}: ${selection.logoFiles.join(", ")}. Keep only ${selection.exactLogo}.`
      : undefined;
    const fingerprint = JSON.stringify({
      workspaceName,
      root: root.uri.toString(),
      logo: logo.toString(),
      logoMtime: logoStat.mtime,
      logoSize: logoStat.size,
      settings,
      warning
    });

    return { workspaceName, root, logo, settings, warning, fingerprint };
  }

  private resolveSettings(root: vscode.WorkspaceFolder): HaloSettings {
    const halo = vscode.workspace.getConfiguration("workspaceHalo", root.uri);
    const peacock = vscode.workspace.getConfiguration("peacock", root.uri);
    const haloColor = workspaceScopedValue(halo.inspect<string>("color"), "#ff2d55");
    const peacockColor = optionalWorkspaceScopedValue(peacock.inspect<string>("color"));

    return {
      color: resolveSharedColor(peacockColor, haloColor, "#ff2d55"),
      borderWidth: workspaceScopedValue(halo.inspect<number>("borderWidth"), 12),
      borderMotif: workspaceScopedValue(
        halo.inspect<HaloSettings["borderMotif"]>("borderMotif"),
        "solid"
      ),
      fontFamily: workspaceScopedValue(halo.inspect<string>("fontFamily"), "Segoe UI"),
      fontWeight: workspaceScopedValue(halo.inspect<number>("fontWeight"), 700),
      textShadow: workspaceScopedValue(halo.inspect<boolean>("textShadow"), true)
    };
  }

  private reportWarning(warning: string | undefined): void {
    if (warning === undefined || warning === this.lastWarning) {
      this.lastWarning = warning;
      return;
    }
    this.lastWarning = warning;
    console.warn(warning);
    this.output.warn(warning);
  }

  private async startHost(registration: Registration): Promise<void> {
    const hostPath = this.context.asAbsolutePath(
      path.join("bin", "win32-x64", "workspace-halo-host.exe")
    );
    try {
      await vscode.workspace.fs.stat(vscode.Uri.file(hostPath));
      await vscode.workspace.fs.createDirectory(this.context.logUri);
    } catch (error) {
      this.output.error(`Native host is unavailable: ${String(error)}`);
      return;
    }

    const logPath = vscode.Uri.joinPath(this.context.logUri, "native-host.log").fsPath;
    const args = [
      "--window-mode", "child",
      "--name", registration.workspaceName,
      "--logo", registration.logo.fsPath,
      "--color", registration.settings.color,
      "--border-width", String(registration.settings.borderWidth),
      "--border-style", registration.settings.borderMotif,
      "--font", registration.settings.fontFamily,
      "--font-weight", String(registration.settings.fontWeight),
      `--shadow=${String(registration.settings.textShadow)}`,
      "--log", logPath,
      "--wait-for-vscode", "5s"
    ];
    if (registration.warning !== undefined) {
      args.push("--startup-warning", registration.warning);
    }

    this.output.info(
      `Tracking ${registration.workspaceName}: root=${registration.root.uri.fsPath}, logo=${registration.logo.fsPath}.`
    );
    this.output.info(`Native host log: ${logPath}`);
    const child = spawn(hostPath, args, {
      cwd: registration.root.uri.fsPath,
      stdio: ["ignore", "pipe", "pipe"],
      windowsHide: true
    });
    this.host = child;
    this.output.info(`Native host started (pid=${String(child.pid)}).`);
    this.pipeHostOutput(child);
    child.once("error", (error) => {
      this.output.error(`Native host failed to start: ${String(error)}`);
    });
    child.once("exit", (code, signal) => {
      if (this.host !== child) {
        return;
      }
      this.host = undefined;
      this.output.info(`Native host exited (code=${String(code)}, signal=${String(signal)}).`);
      if (!this.stopping && !this.disposed && vscode.window.state.focused) {
        this.scheduleRefresh(1000);
      }
    });
  }

  private pipeHostOutput(child: ChildProcess): void {
    for (const stream of [child.stdout, child.stderr]) {
      if (stream === null) {
        continue;
      }
      stream.setEncoding("utf8");
      let pending = "";
      stream.on("data", (chunk: string) => {
        pending += chunk;
        const lines = pending.split(/\r?\n/);
        pending = lines.pop() ?? "";
        for (const line of lines) {
          if (line.length > 0) {
            this.output.appendLine(line);
          }
        }
      });
      stream.on("end", () => {
        if (pending.length > 0) {
          this.output.appendLine(pending);
        }
      });
    }
  }

  private async stopHost(): Promise<void> {
    const child = this.host;
    if (child === undefined) {
      return;
    }
    this.host = undefined;
    this.stopping = true;
    if (child.exitCode === null && child.signalCode === null) {
      child.kill();
      await new Promise<void>((resolve) => {
        const timer = setTimeout(resolve, 1500);
        child.once("exit", () => {
          clearTimeout(timer);
          resolve();
        });
      });
    }
    this.stopping = false;
  }
}

let controller: WorkspaceHaloController | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  controller = new WorkspaceHaloController(context);
  context.subscriptions.push(controller);
  await controller.start();
}

export async function deactivate(): Promise<void> {
  await controller?.shutdown();
  controller = undefined;
}
