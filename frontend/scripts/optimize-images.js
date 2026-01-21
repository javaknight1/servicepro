#!/usr/bin/env node

/**
 * =============================================================================
 * Image Optimization Script
 * =============================================================================
 * Optimizes images using Sharp for production builds
 */

const fs = require('fs');
const path = require('path');
const sharp = require('sharp');

// =============================================================================
// Configuration
// =============================================================================

const config = {
  // Input/output directories
  inputDir: path.resolve(__dirname, '../src/assets/images'),
  outputDir: path.resolve(__dirname, '../public/images'),

  // Quality settings
  quality: {
    jpeg: 80,
    webp: 80,
    avif: 65,
    png: 80,
  },

  // Responsive breakpoints
  breakpoints: [320, 640, 768, 1024, 1280, 1536, 1920],

  // Output formats
  formats: ['webp', 'avif'],

  // File patterns to process
  patterns: /\.(jpe?g|png|gif)$/i,

  // Skip files smaller than this (bytes)
  minSize: 1024,

  // Maximum dimensions (will resize if larger)
  maxDimensions: {
    width: 2560,
    height: 2560,
  },

  // Placeholder settings
  placeholder: {
    enabled: true,
    size: 20,
    quality: 50,
    blur: 10,
  },
};

// =============================================================================
// Utilities
// =============================================================================

/**
 * Get all image files recursively
 */
function getImageFiles(dir, files = []) {
  if (!fs.existsSync(dir)) {
    return files;
  }

  const items = fs.readdirSync(dir, { withFileTypes: true });

  for (const item of items) {
    const fullPath = path.join(dir, item.name);

    if (item.isDirectory()) {
      getImageFiles(fullPath, files);
    } else if (config.patterns.test(item.name)) {
      files.push(fullPath);
    }
  }

  return files;
}

/**
 * Ensure directory exists
 */
function ensureDir(dir) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
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
 * Get relative output path
 */
function getOutputPath(inputPath, suffix = '', ext = null) {
  const relativePath = path.relative(config.inputDir, inputPath);
  const parsed = path.parse(relativePath);
  const newExt = ext || parsed.ext;
  const newName = `${parsed.name}${suffix}${newExt}`;

  return path.join(config.outputDir, parsed.dir, newName);
}

// =============================================================================
// Optimization Functions
// =============================================================================

/**
 * Optimize a single image
 */
async function optimizeImage(inputPath) {
  const stats = fs.statSync(inputPath);

  // Skip small files
  if (stats.size < config.minSize) {
    return { skipped: true, reason: 'too small' };
  }

  const results = {
    input: {
      path: inputPath,
      size: stats.size,
    },
    outputs: [],
    savings: 0,
  };

  try {
    // Get image metadata
    const image = sharp(inputPath);
    const metadata = await image.metadata();

    // Calculate if resize is needed
    let needsResize = false;
    let targetWidth = metadata.width;
    let targetHeight = metadata.height;

    if (metadata.width > config.maxDimensions.width) {
      targetWidth = config.maxDimensions.width;
      targetHeight = Math.round(
        (metadata.height * config.maxDimensions.width) / metadata.width
      );
      needsResize = true;
    }

    if (targetHeight > config.maxDimensions.height) {
      targetHeight = config.maxDimensions.height;
      targetWidth = Math.round(
        (metadata.width * config.maxDimensions.height) / metadata.height
      );
      needsResize = true;
    }

    // Base pipeline
    let pipeline = sharp(inputPath);

    if (needsResize) {
      pipeline = pipeline.resize(targetWidth, targetHeight, {
        fit: 'inside',
        withoutEnlargement: true,
      });
    }

    // Process original format (optimized)
    const ext = path.extname(inputPath).toLowerCase();
    const originalOutput = getOutputPath(inputPath);
    ensureDir(path.dirname(originalOutput));

    if (ext === '.jpg' || ext === '.jpeg') {
      await pipeline
        .clone()
        .jpeg({ quality: config.quality.jpeg, mozjpeg: true })
        .toFile(originalOutput);
    } else if (ext === '.png') {
      await pipeline
        .clone()
        .png({ quality: config.quality.png, compressionLevel: 9 })
        .toFile(originalOutput);
    } else if (ext === '.gif') {
      // Just copy GIFs (Sharp doesn't support animated GIF optimization well)
      fs.copyFileSync(inputPath, originalOutput);
    }

    const originalStats = fs.statSync(originalOutput);
    results.outputs.push({
      path: originalOutput,
      format: ext.slice(1),
      size: originalStats.size,
      dimensions: { width: targetWidth, height: targetHeight },
    });
    results.savings += stats.size - originalStats.size;

    // Generate WebP
    if (config.formats.includes('webp')) {
      const webpOutput = getOutputPath(inputPath, '', '.webp');
      await pipeline
        .clone()
        .webp({ quality: config.quality.webp })
        .toFile(webpOutput);

      const webpStats = fs.statSync(webpOutput);
      results.outputs.push({
        path: webpOutput,
        format: 'webp',
        size: webpStats.size,
        dimensions: { width: targetWidth, height: targetHeight },
      });
    }

    // Generate AVIF
    if (config.formats.includes('avif')) {
      const avifOutput = getOutputPath(inputPath, '', '.avif');
      await pipeline
        .clone()
        .avif({ quality: config.quality.avif })
        .toFile(avifOutput);

      const avifStats = fs.statSync(avifOutput);
      results.outputs.push({
        path: avifOutput,
        format: 'avif',
        size: avifStats.size,
        dimensions: { width: targetWidth, height: targetHeight },
      });
    }

    // Generate responsive images
    for (const breakpoint of config.breakpoints) {
      if (breakpoint >= targetWidth) continue;

      const responsiveHeight = Math.round(
        (targetHeight * breakpoint) / targetWidth
      );

      // WebP responsive
      if (config.formats.includes('webp')) {
        const responsiveWebp = getOutputPath(
          inputPath,
          `-${breakpoint}w`,
          '.webp'
        );
        await sharp(inputPath)
          .resize(breakpoint, responsiveHeight, { fit: 'inside' })
          .webp({ quality: config.quality.webp })
          .toFile(responsiveWebp);

        results.outputs.push({
          path: responsiveWebp,
          format: 'webp',
          size: fs.statSync(responsiveWebp).size,
          dimensions: { width: breakpoint, height: responsiveHeight },
          responsive: true,
        });
      }
    }

    // Generate placeholder
    if (config.placeholder.enabled) {
      const placeholderOutput = getOutputPath(
        inputPath,
        '-placeholder',
        '.webp'
      );
      await sharp(inputPath)
        .resize(config.placeholder.size)
        .blur(config.placeholder.blur)
        .webp({ quality: config.placeholder.quality })
        .toFile(placeholderOutput);

      const placeholderStats = fs.statSync(placeholderOutput);
      results.outputs.push({
        path: placeholderOutput,
        format: 'webp',
        size: placeholderStats.size,
        isPlaceholder: true,
      });

      // Also generate base64 placeholder for inline use
      const placeholderBuffer = await sharp(inputPath)
        .resize(config.placeholder.size)
        .blur(config.placeholder.blur)
        .webp({ quality: config.placeholder.quality })
        .toBuffer();

      results.placeholderBase64 = `data:image/webp;base64,${placeholderBuffer.toString(
        'base64'
      )}`;
    }

    return results;
  } catch (error) {
    return { error: error.message, input: inputPath };
  }
}

/**
 * Generate image manifest
 */
function generateManifest(results) {
  const manifest = {};

  for (const result of results) {
    if (result.skipped || result.error) continue;

    const relativePath = path.relative(config.inputDir, result.input.path);
    const key = relativePath.replace(/\\/g, '/');

    manifest[key] = {
      original: {
        size: result.input.size,
      },
      outputs: result.outputs.map((output) => ({
        path: path.relative(config.outputDir, output.path).replace(/\\/g, '/'),
        format: output.format,
        size: output.size,
        dimensions: output.dimensions,
        responsive: output.responsive || false,
        isPlaceholder: output.isPlaceholder || false,
      })),
      savings: result.savings,
      savingsPercent: ((result.savings / result.input.size) * 100).toFixed(1),
      placeholder: result.placeholderBase64 || null,
    };
  }

  return manifest;
}

// =============================================================================
// Report
// =============================================================================

/**
 * Print optimization report
 */
function printReport(results) {
  let totalInput = 0;
  let totalOutput = 0;
  let totalSavings = 0;
  let processedCount = 0;
  let skippedCount = 0;
  let errorCount = 0;

  console.log('\n═══════════════════════════════════════════════════════════');
  console.log('              Image Optimization Report');
  console.log('═══════════════════════════════════════════════════════════\n');

  for (const result of results) {
    if (result.skipped) {
      skippedCount++;
      continue;
    }

    if (result.error) {
      errorCount++;
      console.log(`❌ ${result.input}: ${result.error}`);
      continue;
    }

    processedCount++;
    totalInput += result.input.size;

    const originalOutput = result.outputs.find(
      (o) =>
        !o.responsive &&
        !o.isPlaceholder &&
        o.format !== 'webp' &&
        o.format !== 'avif'
    );

    if (originalOutput) {
      totalOutput += originalOutput.size;
    }

    totalSavings += result.savings;

    // Print individual file stats
    const relativePath = path.relative(config.inputDir, result.input.path);
    const savingsPercent = ((result.savings / result.input.size) * 100).toFixed(
      1
    );

    console.log(`📷 ${relativePath}`);
    console.log(`   Original: ${formatBytes(result.input.size)}`);
    console.log(`   Outputs:`);

    for (const output of result.outputs) {
      if (output.isPlaceholder) continue;

      const label = output.responsive
        ? `${output.format} ${output.dimensions.width}w`
        : output.format;

      console.log(`     • ${label.padEnd(15)} ${formatBytes(output.size)}`);
    }

    if (result.savings > 0) {
      console.log(
        `   💾 Saved: ${formatBytes(result.savings)} (${savingsPercent}%)`
      );
    }

    console.log('');
  }

  // Summary
  console.log('───────────────────────────────────────────────────────────');
  console.log('📊 Summary');
  console.log('───────────────────────────────────────────────────────────');
  console.log(`  Processed:  ${processedCount} images`);
  console.log(`  Skipped:    ${skippedCount} images`);
  console.log(`  Errors:     ${errorCount} images`);
  console.log('');
  console.log(`  Input size:  ${formatBytes(totalInput)}`);
  console.log(`  Output size: ${formatBytes(totalOutput)}`);
  console.log(
    `  Total saved: ${formatBytes(totalSavings)} (${(
      (totalSavings / totalInput) *
      100
    ).toFixed(1)}%)`
  );
  console.log('═══════════════════════════════════════════════════════════\n');
}

// =============================================================================
// Main
// =============================================================================

async function main() {
  const args = process.argv.slice(2);
  const dryRun = args.includes('--dry-run');
  const single = args.find((a) => a.startsWith('--file='))?.split('=')[1];

  console.log('🖼️  Image Optimization');
  console.log(`   Input:  ${config.inputDir}`);
  console.log(`   Output: ${config.outputDir}`);

  if (dryRun) {
    console.log('   Mode:   Dry run (no files will be written)');
  }

  console.log('');

  // Ensure output directory exists
  ensureDir(config.outputDir);

  // Get files to process
  let files;
  if (single) {
    files = [path.resolve(single)];
  } else {
    files = getImageFiles(config.inputDir);
  }

  if (files.length === 0) {
    console.log('No images found to optimize.');
    return;
  }

  console.log(`Found ${files.length} images to process...\n`);

  // Process images
  const results = [];

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    const progress = `[${i + 1}/${files.length}]`;

    process.stdout.write(`${progress} Processing ${path.basename(file)}...`);

    if (dryRun) {
      console.log(' (skipped - dry run)');
      continue;
    }

    const result = await optimizeImage(file);
    results.push(result);

    if (result.error) {
      console.log(` ❌ ${result.error}`);
    } else if (result.skipped) {
      console.log(` ⏭️ ${result.reason}`);
    } else {
      console.log(` ✓ ${result.outputs.length} outputs`);
    }
  }

  if (!dryRun) {
    // Print report
    printReport(results);

    // Generate manifest
    const manifest = generateManifest(results);
    const manifestPath = path.join(config.outputDir, 'image-manifest.json');
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));
    console.log(`📄 Manifest saved to: ${manifestPath}\n`);
  }
}

main().catch((error) => {
  console.error('Error:', error);
  process.exit(1);
});
