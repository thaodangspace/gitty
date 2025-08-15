import { describe, it, expect } from 'vitest'
import { createStore } from 'jotai'
import { repositoriesAtom, selectedRepositoryIdAtom, selectedRepositoryFromListAtom } from './repository-atoms'

describe('selectedRepositoryFromListAtom', () => {
  it('returns the repository matching the selected id', () => {
    const store = createStore()
    const repoA = { id: 'a', name: 'Repo A' } as any
    const repoB = { id: 'b', name: 'Repo B' } as any
    store.set(repositoriesAtom, [repoA, repoB])
    store.set(selectedRepositoryIdAtom, 'b')
    expect(store.get(selectedRepositoryFromListAtom)).toEqual(repoB)
  })

  it('returns null when no repository matches', () => {
    const store = createStore()
    store.set(repositoriesAtom, [])
    store.set(selectedRepositoryIdAtom, 'missing')
    expect(store.get(selectedRepositoryFromListAtom)).toBeNull()
  })
})
