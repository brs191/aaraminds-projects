// instructions/analyzer.ts — workspace instruction file scanner.
// TypeScript port of phase-1/session-manager/internal/instructions/analyzer.go.
// Uses only Node.js built-ins (fs, path) — zero npm runtime deps (ADR-003).

import * as fs from 'fs';
import * as path from 'path';
import { InstructionFile } from '../types';

// scanWorkspace scans workspacePath for Copilot instruction *.md files at two levels:
//   1. <workspacePath>/.github/instructions/*.md  → scope "workspace-root"
//   2. <workspacePath>/<subdir>/.github/instructions/*.md → scope "project-scoped"
//
// Duplicate physical files (symlinked repos) are deduplicated via fs.realpathSync.
// Results are sorted by estimatedTokens descending.
export async function scanWorkspace(workspacePath: string): Promise<InstructionFile[]> {
  const absRoot = path.resolve(workspacePath);
  const seen = new Set<string>(); // keyed by real (symlink-resolved) path
  const results: InstructionFile[] = [];

  // Level 1: workspace-root instruction files.
  const wsInstructionsDir = path.join(absRoot, '.github', 'instructions');
  await scanDir(wsInstructionsDir, 'workspace-root', '', seen, results);

  // Level 2: one level of subdirectories — each may have its own .github/instructions/.
  let entries: fs.Dirent[];
  try {
    entries = await fs.promises.readdir(absRoot, { withFileTypes: true });
  } catch {
    return results.sort(byTokensDesc); // root unreadable at subdir level — return what we have
  }

  for (const entry of entries) {
    if (!entry.isDirectory()) {
      continue;
    }
    const subdir = path.join(absRoot, entry.name);
    const projInstructionsDir = path.join(subdir, '.github', 'instructions');
    await scanDir(projInstructionsDir, 'project-scoped', entry.name, seen, results);
  }

  return results.sort(byTokensDesc);
}

// severity returns a lowercase severity label for the VS Code extension (no emoji).
// Matches Go Severity() thresholds exactly.
export function severity(tokens: number): 'high' | 'medium' | 'low' {
  if (tokens >= 2000) {
    return 'high';
  }
  if (tokens >= 500) {
    return 'medium';
  }
  return 'low';
}

// savingsRecommendation returns a human-readable recommendation for a token count.
// Matches Go SavingsRecommendation() thresholds and messages exactly.
export function savingsRecommendation(tokens: number): string {
  if (tokens >= 5000) {
    return 'CRITICAL — split or remove; >5K tokens loaded every message';
  }
  if (tokens >= 2000) {
    return 'HIGH — trim to <2K tokens';
  }
  if (tokens >= 500) {
    return 'MEDIUM — review for unnecessary content';
  }
  return 'OK';
}

// scanDir reads *.md files from dir and appends non-duplicate InstructionFiles to results.
async function scanDir(
  dir: string,
  scope: string,
  project: string,
  seen: Set<string>,
  results: InstructionFile[]
): Promise<void> {
  let entries: fs.Dirent[];
  try {
    entries = await fs.promises.readdir(dir, { withFileTypes: true });
  } catch {
    return; // directory absent or unreadable — silently skip (mirrors Go behaviour)
  }

  for (const entry of entries) {
    if (entry.isDirectory()) {
      continue;
    }
    if (path.extname(entry.name) !== '.md') {
      continue;
    }

    const absPath = path.join(dir, entry.name);

    // Resolve symlinks for deduplication — async realpath avoids blocking the extension host.
    let realPath: string;
    try {
      realPath = await fs.promises.realpath(absPath);
    } catch (err) {
      console.error(`copilot-budget: cannot resolve symlink ${absPath}: ${err}`);
      continue;
    }

    if (seen.has(realPath)) {
      continue;
    }
    seen.add(realPath);

    let content: string;
    try {
      content = await fs.promises.readFile(absPath, 'utf8');
    } catch (err) {
      console.error(`copilot-budget: cannot read ${absPath}: ${err}`);
      continue;
    }

    results.push({
      path: absPath,
      scope,
      project,
      // Go estimates from UTF-8 BYTE length (len(content) / 4). content.length here
      // counts UTF-16 code units, which diverges on non-ASCII. Use Buffer.byteLength
      // to match Go's byte count exactly.
      estimatedTokens: Math.floor(Buffer.byteLength(content, 'utf8') / 4),
    });
  }
}

// byTokensDesc is a sort comparator for descending estimatedTokens order.
function byTokensDesc(a: InstructionFile, b: InstructionFile): number {
  return b.estimatedTokens - a.estimatedTokens;
}
