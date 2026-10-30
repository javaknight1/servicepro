#!/usr/bin/env node
/**
 * Fails the build when built JS chunks import each other in a cycle.
 *
 * Rolldown's CommonJS wrappers can't evaluate a static-import cycle between
 * chunks: one side runs before the other is initialised and the app dies at
 * startup with "r is not a function" — a blank page that tsc, lint and the
 * unit tests (dev mode) all miss. That shipped in v0.9.4–v0.10.4 when
 * react-router 8 pulled a transform helper into `vendor`.
 *
 * Usage: node scripts/check-chunk-cycles.cjs [distDir]   (default: dist)
 */
const fs = require('fs');
const path = require('path');

const dist = path.resolve(process.argv[2] || 'dist');
const files = [];
(function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full);
    else if (entry.name.endsWith('.js')) files.push(full);
  }
})(dist);

// Static imports only: `import ... from "./x.js"` / `import "./x.js"`.
// Dynamic import() is lazy and can't deadlock module evaluation.
const STATIC_IMPORT =
  /^\s*import\s+(?:[^'"]*?\s+from\s+)?["'](\.{1,2}\/[^"']+\.js)["']/gm;
const graph = new Map();
for (const file of files) {
  const src = fs.readFileSync(file, 'utf8');
  const deps = new Set();
  for (const m of src.matchAll(STATIC_IMPORT)) {
    deps.add(path.resolve(path.dirname(file), m[1]));
  }
  graph.set(file, deps);
}

const WHITE = 0,
  GREY = 1,
  BLACK = 2;
const color = new Map(files.map((f) => [f, WHITE]));
const cycles = [];
function visit(node, stack) {
  color.set(node, GREY);
  stack.push(node);
  for (const dep of graph.get(node) || []) {
    if (!graph.has(dep)) continue;
    if (color.get(dep) === GREY) {
      cycles.push(stack.slice(stack.indexOf(dep)).concat(dep));
    } else if (color.get(dep) === WHITE) {
      visit(dep, stack);
    }
  }
  stack.pop();
  color.set(node, BLACK);
}
for (const f of files) if (color.get(f) === WHITE) visit(f, []);

if (cycles.length) {
  console.error(`✖ ${cycles.length} chunk import cycle(s) in ${dist}:`);
  for (const c of cycles) {
    console.error('  ' + c.map((f) => path.basename(f)).join(' → '));
  }
  console.error('Fix the codeSplitting groups in config/optimization.ts.');
  process.exit(1);
}
console.log(`✓ No chunk import cycles (${files.length} chunks)`);
