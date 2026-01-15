import {
  createEmptyEntry,
  deleteEntry,
  getEntry,
  getSyncState,
  listPendingEntries,
  putEntry,
  saveSyncState
} from "./db";
import type { EntryRecord, Settings } from "./types";

interface ChangeEvent {
  id: number;
  date: string;
  action: "upsert" | "delete";
  revision: number;
  createdAt: string;
}

interface EntryResponse {
  date: string;
  content: string;
  revision: number;
  deleted: boolean;
  updatedAt: string;
  createdAt: string;
}

function normalizeBaseUrl(raw: string): string {
  return raw.replace(/\/+$/, "");
}

async function apiRequest<T>(baseUrl: string, token: string, path: string, init?: RequestInit) {
  const response = await fetch(`${baseUrl}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...(init?.headers ?? {})
    }
  });

  const text = await response.text();
  const payload = text ? (JSON.parse(text) as T) : ({} as T);

  return { response, payload };
}

async function fetchEntry(baseUrl: string, token: string, date: string): Promise<EntryResponse> {
  const { response, payload } = await apiRequest<EntryResponse>(baseUrl, token, `/api/v1/entries/${date}`);
  if (!response.ok) {
    throw new Error((payload as { error?: string }).error || "fetch_entry_failed");
  }
  return payload;
}

async function applyRemoteUpsert(
  baseUrl: string,
  token: string,
  event: ChangeEvent
): Promise<void> {
  const remote = await fetchEntry(baseUrl, token, event.date);
  const local = await getEntry(event.date);

  if (local && local.syncStatus === "pending") {
    local.syncStatus = "conflict";
    local.conflictRemoteContent = remote.content;
    local.conflictRemoteRevision = remote.revision;
    local.conflictRemoteDeleted = remote.deleted;
    await putEntry(local);
    return;
  }

  if (local && local.syncStatus === "conflict") {
    local.conflictRemoteContent = remote.content;
    local.conflictRemoteRevision = remote.revision;
    local.conflictRemoteDeleted = remote.deleted;
    await putEntry(local);
    return;
  }

  const merged: EntryRecord = local ?? createEmptyEntry(event.date);
  merged.content = remote.content;
  merged.deleted = remote.deleted;
  merged.remoteRevision = remote.revision;
  merged.syncStatus = "synced";
  merged.updatedAt = remote.updatedAt;
  merged.createdAt = merged.createdAt || remote.createdAt;
  merged.conflictRemoteContent = null;
  merged.conflictRemoteRevision = null;
  merged.conflictRemoteDeleted = false;

  await putEntry(merged);
}

async function applyRemoteDelete(event: ChangeEvent): Promise<void> {
  const local = await getEntry(event.date);
  if (!local) {
    return;
  }

  if (local.syncStatus === "pending") {
    if (local.deleted) {
      local.syncStatus = "synced";
      local.remoteRevision = event.revision;
      local.conflictRemoteContent = null;
      local.conflictRemoteRevision = null;
      local.conflictRemoteDeleted = false;
      await putEntry(local);
      return;
    }

    local.syncStatus = "conflict";
    local.conflictRemoteContent = null;
    local.conflictRemoteRevision = event.revision;
    local.conflictRemoteDeleted = true;
    await putEntry(local);
    return;
  }

  if (local.syncStatus === "conflict") {
    local.conflictRemoteDeleted = true;
    local.conflictRemoteRevision = event.revision;
    await putEntry(local);
    return;
  }

  local.deleted = true;
  local.remoteRevision = event.revision;
  local.syncStatus = "synced";
  local.updatedAt = new Date().toISOString();
  await putEntry(local);
}

async function pushEntry(
  baseUrl: string,
  token: string,
  entry: EntryRecord
): Promise<{ conflict: boolean; payload?: EntryResponse }> {
  if (entry.deleted) {
    if (entry.remoteRevision === null) {
      await deleteEntry(entry.date);
      return { conflict: false };
    }

    const { response, payload } = await apiRequest<EntryResponse>(
      baseUrl,
      token,
      `/api/v1/entries/${entry.date}?baseRevision=${entry.remoteRevision ?? 0}`,
      { method: "DELETE" }
    );

    if (response.status === 409) {
      return { conflict: true, payload };
    }

    if (!response.ok) {
      throw new Error((payload as { error?: string }).error || "delete_failed");
    }

    return { conflict: false, payload };
  }

  const { response, payload } = await apiRequest<EntryResponse>(
    baseUrl,
    token,
    `/api/v1/entries/${entry.date}`,
    {
      method: "PUT",
      body: JSON.stringify({
        content: entry.content,
        baseRevision: entry.remoteRevision ?? 0
      })
    }
  );

  if (response.status === 409) {
    return { conflict: true, payload };
  }

  if (!response.ok) {
    throw new Error((payload as { error?: string }).error || "push_failed");
  }

  return { conflict: false, payload };
}

export async function runSync(settings: Settings) {
  if (!settings.serverBaseUrl || !settings.token) {
    throw new Error("missing_server_or_token");
  }

  const baseUrl = normalizeBaseUrl(settings.serverBaseUrl);
  const syncState = await getSyncState();
  let cursor = syncState.lastSyncCursor;

  while (true) {
    const { response, payload } = await apiRequest<{ events: ChangeEvent[]; lastCursor: number }>(
      baseUrl,
      settings.token,
      `/api/v1/sync/changes?after=${cursor}`
    );

    if (!response.ok) {
      throw new Error((payload as { error?: string }).error || "sync_pull_failed");
    }

    const events = payload.events ?? [];
    for (const event of events) {
      if (event.action === "delete") {
        await applyRemoteDelete(event);
      } else {
        await applyRemoteUpsert(baseUrl, settings.token, event);
      }
    }

    cursor = payload.lastCursor ?? cursor;
    if (events.length === 0) {
      break;
    }
  }

  await saveSyncState({
    ...syncState,
    lastSyncCursor: cursor
  });

  const pending = await listPendingEntries();
  for (const entry of pending) {
    const result = await pushEntry(baseUrl, settings.token, entry);
    if (result.conflict) {
      entry.syncStatus = "conflict";
      entry.conflictRemoteContent = result.payload?.content ?? null;
      entry.conflictRemoteRevision = result.payload?.revision ?? null;
      entry.conflictRemoteDeleted = result.payload?.deleted ?? false;
      await putEntry(entry);
      continue;
    }

    if (result.payload) {
      entry.remoteRevision = result.payload.revision;
      entry.updatedAt = result.payload.updatedAt;
      entry.createdAt = entry.createdAt || result.payload.createdAt;
    }
    entry.syncStatus = "synced";
    entry.conflictRemoteContent = null;
    entry.conflictRemoteRevision = null;
    entry.conflictRemoteDeleted = false;
    await putEntry(entry);
  }

  await saveSyncState({
    ...syncState,
    lastSyncCursor: cursor,
    lastSyncAt: new Date().toISOString(),
    lastError: null
  });
}
