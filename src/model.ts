export interface InspectedSetting<T> {
  readonly workspaceFolderValue?: T;
  readonly workspaceValue?: T;
}

export interface LogoSelection {
  readonly exactLogo?: string;
  readonly logoFiles: readonly string[];
}

const HEX_COLOR = /^#[0-9a-fA-F]{6}$/;
const WORKSPACE_FILE_SUFFIX = ".code-workspace";

export function savedWorkspaceName(
  workspaceFileName: string | undefined
): string | undefined {
  if (!workspaceFileName?.toLowerCase().endsWith(WORKSPACE_FILE_SUFFIX)) {
    return undefined;
  }
  const name = workspaceFileName.slice(0, -WORKSPACE_FILE_SUFFIX.length);
  return name === "" ? undefined : name;
}

export function workspaceScopedValue<T>(
  inspected: InspectedSetting<T> | undefined,
  fallback: T
): T {
  return inspected?.workspaceFolderValue
    ?? inspected?.workspaceValue
    ?? fallback;
}

export function optionalWorkspaceScopedValue<T>(
  inspected: InspectedSetting<T> | undefined
): T | undefined {
  return inspected?.workspaceFolderValue ?? inspected?.workspaceValue;
}

export function selectRootName(
  workspaceName: string,
  folderNames: readonly string[],
  rootSynonyms: readonly string[]
): string | undefined {
  const exact = folderNames.find((name) => name === workspaceName);
  if (exact !== undefined) {
    return exact;
  }
  return folderNames.find((name) => rootSynonyms.includes(name));
}

export function selectLogo(
  workspaceName: string,
  directoryEntries: readonly string[]
): LogoSelection {
  const logoFiles = directoryEntries
    .filter((name) => name.endsWith(".logo.png"))
    .sort((left, right) => left.localeCompare(right));
  const expected = `${workspaceName}.logo.png`;
  return {
    exactLogo: logoFiles.find((name) => name === expected),
    logoFiles
  };
}

export function isHexColor(value: string): boolean {
  return HEX_COLOR.test(value);
}

export function resolveSharedColor(
  peacockColor: string | undefined,
  haloColor: string | undefined,
  fallback: string
): string {
  if (peacockColor !== undefined && HEX_COLOR.test(peacockColor)) {
    return peacockColor;
  }
  if (haloColor !== undefined && HEX_COLOR.test(haloColor)) {
    return haloColor;
  }
  return fallback;
}

// randomHaloColor draws a random hue at fixed saturation and value, so an
// assigned color is always vivid enough to identify a window while the pill
// contrast machinery keeps the name readable over it.
export function randomHaloColor(random: () => number): string {
  const sextant = random() * 6;
  const value = 0.92;
  const chroma = value * 0.85;
  const secondary = chroma * (1 - Math.abs((sextant % 2) - 1));
  const base = value - chroma;
  const index = Math.floor(sextant) % 6;
  const [red, green, blue]: readonly [number, number, number] =
    index === 0 ? [chroma, secondary, 0]
    : index === 1 ? [secondary, chroma, 0]
    : index === 2 ? [0, chroma, secondary]
    : index === 3 ? [0, secondary, chroma]
    : index === 4 ? [secondary, 0, chroma]
    : [chroma, 0, secondary];
  const channel = (share: number): string =>
    Math.round((share + base) * 255).toString(16).padStart(2, "0");
  return `#${channel(red)}${channel(green)}${channel(blue)}`;
}
