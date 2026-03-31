import { atom } from 'jotai';
import { atomWithStorage } from 'jotai/utils';

// Persisted auth state in localStorage
export const authTokenAtom = atomWithStorage<string | null>('gitty_auth_token', null);
export const authDeviceIdAtom = atomWithStorage<string | null>('gitty_auth_device_id', null);

// Derived atom: true if user has a stored token
export const isAuthenticatedAtom = atom((get) => get(authTokenAtom) !== null);
