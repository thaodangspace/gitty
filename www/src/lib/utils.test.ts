import { describe, it, expect } from 'vitest'
import { cn } from './utils'

describe('cn', () => {
  it('merges class names and removes duplicates', () => {
    expect(cn('p-2', 'p-4', 'font-bold')).toBe('p-4 font-bold')
  })

  it('handles falsy values gracefully', () => {
    expect(cn('text-lg', undefined, 'text-center')).toBe('text-lg text-center')
  })
})
