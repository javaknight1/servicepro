#!/usr/bin/env node
/**
 * Dead Code Detection Script
 *
 * Uses ts-prune to detect unused exports and compares against a baseline.
 * Fails if new unused exports are introduced that aren't in the baseline.
 *
 * Usage:
 *   node scripts/check-dead-code.js           # Check for new dead code
 *   node scripts/check-dead-code.js --update  # Update baseline with current state
 *   node scripts/check-dead-code.js --list    # List all unused exports
 */

import { execSync } from 'child_process';
import { readFileSync, writeFileSync, existsSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const BASELINE_PATH = resolve(__dirname, '../.dead-code-baseline.json');

// Patterns to ignore (false positives)
const IGNORE_PATTERNS = [
  // Barrel files / re-exports - ts-prune doesn't track usage through re-exports
  /\/index\.ts:/,
  /\/index\.tsx:/,

  // Generated files
  /api\.generated\.ts/,

  // Test files
  /\.test\./,
  /\.spec\./,
  /\/__mocks__\//,
  /\/test\//,

  // Entry points (used by bundler, not directly imported)
  /^src\/App\.tsx:\d+ - default$/,
  /^src\/main\.tsx:/,

  // Service worker (loaded at runtime)
  /service-worker\.ts/,

  // Items marked "used in module" are re-exported types, not dead code
  /\(used in module\)/,
];

// Categories of exports that are intentionally unused but kept for future use
// These should be documented in TODO.md or have clear comments
const KNOWN_FUTURE_USE = [
  // Analytics context - will be replaced with PostHog (T025)
  'src/contexts/AnalyticsContext.tsx',
  // KPI store - will be replaced with Metabase (T026)
  'src/store/kpiStore.tsx',
  // Roles hooks - will be used with T035 (Create Technician role)
  'src/hooks/useRoles.ts',
];

function runTsPrune() {
  try {
    const output = execSync('npx ts-prune 2>&1', {
      cwd: resolve(__dirname, '..'),
      encoding: 'utf-8',
      maxBuffer: 10 * 1024 * 1024, // 10MB buffer
    });
    return output
      .split('\n')
      .filter((line) => line.trim())
      .filter((line) => !line.includes('error TS'));
  } catch (error) {
    // ts-prune exits with code 0 even on success, but execSync may throw
    if (error.stdout) {
      return error.stdout
        .split('\n')
        .filter((line) => line.trim())
        .filter((line) => !line.includes('error TS'));
    }
    throw error;
  }
}

function filterResults(results) {
  return results.filter((line) => {
    // Check against ignore patterns
    for (const pattern of IGNORE_PATTERNS) {
      if (pattern.test(line)) {
        return false;
      }
    }
    return true;
  });
}

function parseExport(line) {
  // Format: "src/path/file.ts:123 - exportName"
  const match = line.match(/^(.+):(\d+) - (.+)$/);
  if (match) {
    return {
      file: match[1],
      line: parseInt(match[2], 10),
      name: match[3],
      raw: line,
    };
  }
  return null;
}

function loadBaseline() {
  if (!existsSync(BASELINE_PATH)) {
    return { exports: [], lastUpdated: null, version: 1 };
  }
  try {
    return JSON.parse(readFileSync(BASELINE_PATH, 'utf-8'));
  } catch {
    return { exports: [], lastUpdated: null, version: 1 };
  }
}

function saveBaseline(exports) {
  const baseline = {
    version: 1,
    lastUpdated: new Date().toISOString(),
    description:
      'Known unused exports baseline. Run `npm run dead-code:update` to update.',
    exports: exports.sort(),
  };
  writeFileSync(BASELINE_PATH, JSON.stringify(baseline, null, 2) + '\n');
  console.log(`Baseline updated with ${exports.length} entries.`);
}

function categorizeExport(exportInfo) {
  for (const pattern of KNOWN_FUTURE_USE) {
    if (exportInfo.file.includes(pattern)) {
      return 'future-use';
    }
  }
  return 'unknown';
}

function main() {
  const args = process.argv.slice(2);
  const updateMode = args.includes('--update');
  const listMode = args.includes('--list');

  console.log('Running ts-prune...');
  const rawResults = runTsPrune();
  const filteredResults = filterResults(rawResults);
  const parsedExports = filteredResults.map(parseExport).filter(Boolean);

  if (listMode) {
    console.log('\nAll unused exports (after filtering):');
    console.log('=====================================');
    for (const exp of parsedExports) {
      const category = categorizeExport(exp);
      const marker = category === 'future-use' ? ' [FUTURE]' : '';
      console.log(`  ${exp.raw}${marker}`);
    }
    console.log(`\nTotal: ${parsedExports.length} unused exports`);
    return;
  }

  const currentExports = new Set(filteredResults);
  const baseline = loadBaseline();
  const baselineExports = new Set(baseline.exports);

  if (updateMode) {
    saveBaseline([...currentExports]);
    return;
  }

  // Find new unused exports (not in baseline)
  const newUnused = [...currentExports].filter(
    (exp) => !baselineExports.has(exp)
  );

  // Find removed exports (in baseline but no longer unused - these are good!)
  const removed = [...baselineExports].filter(
    (exp) => !currentExports.has(exp)
  );

  if (removed.length > 0) {
    console.log('\nExports no longer unused (good!):');
    for (const exp of removed) {
      console.log(`  - ${exp}`);
    }
  }

  if (newUnused.length > 0) {
    console.log('\nNew unused exports detected:');
    console.log('============================');
    for (const exp of newUnused) {
      const parsed = parseExport(exp);
      if (parsed) {
        const category = categorizeExport(parsed);
        if (category === 'future-use') {
          console.log(`  ${exp} [Known future use]`);
        } else {
          console.log(`  ${exp}`);
        }
      } else {
        console.log(`  ${exp}`);
      }
    }
    console.log(
      '\nTo fix: Either use these exports, remove them, or update the baseline with:'
    );
    console.log('  npm run dead-code:update');
    console.log(
      '\nNote: Only update baseline if these are intentionally kept for future use.'
    );
    process.exit(1);
  }

  console.log('\nNo new dead code detected.');
  console.log(`Baseline: ${baselineExports.size} known unused exports`);
  console.log(`Current:  ${currentExports.size} unused exports`);
  if (removed.length > 0) {
    console.log(`Removed:  ${removed.length} exports are now being used`);
    console.log('\nConsider updating baseline to remove resolved items:');
    console.log('  npm run dead-code:update');
  }
}

main();
