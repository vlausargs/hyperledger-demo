import { describe, it, expect } from 'vitest';
import { cn } from './utils';

describe('cn (class name utility)', () => {
  it('joins multiple class names', () => {
    expect(cn('a', 'b', 'c')).toBe('a b c');
  });

  it('filters out falsy values', () => {
    expect(cn('a', false, null, undefined, '', 'b')).toBe('a b');
  });

  it('handles conditional class objects', () => {
    expect(cn('base', { active: true, disabled: false })).toBe('base active');
  });

  it('merges tailwind classes deduping conflicting utilities', () => {
    // tailwind-merge collapses conflicting classes — last one wins.
    expect(cn('p-2', 'p-4')).toBe('p-4');
    expect(cn('text-red-500', 'text-blue-500')).toBe('text-blue-500');
  });

  it('preserves non-conflicting tailwind classes', () => {
    expect(cn('px-2', 'py-4')).toBe('px-2 py-4');
  });

  it('returns empty string for no arguments', () => {
    expect(cn()).toBe('');
  });
});
