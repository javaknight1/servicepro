import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

// T262: Inter is bundled, so no page load sends visitor IPs to Google Fonts
// and the CSP doesn't allow the font CDN.
const frontendRoot = resolve(__dirname, '../..');
const read = (file: string) =>
  readFileSync(resolve(frontendRoot, file), 'utf8');

describe('self-hosted fonts (T262)', () => {
  it('index.html does not load Google Fonts', () => {
    const html = read('index.html');
    expect(html).not.toMatch(/fonts\.googleapis\.com|fonts\.gstatic\.com/);
  });

  it('the static-asset CSP does not allow Google Fonts', () => {
    const csp = read('public/_headers')
      .split('\n')
      .find((line) => line.trim().startsWith('Content-Security-Policy:'));
    expect(csp).toBeDefined();
    expect(csp).not.toMatch(/fonts\.googleapis\.com|fonts\.gstatic\.com/);
    expect(csp).toMatch(/font-src 'self' data:;/);
  });

  it('main.tsx imports the bundled Inter font', () => {
    expect(read('src/main.tsx')).toContain(
      "import '@fontsource-variable/inter';"
    );
  });
});
