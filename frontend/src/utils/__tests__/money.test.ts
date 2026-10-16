import { describe, it, expect } from 'vitest';
import {
  formatCents,
  formatDollars,
  lineTotal,
  percentToTaxRate,
  roundToCents,
  taxRateToPercent,
  toAmount,
} from '../money';

describe('money utils', () => {
  it('formats API decimal dollars (string or number)', () => {
    expect(formatDollars('1234.5')).toBe('$1,234.50');
    expect(formatDollars(1080)).toBe('$1,080.00');
    expect(formatDollars(null)).toBe('$0.00');
    expect(formatDollars('1234.56', { wholeUnits: true })).toBe('$1,235');
  });

  it('formats integer cents', () => {
    expect(formatCents(1250)).toBe('$12.50');
    expect(formatCents(0)).toBe('$0.00');
  });

  it('dollars and cents helpers differ by exactly 100x', () => {
    expect(formatCents(4999)).toBe(formatDollars(49.99));
  });

  it('parses decimals safely', () => {
    expect(toAmount('12.30')).toBe(12.3);
    expect(toAmount('abc')).toBe(0);
    expect(toAmount(undefined)).toBe(0);
  });

  it('rounds to cents', () => {
    expect(roundToCents(1.005)).toBe(1.01);
    expect(roundToCents(0.1 + 0.2)).toBe(0.3);
    expect(lineTotal('3', '19.99')).toBe(59.97);
  });

  it('shows tax rates without float artefacts', () => {
    expect(taxRateToPercent(0.07)).toBe('7');
    expect(taxRateToPercent(0.029)).toBe('2.9');
    expect(taxRateToPercent('0.0825')).toBe('8.25');
    expect(taxRateToPercent(0)).toBe('0');
    expect(taxRateToPercent(null)).toBe('0');
  });

  it('converts percent input back to a rate', () => {
    expect(percentToTaxRate('7')).toBe(0.07);
    expect(percentToTaxRate('2.9')).toBe(0.029);
    expect(percentToTaxRate('')).toBe(0);
  });
});
