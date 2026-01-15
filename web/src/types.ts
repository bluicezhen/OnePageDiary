export type SyncStatus = "synced" | "pending" | "conflict" | "error";

export interface EntryRecord {
  date: string;
  content: string;
  createdAt: string;
  updatedAt: string;
  syncStatus: SyncStatus;
  remoteRevision: number | null;
  conflictRemoteContent: string | null;
  conflictRemoteRevision: number | null;
  conflictRemoteDeleted: boolean;
  deleted: boolean;
}

export interface Settings {
  serverBaseUrl: string;
  username: string;
  token: string | null;
  rememberCredentials: boolean;
  autoSyncEnabled: boolean;
}

export interface SyncState {
  lastSyncCursor: number;
  lastSyncAt: string | null;
  lastError: string | null;
}
