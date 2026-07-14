#!/usr/bin/env node
// PreToolUse hook: blocks Edit/Write on a code-migration file that is already
// committed to origin/main. Code migrations run once (see
// apps/api/core/database/db.go GlobalDBMap/ExecuteAll) and are tracked by ID
// in the DB — editing one that already ran anywhere is a silent no-op there,
// causing drift between prod and fresh environments. New (not-yet-on-main)
// migration files are left untouched by this hook.
// See docs/backend/api-code-migration.md rule #1.

import { execFileSync } from "node:child_process";
import { relative, dirname } from "node:path";

const MIGRATION_PATH_RE = /apps\/api\/migrations\/code-migrations\/.*\.go$/;

function readStdin() {
  return new Promise((resolve, reject) => {
    let data = "";
    process.stdin.setEncoding("utf-8");
    process.stdin.on("data", (chunk) => { data += chunk; });
    process.stdin.on("end", () => resolve(data));
    process.stdin.on("error", reject);
    process.stdin.resume();
  });
}

function deny(reason) {
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: reason,
    },
  }));
}

async function main() {
  let payload;
  try {
    payload = JSON.parse(await readStdin());
  } catch {
    return; // malformed input: fail open
  }

  const filePath = payload?.tool_input?.file_path;
  if (!filePath || !MIGRATION_PATH_RE.test(filePath.replace(/\\/g, "/"))) {
    return; // not a code-migration file: no opinion
  }

  let repoRoot;
  try {
    repoRoot = execFileSync("git", ["-C", dirname(filePath), "rev-parse", "--show-toplevel"], {
      encoding: "utf-8",
    }).trim();
  } catch {
    return; // not a git repo: fail open
  }

  const relPath = relative(repoRoot, filePath).split("\\").join("/");

  try {
    execFileSync("git", ["-C", repoRoot, "cat-file", "-e", `origin/main:${relPath}`], {
      stdio: "ignore",
    });
  } catch {
    return; // file does not exist on origin/main yet: allow (new/unmerged migration)
  }

  // File already exists on origin/main -> block the edit.
  deny(
    `"${relPath}" ya está commiteado en origin/main (asumido = ya desplegado/producción). ` +
    "Las code migrations corren una sola vez por ID (apps/api/core/database/db.go) — editar " +
    "una que ya corrió no la re-ejecuta donde ya se aplicó, generando drift silencioso. " +
    "Regla: docs/backend/api-code-migration.md #1 (IDs inmutables). " +
    "Creá un archivo nuevo con un DB<YYYYMMDD>_<N> nuevo en vez de tocar este."
  );
}

main();
