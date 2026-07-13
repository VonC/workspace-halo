export interface InspectedSetting<T> {
  readonly workspaceFolderValue?: T;
  readonly workspaceValue?: T;
}

export interface LogoSelection {
  readonly exactLogo?: string;
  readonly logoFiles: readonly string[];
}

const HEX_COLOR = /^#[0-9a-fA-F]{6}$/;

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

export function resolveSharedColor(
  peacockColor: string | undefined,
  haloColor: string,
  fallback: string
): string {
  if (peacockColor !== undefined && HEX_COLOR.test(peacockColor)) {
    return peacockColor;
  }
  if (HEX_COLOR.test(haloColor)) {
    return haloColor;
  }
  return fallback;
}

