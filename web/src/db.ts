import { openDB } from "idb";
import type { EntryRecord, Settings, SyncState } from "./types";

const DB_NAME = "onepage-diary";
const DB_VERSION = 1;

const DEFAULT_SETTINGS: Settings = {
  serverBaseUrl: "",
  username: "",
  token: null,
  rememberCredentials: false,
  autoSyncEnabled: true
};

const DEFAULT_SYNC_STATE: SyncState = {
  lastSyncCursor: 0,
  lastSyncAt: null,
  lastError: null
};

const dbPromise = openDB(DB_NAME, DB_VERSION, {
  upgrade(db) {
    if (!db.objectStoreNames.contains("entries")) {
      db.createObjectStore("entries", { keyPath: "date" });
    }
    if (!db.objectStoreNames.contains("kv")) {
      db.createObjectStore("kv", { keyPath: "key" });
    }
  }
});

async function getKV<T>(key: string): Promise<T | null> {
  const db = await dbPromise;
  const record = await db.get("kv", key);
  return record ? (record.value as T) : null;
}

async function setKV<T>(key: string, value: T) {
  const db = await dbPromise;
  await db.put("kv", { key, value });
}

export async function getSettings(): Promise<Settings> {
  const stored = await getKV<Settings>("settings");
  return stored ? { ...DEFAULT_SETTINGS, ...stored } : { ...DEFAULT_SETTINGS };
}

export async function saveSettings(settings: Settings) {
  await setKV("settings", settings);
}

export async function getSyncState(): Promise<SyncState> {
  const stored = await getKV<SyncState>("syncState");
  return stored ? { ...DEFAULT_SYNC_STATE, ...stored } : { ...DEFAULT_SYNC_STATE };
}

export async function saveSyncState(state: SyncState) {
  await setKV("syncState", state);
}

export async function getEntry(date: string): Promise<EntryRecord | null> {
  const db = await dbPromise;
  const entry = await db.get("entries", date);
  return entry ?? null;
}

export async function putEntry(entry: EntryRecord) {
  const db = await dbPromise;
  await db.put("entries", entry);
}

export async function deleteEntry(date: string) {
  const db = await dbPromise;
  await db.delete("entries", date);
}

export async function listEntries(includeDeleted = false): Promise<EntryRecord[]> {
  const db = await dbPromise;
  const entries = await db.getAll("entries");
  const filtered = includeDeleted ? entries : entries.filter((entry) => !entry.deleted);
  return filtered.sort((a, b) => (a.date < b.date ? 1 : -1));
}

export async function listPendingEntries(): Promise<EntryRecord[]> {
  const db = await dbPromise;
  const entries = await db.getAll("entries");
  return entries.filter((entry) => entry.syncStatus === "pending");
}

export async function listConflictEntries(): Promise<EntryRecord[]> {
  const db = await dbPromise;
  const entries = await db.getAll("entries");
  return entries.filter((entry) => entry.syncStatus === "conflict");
}

export async function countPendingEntries(): Promise<number> {
  const db = await dbPromise;
  const entries = await db.getAll("entries");
  return entries.filter((entry) => entry.syncStatus === "pending").length;
}

export function createEmptyEntry(date: string): EntryRecord {
  const now = new Date().toISOString();
  return {
    date,
    content: "",
    createdAt: now,
    updatedAt: now,
    syncStatus: "synced",
    remoteRevision: null,
    conflictRemoteContent: null,
    conflictRemoteRevision: null,
    conflictRemoteDeleted: false,
    deleted: false
  };
}
