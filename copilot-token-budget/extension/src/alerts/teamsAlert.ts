// src/alerts/teamsAlert.ts — fires the Phase 3 Go alert binary from within the
// VS Code extension's refresh loop.
//
// Design constraints (ADR-006):
//   - Webhook URL is injected via process.env ONLY — never a CLI argument (ps aux visible).
//   - Binary absence is surfaced once via a VS Code info message (opt-in UX).
//   - Errors in the alert path never block or crash the refresh loop.

import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';

/** Sentinel key stored in globalState to suppress repeat "binary not found" messages. */
const BINARY_NOT_FOUND_SHOWN_KEY = 'copilotBudget.binaryNotFoundShown';

/** Subprocess hard timeout in milliseconds. */
const SUBPROCESS_TIMEOUT_MS = 15_000;

/**
 * fireAlertIfNeeded spawns the copilot-alert binary when a Teams webhook URL is
 * configured. Returns immediately without throwing — all errors are surfaced as
 * VS Code notifications or console.error entries.
 *
 * Call from the refresh loop with:
 *   fireAlertIfNeeded(context).catch(err => console.error('Teams alert error:', err))
 */
export async function fireAlertIfNeeded(context: vscode.ExtensionContext): Promise<void> {
  const cfg = vscode.workspace.getConfiguration('copilotBudget');
  const webhookUrl: string = cfg.get<string>('teamsWebhookUrl') ?? '';

  // Feature is opt-in — if no webhook URL is set, silently skip.
  if (webhookUrl === '') {
    return;
  }

  const binaryPath = await resolveBinaryPath(cfg);
  if (binaryPath === null) {
    await showBinaryNotFoundOnce(context);
    return;
  }

  await spawnAlertBinary(binaryPath, webhookUrl);
}

// ── Binary resolution ────────────────────────────────────────────────────────

/**
 * Resolves the copilot-alert binary path using the priority order:
 *   1. copilotBudget.alertBinaryPath setting (explicit override)
 *   2. ~/bin/copilot-alert (or copilot-alert.exe on Windows)
 * Returns null if neither location contains an accessible file.
 * Uses async fs.promises.access to avoid blocking the extension host.
 */
async function resolveBinaryPath(cfg: vscode.WorkspaceConfiguration): Promise<string | null> {
  const explicit: string = cfg.get<string>('alertBinaryPath') ?? '';
  if (explicit !== '') {
    return (await isAccessible(explicit)) ? explicit : null;
  }

  const binaryName =
    process.platform === 'win32' ? 'copilot-alert.exe' : 'copilot-alert';
  const defaultPath = path.join(os.homedir(), 'bin', binaryName);
  return (await isAccessible(defaultPath)) ? defaultPath : null;
}

/** Returns true if the path exists and is accessible (non-blocking). */
async function isAccessible(filePath: string): Promise<boolean> {
  try {
    await fs.promises.access(filePath, fs.constants.F_OK);
    return true;
  } catch {
    return false;
  }
}

// ── One-time "binary not found" notification ─────────────────────────────────

async function showBinaryNotFoundOnce(context: vscode.ExtensionContext): Promise<void> {
  const alreadyShown = context.globalState.get<boolean>(BINARY_NOT_FOUND_SHOWN_KEY, false);
  if (alreadyShown) {
    return;
  }
  await context.globalState.update(BINARY_NOT_FOUND_SHOWN_KEY, true);
  void vscode.window.showInformationMessage(
    'Teams alerts: copilot-alert binary not found. See README.'
  );
}

// ── Subprocess ───────────────────────────────────────────────────────────────

/**
 * Spawns the alert binary with the webhook URL injected via environment variable.
 * Enforces a 15-second hard timeout and surfaces stderr as a VS Code warning.
 *
 * The webhook URL is NEVER passed as a CLI argument — it is set in the child's
 * environment only, preventing exposure in `ps aux`.
 */
function spawnAlertBinary(binaryPath: string, webhookUrl: string): Promise<void> {
  return new Promise((resolve) => {
    const env = { ...process.env, COPILOT_BUDGET_TEAMS_WEBHOOK: webhookUrl };

    const child = cp.execFile(
      binaryPath,
      [],
      { env, timeout: SUBPROCESS_TIMEOUT_MS },
      (error, _stdout, stderr) => {
        if (stderr !== '') {
          // Surface stderr as a VS Code warning — never swallow silently.
          void vscode.window.showWarningMessage(
            `Copilot Budget alert: ${stderr.trim()}`
          );
        }

        if (error && error.killed) {
          console.error(`copilot-budget: alert binary exceeded ${SUBPROCESS_TIMEOUT_MS / 1000}s timeout and was killed`);
        } else if (error && (error as NodeJS.ErrnoException).code !== undefined) {
          // Spawn error (e.g. permission denied) — log but don't surface to user
          // as a disruptive notification.
          console.error(`copilot-budget: alert binary error: ${error.message}`);
        }
        // Exit codes 0 (no alert) and 1 (alert fired) are both success paths.
        resolve();
      }
    );

    // Defensive: kill if timeout fires before execFile's own timeout handler.
    const killTimer = setTimeout(() => {
      child.kill();
    }, SUBPROCESS_TIMEOUT_MS + 1000);

    child.on('close', () => clearTimeout(killTimer));
  });
}
