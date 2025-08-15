import { describe, it, expect } from 'vitest'
import { createStore } from 'jotai'
import { currentSelectedFileAtom, selectedFilesAtom, currentFilePathAtom } from './ui-atoms'

describe('currentSelectedFileAtom', () => {
  it('sets selectedFiles and currentFilePath when given a path', () => {
    const store = createStore()
    store.set(currentSelectedFileAtom, '/tmp/file.txt')
    expect(store.get(selectedFilesAtom)).toEqual(['/tmp/file.txt'])
    expect(store.get(currentFilePathAtom)).toBe('/tmp/file.txt')
  })

  it('clears selectedFiles and currentFilePath when set to null', () => {
    const store = createStore()
    store.set(currentSelectedFileAtom, '/tmp/file.txt')
    store.set(currentSelectedFileAtom, null)
    expect(store.get(selectedFilesAtom)).toEqual([])
    expect(store.get(currentFilePathAtom)).toBeNull()
  })
})
