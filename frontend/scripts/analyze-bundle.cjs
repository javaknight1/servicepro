#!/usr/bin/env node

/**
 * =============================================================================
 * Bundle Analysis Script
 * =============================================================================
 * Analyzes build output and generates reports
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const zlib = require('zlib');

// =============================================================================
// Configuration
// =============================================================================

const config = {
  distDir: path.resolve(__dirname, '../dist'),
  statsFile: path.resolve(__dirname, '../dist/bundle-stats.json'),
  reportFile: path.resolve(__dirname, '../dist/bundle-report.json'),

  // Generic per-chunk thresholds (in KB, gzipped) for chunks without explicit budgets
  thresholds: {
    warning: 250,
    error: 500,
  },

  // Performance budgets
  budgets: {
    totalSize: 500, // Total JS gzipped warning threshold (KB)
    totalHardFail: 750, // Total JS gzipped hard failure threshold (KB)
    initialJS: 400,
    initialCSS: 100,
    largestChunk: 500,
  },

  // Per-chunk budgets (KB, gzipped) — mirrors performanceBudgets.maxBundleSizes
  // in frontend/config/optimization.ts (source of truth)
  chunkBudgets: {
    'vendor-react': 60,
    'vendor-ui': 25,
    'vendor-forms': 30,
    'vendor-data': 25,
    'vendor-charts': 70,
    'vendor-date': 10,
    'vendor-stripe': 10,
    'vendor-calendar': 50,
    'vendor-table': 10,
    vendor: 60,
    'page-dashboard': 10,
    'page-settings': 20,
    'page-auth': 20,
  },
};

// =============================================================================
// Utilities
// =============================================================================

/**
 * Get file size in different formats
 */
function getFileSizes(filePath) {
  const content = fs.readFileSync(filePath);
  const gzipped = zlib.gzipSync(content);
  const brotli = zlib.brotliCompressSync(content);

  return {
    raw: content.length,
    gzip: gzipped.length,
    brotli: brotli.length,
  };
}

/**
 * Format bytes to human readable
 */
function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

/**
 * Get all files recursively
 */
function getFiles(dir, files = []) {
  const items = fs.readdirSync(dir, { withFileTypes: true });

  for (const item of items) {
    const fullPath = path.join(dir, item.name);

    if (item.isDirectory()) {
      getFiles(fullPath, files);
    } else {
      files.push(fullPath);
    }
  }

  return files;
}

/**
 * Categorize file by type
 */
function categorizeFile(filePath) {
  const ext = path.extname(filePath).toLowerCase();

  if (['.js', '.mjs'].includes(ext)) return 'javascript';
  if (ext === '.css') return 'css';
  if (
    [
      '.png',
      '.jpg',
      '.jpeg',
      '.gif',
      '.svg',
      '.webp',
      '.avif',
      '.ico',
    ].includes(ext)
  ) {
    return 'images';
  }
  if (['.woff', '.woff2', '.ttf', '.eot', '.otf'].includes(ext)) return 'fonts';
  if (ext === '.html') return 'html';
  if (['.json', '.xml', '.txt'].includes(ext)) return 'data';
  if (['.map'].includes(ext)) return 'sourcemaps';

  return 'other';
}

/**
 * Extract chunk name from build filename by stripping the hash suffix.
 * e.g. "vendor-react-B6rzUpVc.js" → "vendor-react"
 *      "index-Da3xK9pQ.js" → "index"
 */
function extractChunkName(filename) {
  const base = path.basename(filename, path.extname(filename));
  const match = base.match(/^(.+)-[A-Za-z0-9]+$/);
  return match ? match[1] : base;
}

/**
 * Look up the per-chunk budget (KB) for a given chunk name.
 * Returns null if no explicit budget is defined.
 */
function getChunkBudget(chunkName) {
  if (config.chunkBudgets.hasOwnProperty(chunkName)) {
    return config.chunkBudgets[chunkName];
  }
  return null;
}

// =============================================================================
// Analysis Functions
// =============================================================================

/**
 * Analyze all build files
 */
function analyzeFiles() {
  if (!fs.existsSync(config.distDir)) {
    console.error('❌ Dist directory not found. Run build first.');
    process.exit(1);
  }

  const files = getFiles(config.distDir);
  const analysis = {
    timestamp: new Date().toISOString(),
    files: [],
    summary: {
      byCategory: {},
      total: { raw: 0, gzip: 0, brotli: 0, count: 0 },
    },
    chunks: [],
    warnings: [],
    errors: [],
  };

  console.log('\n📦 Analyzing bundle...\n');

  for (const filePath of files) {
    const relativePath = path.relative(config.distDir, filePath);
    const category = categorizeFile(filePath);
    const sizes = getFileSizes(filePath);

    // Skip source maps from size calculations
    if (category === 'sourcemaps') {
      continue;
    }

    const fileInfo = {
      path: relativePath,
      category,
      sizes,
    };

    analysis.files.push(fileInfo);

    // Update summary
    if (!analysis.summary.byCategory[category]) {
      analysis.summary.byCategory[category] = {
        raw: 0,
        gzip: 0,
        brotli: 0,
        count: 0,
      };
    }

    analysis.summary.byCategory[category].raw += sizes.raw;
    analysis.summary.byCategory[category].gzip += sizes.gzip;
    analysis.summary.byCategory[category].brotli += sizes.brotli;
    analysis.summary.byCategory[category].count++;

    analysis.summary.total.raw += sizes.raw;
    analysis.summary.total.gzip += sizes.gzip;
    analysis.summary.total.brotli += sizes.brotli;
    analysis.summary.total.count++;

    // Track JS chunks
    if (category === 'javascript') {
      const fileName = path.basename(relativePath);
      const chunkName = extractChunkName(fileName);
      const budget = getChunkBudget(chunkName);

      analysis.chunks.push({
        name: fileName,
        chunkName,
        path: relativePath,
        sizes,
        budget,
      });

      const sizeKB = sizes.gzip / 1024;

      // Per-chunk budget check (explicit budget from chunkBudgets map)
      if (budget !== null) {
        if (sizeKB > budget) {
          analysis.errors.push({
            type: 'chunk_budget',
            file: relativePath,
            message: `Chunk "${chunkName}" exceeds budget (${formatBytes(
              sizes.gzip
            )} > ${budget} KB)`,
          });
        }
      } else {
        // Generic threshold for chunks without explicit budgets
        if (sizeKB > config.thresholds.error) {
          analysis.errors.push({
            type: 'chunk_size',
            file: relativePath,
            message: `Chunk "${chunkName}" exceeds error threshold (${formatBytes(
              sizes.gzip
            )} > ${config.thresholds.error} KB)`,
          });
        } else if (sizeKB > config.thresholds.warning) {
          analysis.warnings.push({
            type: 'chunk_size',
            file: relativePath,
            message: `Chunk "${chunkName}" exceeds warning threshold (${formatBytes(
              sizes.gzip
            )} > ${config.thresholds.warning} KB)`,
          });
        }
      }
    }
  }

  // Sort chunks by size
  analysis.chunks.sort((a, b) => b.sizes.gzip - a.sizes.gzip);

  // Check budgets
  checkBudgets(analysis);

  return analysis;
}

/**
 * Check performance budgets
 */
function checkBudgets(analysis) {
  const { budgets } = config;
  const js = analysis.summary.byCategory.javascript || { gzip: 0 };
  const css = analysis.summary.byCategory.css || { gzip: 0 };

  // Total JS size budgets (warning at totalSize, hard fail at totalHardFail)
  const jsTotal = analysis.summary.byCategory.javascript || { gzip: 0 };
  const jsTotalKB = jsTotal.gzip / 1024;

  if (jsTotalKB > budgets.totalHardFail) {
    analysis.errors.push({
      type: 'budget',
      message: `Total JS gzipped exceeds hard limit (${formatBytes(
        jsTotal.gzip
      )} > ${budgets.totalHardFail} KB)`,
    });
  } else if (jsTotalKB > budgets.totalSize) {
    analysis.warnings.push({
      type: 'budget',
      message: `Total JS gzipped exceeds warning threshold (${formatBytes(
        jsTotal.gzip
      )} > ${budgets.totalSize} KB)`,
    });
  }

  // Total bundle size (all assets)
  const totalKB = analysis.summary.total.gzip / 1024;
  if (totalKB > budgets.totalHardFail * 2) {
    analysis.errors.push({
      type: 'budget',
      message: `Total bundle size is very large (${formatBytes(
        analysis.summary.total.gzip
      )})`,
    });
  }

  // Initial JS budget
  const jsKB = js.gzip / 1024;
  if (jsKB > budgets.initialJS) {
    analysis.warnings.push({
      type: 'budget',
      message: `JavaScript size exceeds budget (${formatBytes(js.gzip)} > ${
        budgets.initialJS
      }KB)`,
    });
  }

  // Initial CSS budget
  const cssKB = css.gzip / 1024;
  if (cssKB > budgets.initialCSS) {
    analysis.warnings.push({
      type: 'budget',
      message: `CSS size exceeds budget (${formatBytes(css.gzip)} > ${
        budgets.initialCSS
      }KB)`,
    });
  }

  // Largest chunk budget
  if (analysis.chunks.length > 0) {
    const largest = analysis.chunks[0];
    const largestKB = largest.sizes.gzip / 1024;

    if (largestKB > budgets.largestChunk) {
      analysis.warnings.push({
        type: 'budget',
        message: `Largest chunk exceeds budget (${largest.name}: ${formatBytes(
          largest.sizes.gzip
        )} > ${budgets.largestChunk}KB)`,
      });
    }
  }
}

/**
 * Analyze dependencies
 */
function analyzeDependencies() {
  const packagePath = path.resolve(__dirname, '../package.json');
  const lockPath = path.resolve(__dirname, '../package-lock.json');

  if (!fs.existsSync(packagePath)) {
    return null;
  }

  const pkg = JSON.parse(fs.readFileSync(packagePath, 'utf-8'));
  const deps = {
    dependencies: Object.keys(pkg.dependencies || {}),
    devDependencies: Object.keys(pkg.devDependencies || {}),
    total: 0,
  };

  deps.total = deps.dependencies.length + deps.devDependencies.length;

  // Check for potentially large packages
  const largePkgs = [
    'moment',
    'lodash',
    'rxjs',
    '@material-ui/core',
    '@mui/material',
    'antd',
    'bootstrap',
    'jquery',
    'angular',
    'vue',
  ];

  deps.warnings = deps.dependencies.filter((dep) =>
    largePkgs.some((large) => dep.includes(large))
  );

  return deps;
}

/**
 * Analyze tree shaking effectiveness
 */
function analyzeTreeShaking() {
  const analysis = {
    esModules: true,
    sideEffects: false,
    unusedExports: [],
  };

  // Check package.json for sideEffects field
  const packagePath = path.resolve(__dirname, '../package.json');
  if (fs.existsSync(packagePath)) {
    const pkg = JSON.parse(fs.readFileSync(packagePath, 'utf-8'));
    analysis.sideEffects = pkg.sideEffects !== false;
  }

  return analysis;
}

// =============================================================================
// Report Generation
// =============================================================================

/**
 * Print concise CI report (used with --ci flag)
 */
function printCIReport(analysis) {
  const jsChunks = analysis.chunks;
  const jsCategory = analysis.summary.byCategory.javascript || { gzip: 0 };
  const jsTotalKB = jsCategory.gzip / 1024;

  console.log('');
  console.log('Bundle Size Report');
  console.log('═══════════════════════════════════════════════════');
  console.log('Chunk                    Gzip       Budget    Status');
  console.log('───────────────────────────────────────────────────');

  for (const chunk of jsChunks) {
    const sizeKB = chunk.sizes.gzip / 1024;
    const budgetStr =
      chunk.budget !== null
        ? `${chunk.budget} KB`
        : `${config.thresholds.error} KB`;
    const over =
      chunk.budget !== null
        ? sizeKB > chunk.budget
        : sizeKB > config.thresholds.error;
    const status = over ? 'OVER' : 'OK';
    const name = chunk.chunkName.padEnd(24);
    const size = `${sizeKB.toFixed(1)} KB`.padStart(10);
    const budget = budgetStr.padStart(9);
    console.log(`${name} ${size}  ${budget}    ${status}`);
  }

  console.log('───────────────────────────────────────────────────');
  const totalSize = `${jsTotalKB.toFixed(0)} KB`.padStart(10);
  const totalBudget =
    `${config.budgets.totalSize}/${config.budgets.totalHardFail}`.padStart(9);
  const totalOver = jsTotalKB > config.budgets.totalHardFail;
  const totalWarn = jsTotalKB > config.budgets.totalSize && !totalOver;
  const totalStatus = totalOver ? 'OVER' : totalWarn ? 'WARN' : 'OK';
  console.log(
    `${'Total JS (gzipped)'.padEnd(24)} ${totalSize}  ${totalBudget}    ${totalStatus}`
  );
  console.log('');

  const hasErrors = analysis.errors.length > 0;
  console.log(`Result: ${hasErrors ? 'FAIL' : 'PASS'}`);

  if (hasErrors) {
    console.log('');
    for (const error of analysis.errors) {
      console.log(`  ERROR: ${error.message}`);
    }
  }

  if (analysis.warnings.length > 0) {
    console.log('');
    for (const warning of analysis.warnings) {
      console.log(`  WARN: ${warning.message}`);
    }
  }

  console.log('');
}

/**
 * Print analysis report to console
 */
function printReport(analysis) {
  console.log('═══════════════════════════════════════════════════════════');
  console.log('                    Bundle Analysis Report');
  console.log('═══════════════════════════════════════════════════════════\n');

  // Summary
  console.log('📊 Summary');
  console.log('───────────────────────────────────────────────────────────');
  console.log(`  Total Files:     ${analysis.summary.total.count}`);
  console.log(`  Raw Size:        ${formatBytes(analysis.summary.total.raw)}`);
  console.log(`  Gzip Size:       ${formatBytes(analysis.summary.total.gzip)}`);
  console.log(
    `  Brotli Size:     ${formatBytes(analysis.summary.total.brotli)}`
  );
  console.log('');

  // By category
  console.log('📁 By Category');
  console.log('───────────────────────────────────────────────────────────');

  for (const [category, data] of Object.entries(analysis.summary.byCategory)) {
    console.log(
      `  ${category.padEnd(15)} ${data.count
        .toString()
        .padStart(3)} files  ${formatBytes(data.gzip).padStart(10)} (gzip)`
    );
  }
  console.log('');

  // Top chunks
  console.log('📦 Largest Chunks (Gzip)');
  console.log('───────────────────────────────────────────────────────────');

  const topChunks = analysis.chunks.slice(0, 10);
  for (const chunk of topChunks) {
    const bar = '█'.repeat(Math.min(Math.ceil(chunk.sizes.gzip / 10240), 30));
    console.log(
      `  ${chunk.name.padEnd(40)} ${formatBytes(chunk.sizes.gzip).padStart(
        10
      )}  ${bar}`
    );
  }
  console.log('');

  // Warnings
  if (analysis.warnings.length > 0) {
    console.log('⚠️  Warnings');
    console.log('───────────────────────────────────────────────────────────');
    for (const warning of analysis.warnings) {
      console.log(`  • ${warning.message}`);
    }
    console.log('');
  }

  // Errors
  if (analysis.errors.length > 0) {
    console.log('❌ Errors');
    console.log('───────────────────────────────────────────────────────────');
    for (const error of analysis.errors) {
      console.log(`  • ${error.message}`);
    }
    console.log('');
  }

  // Recommendations
  console.log('💡 Recommendations');
  console.log('───────────────────────────────────────────────────────────');

  if (analysis.summary.byCategory.javascript?.gzip > 400 * 1024) {
    console.log(
      '  • Consider additional code splitting for large JavaScript bundles'
    );
  }

  if (analysis.chunks.some((c) => c.sizes.gzip > 250 * 1024)) {
    console.log(
      '  • Large chunks detected. Consider lazy loading or splitting'
    );
  }

  const images = analysis.summary.byCategory.images;
  if (images && images.raw > 500 * 1024) {
    console.log(
      '  • Large image assets detected. Consider WebP/AVIF conversion'
    );
  }

  if (analysis.warnings.length === 0 && analysis.errors.length === 0) {
    console.log('  ✓ All performance budgets are within limits!');
  }

  console.log('');
  console.log('═══════════════════════════════════════════════════════════\n');
}

/**
 * Generate markdown report for PR comments
 */
function generateMarkdownReport(analysis) {
  const jsCategory = analysis.summary.byCategory.javascript || { gzip: 0 };
  const jsTotalKB = jsCategory.gzip / 1024;
  const cssCategory = analysis.summary.byCategory.css || { gzip: 0 };
  const hasErrors = analysis.errors.length > 0;
  const hasWarnings = analysis.warnings.length > 0;

  const statusIcon = hasErrors ? '🔴' : hasWarnings ? '🟡' : '🟢';
  const statusText = hasErrors ? 'FAIL' : hasWarnings ? 'WARN' : 'PASS';

  let md = `## ${statusIcon} Bundle Size Report — ${statusText}\n\n`;

  // Summary table
  md += `| Metric | Size | Budget | Status |\n`;
  md += `| ------ | ---- | ------ | ------ |\n`;

  const totalOver = jsTotalKB > config.budgets.totalHardFail;
  const totalWarn = jsTotalKB > config.budgets.totalSize && !totalOver;
  const totalIcon = totalOver ? '🔴' : totalWarn ? '🟡' : '🟢';
  md += `| **Total JS (gzipped)** | **${jsTotalKB.toFixed(1)} KB** | ${config.budgets.totalSize} / ${config.budgets.totalHardFail} KB | ${totalIcon} |\n`;
  md += `| Total CSS (gzipped) | ${(cssCategory.gzip / 1024).toFixed(1)} KB | ${config.budgets.initialCSS} KB | ${cssCategory.gzip / 1024 > config.budgets.initialCSS ? '🟡' : '🟢'} |\n`;
  md += `| Total assets | ${formatBytes(analysis.summary.total.gzip)} | — | — |\n`;
  md += `\n`;

  // Chunk breakdown
  md += `<details>\n<summary>Chunk Breakdown (${analysis.chunks.length} chunks)</summary>\n\n`;
  md += `| Chunk | Gzip | Budget | Status |\n`;
  md += `| ----- | ---- | ------ | ------ |\n`;

  for (const chunk of analysis.chunks) {
    const sizeKB = chunk.sizes.gzip / 1024;
    const budgetStr =
      chunk.budget !== null
        ? `${chunk.budget} KB`
        : `${config.thresholds.error} KB`;
    const over =
      chunk.budget !== null
        ? sizeKB > chunk.budget
        : sizeKB > config.thresholds.error;
    const warn = chunk.budget === null && sizeKB > config.thresholds.warning;
    const icon = over ? '🔴' : warn ? '🟡' : '🟢';
    md += `| \`${chunk.chunkName}\` | ${sizeKB.toFixed(1)} KB | ${budgetStr} | ${icon} |\n`;
  }

  md += `\n</details>\n`;

  // Errors and warnings
  if (hasErrors) {
    md += `\n### Errors\n\n`;
    for (const error of analysis.errors) {
      md += `- ${error.message}\n`;
    }
  }

  if (hasWarnings) {
    md += `\n### Warnings\n\n`;
    for (const warning of analysis.warnings) {
      md += `- ${warning.message}\n`;
    }
  }

  return md;
}

/**
 * Save JSON report
 */
function saveReport(analysis) {
  const report = {
    ...analysis,
    dependencies: analyzeDependencies(),
    treeShaking: analyzeTreeShaking(),
  };

  fs.writeFileSync(config.reportFile, JSON.stringify(report, null, 2));
  console.log(
    `📄 Report saved to: ${path.relative(process.cwd(), config.reportFile)}\n`
  );
}

// =============================================================================
// Comparison
// =============================================================================

/**
 * Compare with previous build
 */
function compareWithPrevious(currentAnalysis) {
  const previousPath = path.resolve(
    __dirname,
    '../dist/bundle-report.previous.json'
  );

  if (!fs.existsSync(previousPath)) {
    return null;
  }

  const previous = JSON.parse(fs.readFileSync(previousPath, 'utf-8'));

  const comparison = {
    totalSizeDiff:
      currentAnalysis.summary.total.gzip - previous.summary.total.gzip,
    chunkDiffs: [],
  };

  // Compare chunks
  for (const chunk of currentAnalysis.chunks) {
    const prevChunk = previous.chunks?.find((c) => c.name === chunk.name);

    if (prevChunk) {
      const diff = chunk.sizes.gzip - prevChunk.sizes.gzip;
      if (Math.abs(diff) > 1024) {
        // Only report changes > 1KB
        comparison.chunkDiffs.push({
          name: chunk.name,
          diff,
          percent: ((diff / prevChunk.sizes.gzip) * 100).toFixed(1),
        });
      }
    }
  }

  return comparison;
}

// =============================================================================
// Main
// =============================================================================

async function main() {
  const args = process.argv.slice(2);
  const showJson = args.includes('--json');
  const saveOnly = args.includes('--save');
  const ciMode = args.includes('--ci');

  try {
    const analysis = analyzeFiles();

    if (ciMode) {
      printCIReport(analysis);

      // Write markdown report for PR comments / step summary
      const markdownReport = generateMarkdownReport(analysis);
      const mdPath = path.resolve(config.distDir, 'bundle-size-report.md');
      fs.writeFileSync(mdPath, markdownReport);
      console.log(`Markdown report saved to: ${mdPath}`);
    } else if (showJson) {
      console.log(JSON.stringify(analysis, null, 2));
    } else if (!saveOnly) {
      printReport(analysis);

      // Compare with previous
      const comparison = compareWithPrevious(analysis);
      if (comparison) {
        console.log('📈 Comparison with Previous Build');
        console.log(
          '───────────────────────────────────────────────────────────'
        );

        const diffSign = comparison.totalSizeDiff > 0 ? '+' : '';
        console.log(
          `  Total size change: ${diffSign}${formatBytes(
            comparison.totalSizeDiff
          )}`
        );

        if (comparison.chunkDiffs.length > 0) {
          console.log('  Chunk changes:');
          for (const diff of comparison.chunkDiffs) {
            const sign = diff.diff > 0 ? '+' : '';
            console.log(
              `    ${diff.name}: ${sign}${formatBytes(diff.diff)} (${sign}${
                diff.percent
              }%)`
            );
          }
        }
        console.log('');
      }
    }

    // Only save report/archive in non-CI mode
    if (!ciMode) {
      saveReport(analysis);

      // Archive current as previous for next comparison
      if (fs.existsSync(config.reportFile)) {
        const previousPath = path.resolve(
          __dirname,
          '../dist/bundle-report.previous.json'
        );
        fs.copyFileSync(config.reportFile, previousPath);
      }
    }

    // Exit with error code if there are errors
    if (analysis.errors.length > 0) {
      process.exit(1);
    }
  } catch (error) {
    console.error('Error analyzing bundle:', error);
    process.exit(1);
  }
}

main();
