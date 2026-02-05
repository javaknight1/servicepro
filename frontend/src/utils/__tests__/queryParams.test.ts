import { describe, it, expect } from 'vitest';
import {
  buildQueryParams,
  buildQueryString,
  appendQueryParams,
} from '../queryParams';

describe('queryParams utilities', () => {
  describe('buildQueryParams', () => {
    it('should build URLSearchParams from object with string values', () => {
      const params = buildQueryParams({
        name: 'John',
        city: 'New York',
      });

      expect(params.get('name')).toBe('John');
      expect(params.get('city')).toBe('New York');
    });

    it('should convert numbers to strings', () => {
      const params = buildQueryParams({
        page: 1,
        limit: 10,
      });

      expect(params.get('page')).toBe('1');
      expect(params.get('limit')).toBe('10');
    });

    it('should convert booleans to strings', () => {
      const params = buildQueryParams({
        active: true,
        archived: false,
      });

      expect(params.get('active')).toBe('true');
      expect(params.get('archived')).toBe('false');
    });

    it('should filter out undefined values', () => {
      const params = buildQueryParams({
        name: 'John',
        age: undefined,
      });

      expect(params.get('name')).toBe('John');
      expect(params.has('age')).toBe(false);
    });

    it('should filter out null values', () => {
      const params = buildQueryParams({
        name: 'John',
        age: null,
      });

      expect(params.get('name')).toBe('John');
      expect(params.has('age')).toBe(false);
    });

    it('should filter out empty string values', () => {
      const params = buildQueryParams({
        name: 'John',
        search: '',
      });

      expect(params.get('name')).toBe('John');
      expect(params.has('search')).toBe(false);
    });

    it('should handle mixed value types', () => {
      const params = buildQueryParams({
        name: 'John',
        page: 1,
        active: true,
        search: undefined,
        filter: null,
        query: '',
      });

      expect(params.toString()).toBe('name=John&page=1&active=true');
    });

    it('should handle empty object', () => {
      const params = buildQueryParams({});
      expect(params.toString()).toBe('');
    });

    it('should handle zero as valid value', () => {
      const params = buildQueryParams({
        offset: 0,
      });

      expect(params.get('offset')).toBe('0');
    });
  });

  describe('buildQueryString', () => {
    it('should return query string with leading ?', () => {
      const queryString = buildQueryString({
        page: 1,
        search: 'test',
      });

      expect(queryString).toBe('?page=1&search=test');
    });

    it('should return empty string when no params', () => {
      const queryString = buildQueryString({});
      expect(queryString).toBe('');
    });

    it('should return empty string when all params are filtered out', () => {
      const queryString = buildQueryString({
        empty: '',
        nothing: null,
        missing: undefined,
      });

      expect(queryString).toBe('');
    });

    it('should properly encode special characters', () => {
      const queryString = buildQueryString({
        search: 'hello world',
        filter: 'a&b',
      });

      expect(queryString).toContain('search=hello+world');
      expect(queryString).toContain('filter=a%26b');
    });
  });

  describe('appendQueryParams', () => {
    it('should append query params to base URL', () => {
      const url = appendQueryParams('/api/users', {
        page: 1,
        search: 'test',
      });

      expect(url).toBe('/api/users?page=1&search=test');
    });

    it('should return base URL when no params', () => {
      const url = appendQueryParams('/api/users', {});
      expect(url).toBe('/api/users');
    });

    it('should handle URL with existing path', () => {
      const url = appendQueryParams('/api/v1/users/list', {
        active: true,
      });

      expect(url).toBe('/api/v1/users/list?active=true');
    });

    it('should handle full URLs', () => {
      const url = appendQueryParams('https://example.com/api/users', {
        page: 1,
      });

      expect(url).toBe('https://example.com/api/users?page=1');
    });
  });
});
