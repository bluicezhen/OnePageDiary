<template>
  <div class="app">
    <header class="app-bar">
      <div class="app-bar__left">
        <button class="icon-btn" @click="toggleSidebar" aria-label="toggle calendar">
          ☰
        </button>
        <div class="title">{{ headerTitle }}</div>
      </div>
      <div class="app-bar__right">
        <button class="ghost-btn" @click="openView('history')">历史</button>
        <button class="ghost-btn" @click="openView('search')">搜索</button>
        <button class="ghost-btn" @click="openView('settings')">设置</button>
        <button class="icon-btn" @click="toggleSyncPanel" aria-label="sync status">
          <span :class="['status-dot', syncDotClass]"></span>
        </button>
        <button class="icon-btn" @click="openCalendarDrawer" aria-label="calendar">
          日历
        </button>
      </div>
    </header>

    <div class="layout">
      <aside class="sidebar" :class="{ collapsed: isSidebarCollapsed }">
        <div class="sidebar-header">
          <div class="month-title">{{ calendarTitle }}</div>
          <div class="month-actions">
            <button class="icon-btn" @click="prevMonth">‹</button>
            <button class="icon-btn" @click="nextMonth">›</button>
          </div>
        </div>
        <div class="calendar-weekdays">
          <div v-for="label in weekdayLabels" :key="label" class="calendar-weekday">
            {{ label }}
          </div>
        </div>
        <div class="calendar-grid">
          <div v-for="(week, wIdx) in calendarWeeks" :key="wIdx" class="calendar-week">
            <button
              v-for="(day, dIdx) in week"
              :key="day.date ?? `empty-${wIdx}-${dIdx}`"
              class="calendar-day"
              :class="calendarDayClass(day.date)"
              :disabled="!day.date"
              @click="day.date && openEntry(day.date)"
            >
              <span>{{ day.dayNumber ?? "" }}</span>
              <span v-if="day.date && entryMarkers[day.date]" class="calendar-dot"></span>
            </button>
          </div>
        </div>
      </aside>

      <main class="main">
        <section v-if="view === 'editor'" class="panel editor-panel">
          <div class="panel-header">
            <div class="panel-title">{{ formatDateDisplay(selectedDate) }}</div>
            <div class="panel-status">{{ entryStatusLabel }}</div>
            <div class="panel-actions">
              <button class="ghost-btn" @click="toggleEditorMode">{{ editorModeLabel }}</button>
              <button class="ghost-btn" @click="deleteCurrentEntry" :disabled="!currentEntry">
                删除
              </button>
            </div>
          </div>
          <div v-if="editorMode === 'edit'" class="editor-body">
            <textarea
              v-model="editorContent"
              class="editor-textarea"
              placeholder="今天写点什么..."
            ></textarea>
          </div>
          <div v-else class="preview-body" v-html="previewHtml"></div>
        </section>

        <section v-if="view === 'history'" class="panel">
          <div class="panel-header">
            <div class="panel-title">历史日记</div>
          </div>
          <div class="list">
            <button v-for="entry in entries" :key="entry.date" class="entry-card" @click="openEntry(entry.date)">
              <div class="entry-card__header">
                <div class="entry-date">{{ formatDateDisplay(entry.date) }}</div>
                <div class="entry-status">{{ statusLabel(entry) }}</div>
              </div>
              <div class="entry-preview">{{ previewText(entry.content) }}</div>
              <div class="entry-meta">{{ formatTimeDisplay(entry.updatedAt) }}</div>
            </button>
            <div v-if="entries.length === 0" class="empty">暂无记录</div>
          </div>
        </section>

        <section v-if="view === 'search'" class="panel">
          <div class="panel-header">
            <div class="panel-title">搜索</div>
          </div>
          <input v-model="searchQuery" class="search-input" placeholder="输入关键词" />
          <div class="list">
            <button
              v-for="entry in searchResults"
              :key="entry.date"
              class="entry-card"
              @click="openEntry(entry.date)"
            >
              <div class="entry-card__header">
                <div class="entry-date">{{ formatDateDisplay(entry.date) }}</div>
                <div class="entry-status">{{ statusLabel(entry) }}</div>
              </div>
              <div class="entry-preview">{{ previewText(entry.content) }}</div>
            </button>
            <div v-if="searchQuery && searchResults.length === 0" class="empty">未找到匹配内容</div>
          </div>
        </section>

        <section v-if="view === 'settings'" class="panel">
          <div class="panel-header">
            <div class="panel-title">设置</div>
          </div>
          <div class="form">
            <label class="form-label">服务端地址</label>
            <input v-model="settings.serverBaseUrl" class="text-input" placeholder="http://127.0.0.1:8080" />

            <label class="form-label">用户名</label>
            <input v-model="settings.username" class="text-input" placeholder="用户名" />

            <label class="form-label">密码</label>
            <input v-model="loginPassword" class="text-input" type="password" placeholder="密码" />

            <div class="form-row">
              <label class="checkbox">
                <input v-model="settings.rememberCredentials" type="checkbox" />
                记住登录
              </label>
            </div>

            <div class="form-actions">
              <button class="primary-btn" @click="login">登录</button>
              <button class="ghost-btn" @click="logout">登出</button>
            </div>

            <div class="form-row">
              <label class="checkbox">
                <input v-model="settings.autoSyncEnabled" type="checkbox" />
                前台自动同步
              </label>
            </div>

            <div class="form-actions">
              <button class="ghost-btn" @click="exportData">导出</button>
              <label class="ghost-btn file-btn">
                导入
                <input type="file" accept="application/json" @change="importData" />
              </label>
            </div>

            <button class="danger-btn" @click="clearLocalData">清除本地数据</button>
          </div>
        </section>

        <section v-if="view === 'conflict'" class="panel conflict-panel">
          <div class="panel-header">
            <div class="panel-title">冲突处理</div>
            <div class="panel-status">
              {{ conflictEntry ? formatDateDisplay(conflictEntry.date) : "" }}
            </div>
          </div>
          <div v-if="conflictEntry" class="conflict-body">
            <div class="conflict-column">
              <div class="conflict-title">本地版本</div>
              <textarea v-model="editorContent" class="editor-textarea"></textarea>
            </div>
            <div class="conflict-column">
              <div class="conflict-title">远端版本</div>
              <div v-if="conflictEntry.conflictRemoteDeleted" class="conflict-remote">远端已删除</div>
              <pre v-else class="conflict-remote">{{ conflictEntry.conflictRemoteContent }}</pre>
            </div>
          </div>
          <div class="conflict-actions">
            <button class="danger-btn" @click="resolveConflictLocal">用本地覆盖远端</button>
            <button class="ghost-btn" @click="resolveConflictRemote">用远端覆盖本地</button>
          </div>
        </section>
      </main>
    </div>

    <div v-if="showSyncPanel" class="sync-panel">
      <div class="sync-row">
        <span>网络</span>
        <span>{{ syncStatus.online ? "在线" : "离线" }}</span>
      </div>
      <div class="sync-row">
        <span>服务端</span>
        <span>{{ syncStatus.serverReachable ? "可达" : "不可达" }}</span>
      </div>
      <div class="sync-row">
        <span>待同步</span>
        <span>{{ pendingCount }} 天</span>
      </div>
      <div class="sync-row">
        <span>上次同步</span>
        <span>{{ formatTimeDisplay(syncStatus.lastSyncAt) }}</span>
      </div>
      <div class="sync-row" v-if="syncStatus.lastError">
        <span>错误</span>
        <span>{{ syncStatus.lastError }}</span>
      </div>
      <div class="sync-actions">
        <button class="primary-btn" @click="triggerSync" :disabled="syncStatus.syncing">
          {{ syncStatus.syncing ? "同步中" : "立即同步" }}
        </button>
        <button class="ghost-btn" @click="openConflictList" :disabled="conflictCount === 0">
          查看冲突 ({{ conflictCount }})
        </button>
      </div>
    </div>

    <div v-if="calendarDrawerOpen" class="drawer-overlay" @click="closeCalendarDrawer">
      <div class="drawer" @click.stop>
        <div class="sidebar-header">
          <div class="month-title">{{ calendarTitle }}</div>
          <div class="month-actions">
            <button class="icon-btn" @click="prevMonth">‹</button>
            <button class="icon-btn" @click="nextMonth">›</button>
          </div>
        </div>
        <div class="calendar-weekdays">
          <div v-for="label in weekdayLabels" :key="label" class="calendar-weekday">
            {{ label }}
          </div>
        </div>
        <div class="calendar-grid">
          <div v-for="(week, wIdx) in calendarWeeks" :key="`drawer-${wIdx}`" class="calendar-week">
            <button
              v-for="(day, dIdx) in week"
              :key="day.date ?? `drawer-empty-${wIdx}-${dIdx}`"
              class="calendar-day"
              :class="calendarDayClass(day.date)"
              :disabled="!day.date"
              @click="day.date && openEntry(day.date)"
            >
              <span>{{ day.dayNumber ?? "" }}</span>
              <span v-if="day.date && entryMarkers[day.date]" class="calendar-dot"></span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { marked } from "marked";
import {
  countPendingEntries,
  createEmptyEntry,
  deleteEntry,
  getEntry,
  getSettings,
  getSyncState,
  listEntries,
  listConflictEntries,
  putEntry,
  saveSettings,
  saveSyncState
} from "./db";
import { runSync } from "./sync";
import type { EntryRecord, Settings } from "./types";
import { buildMonthCalendar } from "./utils/calendar";
import { formatDateDisplay, formatTimeDisplay, todayString } from "./utils/date";

marked.setOptions({ breaks: true });

const view = ref<"editor" | "history" | "search" | "settings" | "conflict">("editor");
const editorMode = ref<"edit" | "preview">("edit");
const selectedDate = ref(todayString());
const currentEntry = ref<EntryRecord | null>(null);
const entries = ref<EntryRecord[]>([]);
const conflictEntry = ref<EntryRecord | null>(null);
const searchQuery = ref("");
const editorContent = ref("");
const calendarCursor = ref(new Date());
const isSidebarCollapsed = ref(false);
const calendarDrawerOpen = ref(false);
const showSyncPanel = ref(false);
const loginPassword = ref("");
const sessionToken = ref<string | null>(null);

const settings = reactive<Settings>({
  serverBaseUrl: "",
  username: "",
  token: null,
  rememberCredentials: false,
  autoSyncEnabled: true
});

const syncStatus = reactive({
  online: navigator.onLine,
  serverReachable: false,
  syncing: false,
  lastSyncAt: "",
  lastError: ""
});

const weekdayLabels = ["日", "一", "二", "三", "四", "五", "六"];

const calendarWeeks = computed(() => buildMonthCalendar(calendarCursor.value));
const calendarTitle = computed(() =>
  calendarCursor.value.toLocaleDateString("zh-CN", { year: "numeric", month: "long" })
);

const headerTitle = computed(() => {
  if (view.value === "editor") {
    return selectedDate.value === todayString() ? "今日" : selectedDate.value;
  }
  if (view.value === "history") {
    return "历史";
  }
  if (view.value === "search") {
    return "搜索";
  }
  if (view.value === "settings") {
    return "设置";
  }
  return "冲突";
});

const previewHtml = computed(() => marked.parse(editorContent.value || ""));

const entryMarkers = computed<Record<string, boolean>>(() => {
  const markers: Record<string, boolean> = {};
  for (const entry of entries.value) {
    markers[entry.date] = true;
  }
  return markers;
});

const searchResults = computed(() => {
  if (!searchQuery.value) {
    return [];
  }
  const keyword = searchQuery.value.toLowerCase();
  return entries.value.filter((entry) => entry.content.toLowerCase().includes(keyword));
});

const pendingCount = ref(0);
const conflictCount = ref(0);

const entryStatusLabel = computed(() => {
  if (!currentEntry.value) {
    return "";
  }
  if (currentEntry.value.syncStatus === "conflict") {
    return "存在冲突";
  }
  if (currentEntry.value.syncStatus === "pending") {
    return syncStatus.online ? "待同步" : "已保存（本地）";
  }
  if (currentEntry.value.syncStatus === "error") {
    return "同步失败";
  }
  return "已同步";
});

const editorModeLabel = computed(() => (editorMode.value === "edit" ? "预览" : "编辑"));

const syncDotClass = computed(() => {
  if (conflictCount.value > 0) {
    return "status-dot--conflict";
  }
  if (pendingCount.value > 0) {
    return "status-dot--pending";
  }
  if (!syncStatus.online) {
    return "status-dot--offline";
  }
  return syncStatus.syncing ? "status-dot--syncing" : "status-dot--ok";
});

let saveTimer: number | null = null;

function toggleSidebar() {
  isSidebarCollapsed.value = !isSidebarCollapsed.value;
}

function openCalendarDrawer() {
  calendarDrawerOpen.value = true;
}

function closeCalendarDrawer() {
  calendarDrawerOpen.value = false;
}

function toggleSyncPanel() {
  showSyncPanel.value = !showSyncPanel.value;
}

function openView(target: typeof view.value) {
  view.value = target;
}

function openEntry(date: string) {
  selectedDate.value = date;
  view.value = "editor";
  calendarDrawerOpen.value = false;
  loadEntry(date);
}

function toggleEditorMode() {
  editorMode.value = editorMode.value === "edit" ? "preview" : "edit";
}

function calendarDayClass(date: string | null) {
  if (!date) {
    return "calendar-day--empty";
  }
  if (date === selectedDate.value) {
    return "calendar-day--selected";
  }
  if (date === todayString()) {
    return "calendar-day--today";
  }
  return "";
}

function prevMonth() {
  const d = new Date(calendarCursor.value);
  d.setMonth(d.getMonth() - 1);
  calendarCursor.value = d;
}

function nextMonth() {
  const d = new Date(calendarCursor.value);
  d.setMonth(d.getMonth() + 1);
  calendarCursor.value = d;
}

function previewText(text: string) {
  const normalized = text.replace(/\s+/g, " ").trim();
  return normalized.length > 80 ? `${normalized.slice(0, 80)}...` : normalized || "（空）";
}

function statusLabel(entry: EntryRecord) {
  if (entry.syncStatus === "pending") {
    return "待同步";
  }
  if (entry.syncStatus === "conflict") {
    return "冲突";
  }
  if (entry.syncStatus === "error") {
    return "错误";
  }
  return "已同步";
}

async function loadEntry(date: string) {
  const entry = await getEntry(date);
  if (entry) {
    const visibleEntry = entry.deleted ? { ...entry, content: "" } : entry;
    currentEntry.value = visibleEntry;
    editorContent.value = visibleEntry.content;
    return;
  }

  const empty = createEmptyEntry(date);
  await putEntry(empty);
  currentEntry.value = empty;
  editorContent.value = empty.content;
  await refreshEntries();
}

async function refreshEntries() {
  entries.value = await listEntries();
  await refreshCounts();
}

async function refreshCounts() {
  pendingCount.value = await countPendingEntries();
  conflictCount.value = (await listConflictEntries()).length;
}

function scheduleSave() {
  if (saveTimer) {
    window.clearTimeout(saveTimer);
  }
  saveTimer = window.setTimeout(() => {
    saveCurrentEntry();
  }, 400);
}

async function saveCurrentEntry() {
  if (!currentEntry.value) {
    return;
  }
  if (editorContent.value === currentEntry.value.content) {
    return;
  }
  const entry = { ...currentEntry.value };
  entry.content = editorContent.value;
  entry.updatedAt = new Date().toISOString();

  const trimmed = entry.content.trim();
  if (entry.deleted && trimmed.length === 0) {
    await putEntry(entry);
    currentEntry.value = entry;
    return;
  }

  if (entry.deleted && trimmed.length > 0) {
    entry.deleted = false;
  }

  if (entry.syncStatus !== "conflict") {
    if (entry.remoteRevision === null && trimmed.length === 0) {
      entry.syncStatus = "synced";
    } else {
      entry.syncStatus = "pending";
    }
  }

  await putEntry(entry);
  currentEntry.value = entry;
  await refreshEntries();
}

async function deleteCurrentEntry() {
  if (!currentEntry.value) {
    return;
  }
  if (!confirm("确定删除这一天的日记？")) {
    return;
  }

  if (currentEntry.value.remoteRevision === null) {
    await deleteEntry(currentEntry.value.date);
  } else {
    const entry = { ...currentEntry.value };
    entry.deleted = true;
    entry.syncStatus = "pending";
    await putEntry(entry);
  }

  currentEntry.value = null;
  editorContent.value = "";
  await refreshEntries();
}

function activeToken() {
  return settings.rememberCredentials ? settings.token : sessionToken.value;
}

async function login() {
  if (!settings.serverBaseUrl) {
    syncStatus.lastError = "请先填写服务端地址";
    return;
  }
  if (!settings.username || !loginPassword.value) {
    syncStatus.lastError = "请输入用户名和密码";
    return;
  }

  try {
    const response = await fetch(`${settings.serverBaseUrl.replace(/\/+$/, "")}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: settings.username, password: loginPassword.value })
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || "login_failed");
    }
    const token = payload.token as string;
    sessionToken.value = token;
    if (settings.rememberCredentials) {
      settings.token = token;
    } else {
      settings.token = null;
    }
    await saveSettings({ ...settings });
    syncStatus.lastError = "";
  } catch (err) {
    syncStatus.lastError = err instanceof Error ? err.message : "login_failed";
  }
}

async function logout() {
  sessionToken.value = null;
  settings.token = null;
  await saveSettings({ ...settings });
}

async function checkServerReachable() {
  if (!settings.serverBaseUrl) {
    syncStatus.serverReachable = false;
    return;
  }
  try {
    const response = await fetch(`${settings.serverBaseUrl.replace(/\/+$/, "")}/api/v1/health`);
    syncStatus.serverReachable = response.ok;
  } catch {
    syncStatus.serverReachable = false;
  }
}

async function triggerSync() {
  if (syncStatus.syncing) {
    return;
  }
  if (!settings.serverBaseUrl || !activeToken()) {
    syncStatus.lastError = "请先登录并配置服务端";
    return;
  }

  syncStatus.syncing = true;
  syncStatus.lastError = "";
  try {
    await runSync({ ...settings, token: activeToken() });
    const state = await getSyncState();
    syncStatus.lastSyncAt = state.lastSyncAt ?? "";
    syncStatus.lastError = state.lastError ?? "";
    syncStatus.serverReachable = true;
  } catch (err) {
    syncStatus.lastError = err instanceof Error ? err.message : "sync_failed";
    syncStatus.serverReachable = false;
    const state = await getSyncState();
    await saveSyncState({ ...state, lastError: syncStatus.lastError });
  } finally {
    syncStatus.syncing = false;
    await refreshEntries();
  }
}

async function openConflictList() {
  const conflicts = await listConflictEntries();
  if (conflicts.length === 0) {
    return;
  }
  conflictEntry.value = conflicts[0];
  currentEntry.value = conflicts[0];
  editorContent.value = conflicts[0].content;
  selectedDate.value = conflicts[0].date;
  view.value = "conflict";
}

async function resolveConflictLocal() {
  if (!conflictEntry.value || !activeToken() || !settings.serverBaseUrl) {
    return;
  }

  try {
    const baseUrl = settings.serverBaseUrl.replace(/\/+$/, "");
    const response = await fetch(
      `${baseUrl}/api/v1/entries/${conflictEntry.value.date}?force=1`,
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${activeToken()}`
        },
        body: JSON.stringify({ content: editorContent.value, baseRevision: conflictEntry.value.remoteRevision ?? 0 })
      }
    );
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || "conflict_push_failed");
    }

    const entry = { ...conflictEntry.value };
    entry.content = editorContent.value;
    entry.syncStatus = "synced";
    entry.deleted = false;
    entry.remoteRevision = payload.revision;
    entry.conflictRemoteContent = null;
    entry.conflictRemoteRevision = null;
    entry.conflictRemoteDeleted = false;
    await putEntry(entry);
    conflictEntry.value = null;
    currentEntry.value = entry;
    view.value = "editor";
    await refreshEntries();
  } catch (err) {
    syncStatus.lastError = err instanceof Error ? err.message : "conflict_push_failed";
  }
}

async function resolveConflictRemote() {
  if (!conflictEntry.value) {
    return;
  }

  if (conflictEntry.value.conflictRemoteDeleted) {
    await deleteEntry(conflictEntry.value.date);
    currentEntry.value = null;
  } else {
    const entry = { ...conflictEntry.value };
    entry.content = conflictEntry.value.conflictRemoteContent ?? "";
    entry.syncStatus = "synced";
    entry.deleted = false;
    entry.remoteRevision = conflictEntry.value.conflictRemoteRevision ?? entry.remoteRevision;
    entry.conflictRemoteContent = null;
    entry.conflictRemoteRevision = null;
    entry.conflictRemoteDeleted = false;
    await putEntry(entry);
    currentEntry.value = entry;
  }

  conflictEntry.value = null;
  view.value = "editor";
  await refreshEntries();
}

async function exportData() {
  const data = await listEntries(true);
  const payload = {
    schemaVersion: 1,
    exportedAt: new Date().toISOString(),
    entries: data.filter((entry) => !entry.deleted).map((entry) => ({
      date: entry.date,
      content: entry.content,
      createdAt: entry.createdAt,
      updatedAt: entry.updatedAt
    }))
  };

  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `onepage-diary-${todayString()}.json`;
  link.click();
  URL.revokeObjectURL(url);
}

async function importData(event: Event) {
  const input = event.target as HTMLInputElement;
  if (!input.files || input.files.length === 0) {
    return;
  }
  const file = input.files[0];
  const text = await file.text();
  const payload = JSON.parse(text) as {
    schemaVersion: number;
    entries: { date: string; content: string; createdAt: string; updatedAt: string }[];
  };

  if (payload.schemaVersion !== 1) {
    alert("不支持的备份版本");
    return;
  }

  for (const entry of payload.entries) {
    const record: EntryRecord = {
      date: entry.date,
      content: entry.content,
      createdAt: entry.createdAt,
      updatedAt: entry.updatedAt,
      syncStatus: "pending",
      remoteRevision: null,
      conflictRemoteContent: null,
      conflictRemoteRevision: null,
      conflictRemoteDeleted: false,
      deleted: false
    };
    await putEntry(record);
  }

  await refreshEntries();
}

async function clearLocalData() {
  if (!confirm("将清除全部本地数据，是否继续？")) {
    return;
  }
  const data = await listEntries(true);
  for (const entry of data) {
    await deleteEntry(entry.date);
  }
  await refreshEntries();
  currentEntry.value = null;
  editorContent.value = "";
}

watch(editorContent, () => {
  scheduleSave();
});

watch(
  () => settings,
  () => {
    saveSettings({ ...settings });
  },
  { deep: true }
);

onMounted(async () => {
  const storedSettings = await getSettings();
  Object.assign(settings, storedSettings);
  if (!settings.serverBaseUrl) {
    settings.serverBaseUrl = window.location.origin;
  }
  if (!settings.rememberCredentials && settings.token) {
    sessionToken.value = settings.token;
    settings.token = null;
    await saveSettings({ ...settings });
  }

  const state = await getSyncState();
  syncStatus.lastSyncAt = state.lastSyncAt ?? "";
  syncStatus.lastError = state.lastError ?? "";

  await loadEntry(selectedDate.value);
  await refreshEntries();
  await checkServerReachable();

  if (settings.autoSyncEnabled && navigator.onLine) {
    await triggerSync();
  }

  window.addEventListener("online", () => {
    syncStatus.online = true;
  });
  window.addEventListener("offline", () => {
    syncStatus.online = false;
  });
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible" && settings.autoSyncEnabled) {
      triggerSync();
    }
  });
});
</script>

<style scoped>
</style>
