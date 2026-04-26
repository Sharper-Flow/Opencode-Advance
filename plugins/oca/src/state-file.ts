import * as fs from "fs";
import * as path from "path";
import * as os from "os";

export function xdgStateHome(...subdirs: string[]): string {
  const base = process.env.XDG_STATE_HOME || path.join(os.homedir(), ".local", "state");
  return path.join(base, ...subdirs);
}

export function atomicWriteJSON(filePath: string, data: unknown): void {
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
  const tmpFile = `${filePath}.${process.pid}.tmp`;
  fs.writeFileSync(tmpFile, JSON.stringify(data, null, 2), { encoding: "utf8" });
  fs.renameSync(tmpFile, filePath);
}

export function readJSON(filePath: string): any {
  try {
    const raw = fs.readFileSync(filePath, { encoding: "utf8" });
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

export function deleteStateFile(filePath: string): void {
  try {
    fs.unlinkSync(filePath);
  } catch {
    // ignore missing file
  }
}
