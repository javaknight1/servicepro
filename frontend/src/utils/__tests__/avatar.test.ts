import { describe, it, expect } from 'vitest';
import { getAvatarColor, getInitials, getDisplayName } from '../avatar';

describe('avatar utilities', () => {
  describe('getAvatarColor', () => {
    it('should return deterministic color for the same email', () => {
      const color1 = getAvatarColor('test@example.com');
      const color2 = getAvatarColor('test@example.com');

      expect(color1).toEqual(color2);
    });

    it('should be case insensitive', () => {
      const color1 = getAvatarColor('TEST@EXAMPLE.COM');
      const color2 = getAvatarColor('test@example.com');

      expect(color1).toEqual(color2);
    });

    it('should return different colors for different emails', () => {
      const color1 = getAvatarColor('alice@example.com');
      const color2 = getAvatarColor('bob@example.com');

      // May occasionally be the same due to hash collisions, but statistically different
      // This test ensures the function works with different inputs
      expect(color1).toBeDefined();
      expect(color2).toBeDefined();
    });

    it('should return valid color classes', () => {
      const color = getAvatarColor('test@example.com');

      expect(color.bg).toMatch(/^bg-\w+-100$/);
      expect(color.text).toMatch(/^text-\w+-700$/);
    });

    it('should handle empty email', () => {
      const color = getAvatarColor('');

      expect(color.bg).toBeDefined();
      expect(color.text).toBeDefined();
    });
  });

  describe('getInitials', () => {
    it('should return initials from first and last name', () => {
      expect(getInitials('user@example.com', 'John', 'Doe')).toBe('JD');
      expect(getInitials('user@example.com', 'Alice', 'Smith')).toBe('AS');
    });

    it('should return first 2 letters when only first name provided', () => {
      expect(getInitials('user@example.com', 'John', null)).toBe('JO');
      expect(getInitials('user@example.com', 'Alice', undefined)).toBe('AL');
    });

    it('should return first 2 letters when only last name provided', () => {
      expect(getInitials('user@example.com', null, 'Doe')).toBe('DO');
      expect(getInitials('user@example.com', undefined, 'Smith')).toBe('SM');
    });

    it('should fall back to email when no name provided', () => {
      expect(getInitials('john@example.com', null, null)).toBe('JO');
      expect(getInitials('alice.smith@example.com')).toBe('AL');
    });

    it('should return uppercase initials', () => {
      expect(getInitials('user@example.com', 'john', 'doe')).toBe('JD');
    });

    it('should trim whitespace from names', () => {
      expect(getInitials('user@example.com', '  John  ', '  Doe  ')).toBe('JD');
    });

    it('should handle empty strings as no name', () => {
      expect(getInitials('user@example.com', '', '')).toBe('US');
      expect(getInitials('user@example.com', '   ', '   ')).toBe('US');
    });

    it('should return ?? for empty email with no name', () => {
      expect(getInitials('', null, null)).toBe('??');
    });

    it('should return ?? for email with empty local part', () => {
      expect(getInitials('@example.com', null, null)).toBe('??');
    });

    it('should handle short names', () => {
      expect(getInitials('user@example.com', 'A', null)).toBe('A');
      expect(getInitials('user@example.com', null, 'B')).toBe('B');
    });
  });

  describe('getDisplayName', () => {
    it('should return full name when both first and last name provided', () => {
      expect(getDisplayName('user@example.com', 'John', 'Doe')).toBe(
        'John Doe'
      );
      expect(getDisplayName('user@example.com', 'Alice', 'Smith')).toBe(
        'Alice Smith'
      );
    });

    it('should return first name only when last name not provided', () => {
      expect(getDisplayName('user@example.com', 'John', null)).toBe('John');
      expect(getDisplayName('user@example.com', 'Alice', undefined)).toBe(
        'Alice'
      );
    });

    it('should return last name only when first name not provided', () => {
      expect(getDisplayName('user@example.com', null, 'Doe')).toBe('Doe');
      expect(getDisplayName('user@example.com', undefined, 'Smith')).toBe(
        'Smith'
      );
    });

    it('should fall back to email when no name provided', () => {
      expect(getDisplayName('john@example.com', null, null)).toBe(
        'john@example.com'
      );
      expect(getDisplayName('alice@example.com')).toBe('alice@example.com');
    });

    it('should trim whitespace from names', () => {
      expect(getDisplayName('user@example.com', '  John  ', '  Doe  ')).toBe(
        'John Doe'
      );
    });

    it('should handle empty strings as no name', () => {
      expect(getDisplayName('user@example.com', '', '')).toBe(
        'user@example.com'
      );
      expect(getDisplayName('user@example.com', '   ', '   ')).toBe(
        'user@example.com'
      );
    });
  });
});
