#!/usr/bin/env node

/**
 * =============================================================================
 * Dependencies Audit Script
 * =============================================================================
 * Analyzes project dependencies for security, size, and optimization
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// =============================================================================
// Configuration
// =============================================================================

const config = {
  packagePath: path.resolve(__dirname, '../package.json'),
  lockPath: path.resolve(__dirname, '../package-lock.json'),
  nodeModulesPath: path.resolve(__dirname, '../node_modules'),
  reportPath: path.resolve(__dirname, '../dist/deps-audit.json'),

  // Size thresholds (in KB)
  sizeThresholds: {
    warning: 100,
    error: 500,
  },

  // Known large packages that should be lazy loaded
  largePkgs: [
    'moment',
    'lodash',
    'lodash-es',
    '@material-ui/core',
    '@mui/material',
    'antd',
    'bootstrap',
    'jquery',
    '@fortawesome/fontawesome-free',
  ],

  // Packages that don't tree-shake well
  noTreeShake: ['lodash', 'moment', 'rxjs'],

  // Known alternatives for optimization
  alternatives: {
    moment: 'date-fns or dayjs',
    lodash: 'lodash-es (for tree shaking) or native methods',
    axios: 'native fetch API',
    jquery: 'native DOM APIs',
    classnames: 'clsx (smaller)',
    uuid: 'crypto.randomUUID() (native)',
  },
};

// =============================================================================
// Utilities
// =============================================================================

/**
 * Execute command and return output
 */
function exec(cmd, options = {}) {
  try {
    return execSync(cmd, {
      encoding: 'utf-8',
      stdio: ['pipe', 'pipe', 'pipe'],
      ...options,
    });
  } catch (error) {
    return error.stdout || '';
  }
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
 * Get directory size recursively
 */
function getDirectorySize(dirPath) {
  if (!fs.existsSync(dirPath)) return 0;

  let size = 0;
  const files = fs.readdirSync(dirPath, { withFileTypes: true });

  for (const file of files) {
    const filePath = path.join(dirPath, file.name);

    if (file.isDirectory()) {
      size += getDirectorySize(filePath);
    } else {
      size += fs.statSync(filePath).size;
    }
  }

  return size;
}

/**
 * Parse semver version
 */
function parseVersion(version) {
  const match = version.replace(/^[\^~]/, '').match(/(\d+)\.(\d+)\.(\d+)/);
  if (!match) return { major: 0, minor: 0, patch: 0 };
  return {
    major: parseInt(match[1], 10),
    minor: parseInt(match[2], 10),
    patch: parseInt(match[3], 10),
  };
}

// =============================================================================
// Analysis Functions
// =============================================================================

/**
 * Analyze package.json dependencies
 */
function analyzeDependencies() {
  const pkg = JSON.parse(fs.readFileSync(config.packagePath, 'utf-8'));

  const analysis = {
    dependencies: Object.keys(pkg.dependencies || {}),
    devDependencies: Object.keys(pkg.devDependencies || {}),
    peerDependencies: Object.keys(pkg.peerDependencies || {}),
    counts: {
      production: Object.keys(pkg.dependencies || {}).length,
      development: Object.keys(pkg.devDependencies || {}).length,
      peer: Object.keys(pkg.peerDependencies || {}).length,
    },
  };

  analysis.counts.total =
    analysis.counts.production +
    analysis.counts.development +
    analysis.counts.peer;

  return analysis;
}

/**
 * Check for duplicate packages
 */
function checkDuplicates() {
  const duplicates = [];

  try {
    const output = exec('npm ls --json 2>/dev/null');
    const tree = JSON.parse(output);

    // Track package versions
    const versions = new Map();

    function traverse(deps, depth = 0) {
      if (!deps) return;

      for (const [name, info] of Object.entries(deps)) {
        const version = info.version || 'unknown';

        if (!versions.has(name)) {
          versions.set(name, []);
        }

        const existing = versions.get(name);
        if (!existing.includes(version)) {
          existing.push(version);
        }

        if (info.dependencies) {
          traverse(info.dependencies, depth + 1);
        }
      }
    }

    traverse(tree.dependencies);

    // Find duplicates
    for (const [name, versionList] of versions) {
      if (versionList.length > 1) {
        duplicates.push({
          package: name,
          versions: versionList,
          count: versionList.length,
        });
      }
    }
  } catch (error) {
    // Fallback if npm ls fails
  }

  return duplicates.sort((a, b) => b.count - a.count);
}

/**
 * Analyze package sizes
 */
function analyzePackageSizes() {
  const sizes = [];
  const pkg = JSON.parse(fs.readFileSync(config.packagePath, 'utf-8'));
  const allDeps = {
    ...pkg.dependencies,
    ...pkg.devDependencies,
  };

  for (const [name] of Object.entries(allDeps)) {
    const pkgPath = path.join(config.nodeModulesPath, name);

    if (fs.existsSync(pkgPath)) {
      const size = getDirectorySize(pkgPath);

      sizes.push({
        package: name,
        size,
        sizeFormatted: formatBytes(size),
        isLarge: size > config.sizeThresholds.warning * 1024,
        isVeryLarge: size > config.sizeThresholds.error * 1024,
      });
    }
  }

  return sizes.sort((a, b) => b.size - a.size);
}

/**
 * Check for security vulnerabilities
 */
function checkSecurity() {
  const result = {
    vulnerabilities: [],
    summary: { low: 0, moderate: 0, high: 0, critical: 0 },
    hasVulnerabilities: false,
  };

  try {
    const output = exec('npm audit --json 2>/dev/null');
    const audit = JSON.parse(output);

    if (audit.vulnerabilities) {
      for (const [name, info] of Object.entries(audit.vulnerabilities)) {
        result.vulnerabilities.push({
          package: name,
          severity: info.severity,
          via: Array.isArray(info.via)
            ? info.via.map((v) => (typeof v === 'string' ? v : v.title))
            : [],
          range: info.range,
          fixAvailable: info.fixAvailable,
        });

        result.summary[info.severity]++;
      }
    }

    result.hasVulnerabilities = result.vulnerabilities.length > 0;
  } catch (error) {
    // npm audit might fail in some environments
  }

  return result;
}

/**
 * Check for outdated packages
 */
function checkOutdated() {
  const outdated = [];

  try {
    const output = exec('npm outdated --json 2>/dev/null');
    const data = JSON.parse(output || '{}');

    for (const [name, info] of Object.entries(data)) {
      const current = parseVersion(info.current || '0.0.0');
      const latest = parseVersion(info.latest || '0.0.0');

      outdated.push({
        package: name,
        current: info.current,
        wanted: info.wanted,
        latest: info.latest,
        type: info.type,
        isMajor: latest.major > current.major,
        isMinor: latest.minor > current.minor && latest.major === current.major,
        isPatch: latest.patch > current.patch && latest.minor === current.minor,
      });
    }
  } catch (error) {
    // npm outdated might fail
  }

  return outdated;
}

/**
 * Check for tree-shaking issues
 */
function checkTreeShaking() {
  const issues = [];
  const pkg = JSON.parse(fs.readFileSync(config.packagePath, 'utf-8'));
  const deps = Object.keys(pkg.dependencies || {});

  for (const dep of deps) {
    // Check against known non-tree-shakeable packages
    if (config.noTreeShake.some((p) => dep.includes(p))) {
      issues.push({
        package: dep,
        issue: 'Package does not support tree-shaking',
        recommendation: config.alternatives[dep] || 'Consider alternatives',
      });
    }

    // Check package.json of dependency
    const depPkgPath = path.join(config.nodeModulesPath, dep, 'package.json');
    if (fs.existsSync(depPkgPath)) {
      const depPkg = JSON.parse(fs.readFileSync(depPkgPath, 'utf-8'));

      // Check for module/exports field
      const hasESM =
        depPkg.module || depPkg.exports || depPkg.type === 'module';
      const hasSideEffects = depPkg.sideEffects !== false;

      if (!hasESM) {
        issues.push({
          package: dep,
          issue: 'Package may not support ES modules',
          recommendation: 'Check if ESM version is available',
        });
      }

      if (hasSideEffects && !depPkg.sideEffects) {
        // Only warn if sideEffects is not explicitly set
        if (depPkg.sideEffects === undefined) {
          issues.push({
            package: dep,
            issue: 'Package does not declare sideEffects',
            recommendation: 'Tree-shaking may be less effective',
          });
        }
      }
    }
  }

  return issues;
}

/**
 * Check for potential optimizations
 */
function checkOptimizations() {
  const optimizations = [];
  const pkg = JSON.parse(fs.readFileSync(config.packagePath, 'utf-8'));
  const deps = Object.keys(pkg.dependencies || {});

  // Check for large packages
  for (const dep of deps) {
    if (config.largePkgs.some((p) => dep.includes(p))) {
      optimizations.push({
        package: dep,
        type: 'large-package',
        recommendation: 'Consider lazy loading or alternative',
        alternative: config.alternatives[dep] || null,
      });
    }
  }

  // Check for packages with better alternatives
  for (const [pkg, alt] of Object.entries(config.alternatives)) {
    if (deps.includes(pkg)) {
      optimizations.push({
        package: pkg,
        type: 'has-alternative',
        recommendation: `Consider using ${alt}`,
        alternative: alt,
      });
    }
  }

  // Check for potential duplicate functionality
  const duplicateFunctionality = [
    { packages: ['axios', 'node-fetch', 'got', 'ky'], type: 'HTTP client' },
    {
      packages: ['moment', 'dayjs', 'date-fns', 'luxon'],
      type: 'Date library',
    },
    { packages: ['lodash', 'underscore', 'ramda'], type: 'Utility library' },
    { packages: ['uuid', 'nanoid', 'shortid'], type: 'ID generation' },
  ];

  for (const { packages, type } of duplicateFunctionality) {
    const found = packages.filter((p) =>
      deps.some((d) => d === p || d.startsWith(`${p}/`))
    );
    if (found.length > 1) {
      optimizations.push({
        packages: found,
        type: 'duplicate-functionality',
        recommendation: `Multiple ${type} packages detected. Consider consolidating.`,
      });
    }
  }

  return optimizations;
}

/**
 * Detect unused dependencies
 */
function checkUnusedDependencies() {
  const unused = [];
  const pkg = JSON.parse(fs.readFileSync(config.packagePath, 'utf-8'));
  const deps = Object.keys(pkg.dependencies || {});

  // Get all source files
  const srcDir = path.resolve(__dirname, '../src');
  const sourceFiles = [];

  function findSourceFiles(dir) {
    if (!fs.existsSync(dir)) return;

    const items = fs.readdirSync(dir, { withFileTypes: true });

    for (const item of items) {
      const fullPath = path.join(dir, item.name);

      if (item.isDirectory() && !item.name.includes('node_modules')) {
        findSourceFiles(fullPath);
      } else if (/\.(ts|tsx|js|jsx)$/.test(item.name)) {
        sourceFiles.push(fullPath);
      }
    }
  }

  findSourceFiles(srcDir);

  // Read all source content
  let allContent = '';
  for (const file of sourceFiles) {
    allContent += fs.readFileSync(file, 'utf-8');
  }

  // Check each dependency
  for (const dep of deps) {
    // Check for common import patterns
    const patterns = [
      `from '${dep}'`,
      `from "${dep}"`,
      `from '${dep}/`,
      `from "${dep}/`,
      `require('${dep}')`,
      `require("${dep}")`,
      `require('${dep}/`,
      `require("${dep}/`,
      `import '${dep}'`,
      `import "${dep}"`,
    ];

    const isUsed = patterns.some((p) => allContent.includes(p));

    if (!isUsed) {
      // Some packages are used indirectly
      const indirectUse = [
        '@types/',
        'typescript',
        'vite',
        'vitest',
        'eslint',
        'postcss',
        'tailwindcss',
        'autoprefixer',
      ];

      if (!indirectUse.some((p) => dep.includes(p))) {
        unused.push({
          package: dep,
          note: 'No direct imports found. May be used indirectly.',
        });
      }
    }
  }

  return unused;
}

// =============================================================================
// Report Generation
// =============================================================================

/**
 * Print audit report
 */
function printReport(results) {
  console.log('\n' + '='.repeat(65));
  console.log('                    Dependencies Audit Report');
  console.log('='.repeat(65) + '\n');

  // Summary
  console.log('Summary');
  console.log('-'.repeat(65));
  console.log(
    `  Production dependencies:  ${results.dependencies.counts.production}`
  );
  console.log(
    `  Dev dependencies:         ${results.dependencies.counts.development}`
  );
  console.log(
    `  Total:                    ${results.dependencies.counts.total}`
  );
  console.log('');

  // Security
  console.log('Security Vulnerabilities');
  console.log('-'.repeat(65));
  if (results.security.hasVulnerabilities) {
    console.log(`  Critical: ${results.security.summary.critical}`);
    console.log(`  High:     ${results.security.summary.high}`);
    console.log(`  Moderate: ${results.security.summary.moderate}`);
    console.log(`  Low:      ${results.security.summary.low}`);

    if (
      results.security.summary.critical > 0 ||
      results.security.summary.high > 0
    ) {
      console.log('\n  Run "npm audit fix" to address vulnerabilities');
    }
  } else {
    console.log('  No vulnerabilities found');
  }
  console.log('');

  // Duplicates
  if (results.duplicates.length > 0) {
    console.log('Duplicate Packages');
    console.log('-'.repeat(65));
    for (const dup of results.duplicates.slice(0, 10)) {
      console.log(`  ${dup.package}: ${dup.versions.join(', ')}`);
    }
    if (results.duplicates.length > 10) {
      console.log(`  ... and ${results.duplicates.length - 10} more`);
    }
    console.log('');
  }

  // Large packages
  const largePackages = results.sizes.filter((s) => s.isLarge).slice(0, 10);
  if (largePackages.length > 0) {
    console.log('Largest Packages');
    console.log('-'.repeat(65));
    for (const pkg of largePackages) {
      const icon = pkg.isVeryLarge ? '!!' : ' >';
      console.log(`  ${icon} ${pkg.package.padEnd(35)} ${pkg.sizeFormatted}`);
    }
    console.log('');
  }

  // Outdated
  const majorOutdated = results.outdated.filter((o) => o.isMajor);
  if (majorOutdated.length > 0) {
    console.log('Major Updates Available');
    console.log('-'.repeat(65));
    for (const pkg of majorOutdated.slice(0, 10)) {
      console.log(
        `  ${pkg.package.padEnd(30)} ${pkg.current} -> ${pkg.latest}`
      );
    }
    console.log('');
  }

  // Tree-shaking issues
  if (results.treeShaking.length > 0) {
    console.log('Tree-Shaking Issues');
    console.log('-'.repeat(65));
    for (const issue of results.treeShaking.slice(0, 5)) {
      console.log(`  ${issue.package}: ${issue.issue}`);
    }
    console.log('');
  }

  // Optimizations
  if (results.optimizations.length > 0) {
    console.log('Optimization Suggestions');
    console.log('-'.repeat(65));
    for (const opt of results.optimizations.slice(0, 5)) {
      if (opt.package) {
        console.log(`  ${opt.package}: ${opt.recommendation}`);
      } else if (opt.packages) {
        console.log(`  ${opt.packages.join(', ')}: ${opt.recommendation}`);
      }
    }
    console.log('');
  }

  // Potentially unused
  if (results.unused.length > 0) {
    console.log('Potentially Unused Dependencies');
    console.log('-'.repeat(65));
    for (const pkg of results.unused.slice(0, 10)) {
      console.log(`  ? ${pkg.package}`);
    }
    console.log(
      '  Note: These may be used indirectly. Verify before removing.'
    );
    console.log('');
  }

  // Total size
  const totalSize = results.sizes.reduce((sum, s) => sum + s.size, 0);
  console.log('-'.repeat(65));
  console.log(`Total node_modules size: ${formatBytes(totalSize)}`);
  console.log('='.repeat(65) + '\n');

  // Exit code
  if (
    results.security.summary.critical > 0 ||
    results.security.summary.high > 0
  ) {
    return 1;
  }

  return 0;
}

/**
 * Save JSON report
 */
function saveReport(results) {
  const dir = path.dirname(config.reportPath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }

  fs.writeFileSync(config.reportPath, JSON.stringify(results, null, 2));
  console.log(
    `Report saved to: ${path.relative(process.cwd(), config.reportPath)}\n`
  );
}

// =============================================================================
// Main
// =============================================================================

async function main() {
  const args = process.argv.slice(2);
  const showJson = args.includes('--json');
  const skipSecurity = args.includes('--skip-security');

  console.log('Analyzing dependencies...\n');

  const results = {
    timestamp: new Date().toISOString(),
    dependencies: analyzeDependencies(),
    duplicates: checkDuplicates(),
    sizes: analyzePackageSizes(),
    security: skipSecurity
      ? { vulnerabilities: [], summary: {}, hasVulnerabilities: false }
      : checkSecurity(),
    outdated: checkOutdated(),
    treeShaking: checkTreeShaking(),
    optimizations: checkOptimizations(),
    unused: checkUnusedDependencies(),
  };

  if (showJson) {
    console.log(JSON.stringify(results, null, 2));
    return;
  }

  const exitCode = printReport(results);
  saveReport(results);

  process.exit(exitCode);
}

main().catch((error) => {
  console.error('Error:', error);
  process.exit(1);
});
