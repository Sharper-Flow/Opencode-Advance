export function parseSocketFromTmux(tmuxEnv: string | undefined): string {
  if (!tmuxEnv) return "oca";
  // $TMUX format: /tmp/tmux-<uid>/<socket>,<pid>
  const parts = tmuxEnv.split(",");
  if (parts.length < 2) return "oca";
  const socketPath = parts[0];
  const basename = socketPath.split("/").pop();
  return basename || "oca";
}

export function sanitizePaneId(paneId: string): string {
  return paneId.replace(/^%/, "");
}
