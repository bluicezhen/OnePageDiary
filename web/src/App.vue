<template>
  <div class="app">
    <header class="app-bar">
      <div class="app-bar__left">
        <button class="icon-btn" @click="toggleCalendar" :aria-label="t('calendarToggle')">
          ☰
        </button>
        <div class="title">{{ headerTitle }}</div>
      </div>
      <div class="app-bar__right">
        <button class="ghost-btn" @click="openView('history')">{{ t("navHistory") }}</button>
        <button class="ghost-btn" @click="openView('settings')">{{ t("navSettings") }}</button>
        <button class="icon-btn" @click="toggleSyncPanel" :aria-label="t('syncStatusLabel')">
          <span :class="['status-dot', syncDotClass]"></span>
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
            <div class="panel-title">{{ formatDateDisplay(selectedDate, localeTag) }}</div>
            <div class="panel-status">{{ entryStatusLabel }}</div>
            <div class="panel-actions">
              <button class="ghost-btn" @click="toggleEditorMode">{{ editorModeLabel }}</button>
              <button class="ghost-btn" @click="deleteCurrentEntry" :disabled="!currentEntry">
                {{ t("editorDelete") }}
              </button>
            </div>
          </div>
          <div v-if="editorMode === 'edit'" class="editor-body">
            <textarea
              v-model="editorContent"
              class="editor-textarea"
              :placeholder="t('editorPlaceholder')"
            ></textarea>
          </div>
          <div v-else class="preview-body" v-html="previewHtml"></div>
        </section>

        <section v-if="view === 'history'" class="panel">
          <div class="panel-header">
            <div class="panel-title">{{ t("historyTitle") }}</div>
          </div>
          <input v-model="searchQuery" class="search-input" :placeholder="t('historySearchPlaceholder')" />
          <div class="list">
            <button
              v-for="entry in historyList"
              :key="entry.date"
              class="entry-card"
              @click="openEntry(entry.date)"
            >
              <div class="entry-card__header">
                <div class="entry-date">{{ formatDateDisplay(entry.date, localeTag) }}</div>
                <div class="entry-status">{{ statusLabel(entry) }}</div>
              </div>
              <div class="entry-preview">{{ previewText(entry.content) }}</div>
              <div class="entry-meta">{{ formatTimeDisplay(entry.updatedAt, localeTag) }}</div>
            </button>
            <div v-if="historyList.length === 0" class="empty">
              {{ searchQuery ? t("historyNoMatch") : t("historyEmpty") }}
            </div>
          </div>
        </section>

        <section v-if="view === 'settings'" class="panel">
          <div class="panel-header">
            <div class="panel-title">{{ t("titleSettings") }}</div>
          </div>
          <div class="form">
            <label class="form-label">{{ t("settingsServerUrl") }}</label>
            <input
              v-model="settings.serverBaseUrl"
              class="text-input"
              placeholder="http://127.0.0.1:8080"
              :disabled="isLoggedIn"
            />

            <label class="form-label">{{ t("settingsLanguage") }}</label>
            <select v-model="settings.language" class="text-input">
              <option value="auto">{{ t("settingsLanguageAuto") }}</option>
              <option value="zh">{{ t("settingsLanguageZh") }}</option>
              <option value="en">{{ t("settingsLanguageEn") }}</option>
            </select>

            <template v-if="!isLoggedIn">
              <label class="form-label">{{ t("settingsUsername") }}</label>
              <input v-model="settings.username" class="text-input" :placeholder="t('settingsUsername')" />

              <label class="form-label">{{ t("settingsPassword") }}</label>
              <input
                v-model="loginPassword"
                class="text-input"
                type="password"
                :placeholder="t('settingsPassword')"
              />

              <div class="form-row">
                <label class="checkbox">
                  <input v-model="settings.rememberCredentials" type="checkbox" />
                  {{ t("settingsRemember") }}
                </label>
              </div>

              <div class="form-actions">
                <button class="primary-btn" @click="login">{{ t("settingsLogin") }}</button>
              </div>
            </template>

            <template v-else>
              <div class="login-status">{{ loggedInLabel }}</div>
              <div class="form-actions">
                <button class="ghost-btn" @click="logout">{{ t("settingsLogout") }}</button>
              </div>
            </template>

            <div
              v-if="authMessage"
              :class="['form-hint', authMessageTone === 'error' ? 'form-hint--error' : 'form-hint--success']"
            >
              {{ authMessage }}
            </div>

            <div class="form-row">
              <label class="checkbox">
                <input v-model="settings.autoSyncEnabled" type="checkbox" />
                {{ t("settingsAutoSync") }}
              </label>
            </div>

            <div class="form-actions">
              <button class="ghost-btn" @click="exportData">{{ t("settingsExport") }}</button>
              <label class="ghost-btn file-btn">
                {{ t("settingsImport") }}
                <input type="file" accept="application/json" @change="importData" />
              </label>
            </div>

            <button class="danger-btn" @click="clearLocalData">{{ t("settingsClear") }}</button>
          </div>
        </section>

        <section v-if="view === 'conflict'" class="panel conflict-panel">
          <div class="panel-header">
            <div class="panel-title">{{ t("conflictTitle") }}</div>
            <div class="panel-status">
              {{ conflictEntry ? formatDateDisplay(conflictEntry.date, localeTag) : "" }}
            </div>
          </div>
          <div v-if="conflictEntry" class="conflict-body">
            <div class="conflict-column">
              <div class="conflict-title">{{ t("conflictLocal") }}</div>
              <textarea v-model="editorContent" class="editor-textarea"></textarea>
            </div>
            <div class="conflict-column">
              <div class="conflict-title">{{ t("conflictRemote") }}</div>
              <div v-if="conflictEntry.conflictRemoteDeleted" class="conflict-remote">
                {{ t("conflictRemoteDeleted") }}
              </div>
              <pre v-else class="conflict-remote">{{ conflictEntry.conflictRemoteContent }}</pre>
            </div>
          </div>
          <div class="conflict-actions">
            <button class="danger-btn" @click="resolveConflictLocal">{{ t("conflictUseLocal") }}</button>
            <button class="ghost-btn" @click="resolveConflictRemote">{{ t("conflictUseRemote") }}</button>
          </div>
        </section>
      </main>
    </div>

    <div v-if="showSyncPanel" class="sync-panel">
      <div class="sync-row">
        <span>{{ t("syncNetwork") }}</span>
        <span>{{ syncStatus.online ? t("syncOnline") : t("syncOffline") }}</span>
      </div>
      <div class="sync-row">
        <span>{{ t("syncServer") }}</span>
        <span>{{ syncStatus.serverReachable ? t("syncReachable") : t("syncUnreachable") }}</span>
      </div>
      <div class="sync-row">
        <span>{{ t("syncPending") }}</span>
        <span>{{ pendingDaysLabel }}</span>
      </div>
      <div class="sync-row">
        <span>{{ t("syncLast") }}</span>
        <span>{{ formatTimeDisplay(syncStatus.lastSyncAt, localeTag) }}</span>
      </div>
      <div class="sync-row" v-if="syncStatus.lastError">
        <span>{{ t("syncError") }}</span>
        <span>{{ syncStatus.lastError }}</span>
      </div>
      <div class="sync-actions">
        <button class="primary-btn" @click="triggerSync" :disabled="syncStatus.syncing">
          {{ syncStatus.syncing ? t("syncSyncing") : t("syncNow") }}
        </button>
        <button class="ghost-btn" @click="openConflictList" :disabled="conflictCount === 0">
          {{ conflictButtonLabel }}
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

const translations = {
  zh: {
    navHistory: "历史",
    navSettings: "设置",
    titleToday: "今日",
    titleHistory: "历史",
    titleSettings: "设置",
    titleConflict: "冲突",
    editorPreview: "预览",
    editorEdit: "编辑",
    editorDelete: "删除",
    editorPlaceholder: "今天写点什么...",
    historyTitle: "历史日记",
    historySearchPlaceholder: "搜索历史日记",
    historyEmpty: "暂无记录",
    historyNoMatch: "未找到匹配内容",
    settingsServerUrl: "服务端地址",
    settingsUsername: "用户名",
    settingsPassword: "密码",
    settingsRemember: "记住登录",
    settingsLogin: "登录",
    settingsLogout: "登出",
    settingsLoggedInAs: "已登录：",
    settingsUnknownUser: "未知用户",
    settingsAutoSync: "前台自动同步",
    settingsExport: "导出",
    settingsImport: "导入",
    settingsClear: "清除本地数据",
    settingsLanguage: "语言",
    settingsLanguageAuto: "跟随浏览器",
    settingsLanguageZh: "中文",
    settingsLanguageEn: "English",
    conflictTitle: "冲突处理",
    conflictLocal: "本地版本",
    conflictRemote: "远端版本",
    conflictRemoteDeleted: "远端已删除",
    conflictUseLocal: "用本地覆盖远端",
    conflictUseRemote: "用远端覆盖本地",
    syncNetwork: "网络",
    syncOnline: "在线",
    syncOffline: "离线",
    syncServer: "服务端",
    syncReachable: "可达",
    syncUnreachable: "不可达",
    syncPending: "待同步",
    syncLast: "上次同步",
    syncError: "错误",
    syncSyncing: "同步中",
    syncNow: "立即同步",
    syncConflicts: "查看冲突",
    statusConflict: "存在冲突",
    statusPending: "待同步",
    statusPendingOffline: "已保存（本地）",
    statusError: "同步失败",
    statusSynced: "已同步",
    statusConflictShort: "冲突",
    statusErrorShort: "错误",
    emptyContent: "（空）",
    confirmDelete: "确定删除这一天的日记？",
    confirmClear: "将清除全部本地数据，是否继续？",
    importUnsupported: "不支持的备份版本",
    loginNeedServer: "请先填写服务端地址",
    loginNeedCreds: "请输入用户名和密码",
    loginSuccess: "登录成功",
    logoutSuccess: "已登出",
    syncNeedLogin: "请先登录并配置服务端",
    calendarToggle: "切换日历",
    syncStatusLabel: "同步状态"
  },
  en: {
    navHistory: "History",
    navSettings: "Settings",
    titleToday: "Today",
    titleHistory: "History",
    titleSettings: "Settings",
    titleConflict: "Conflict",
    editorPreview: "Preview",
    editorEdit: "Edit",
    editorDelete: "Delete",
    editorPlaceholder: "Write something today...",
    historyTitle: "Journal history",
    historySearchPlaceholder: "Search entries",
    historyEmpty: "No entries yet",
    historyNoMatch: "No matches found",
    settingsServerUrl: "Server URL",
    settingsUsername: "Username",
    settingsPassword: "Password",
    settingsRemember: "Remember login",
    settingsLogin: "Log in",
    settingsLogout: "Log out",
    settingsLoggedInAs: "Logged in as: ",
    settingsUnknownUser: "Unknown",
    settingsAutoSync: "Auto sync in foreground",
    settingsExport: "Export",
    settingsImport: "Import",
    settingsClear: "Clear local data",
    settingsLanguage: "Language",
    settingsLanguageAuto: "Follow browser",
    settingsLanguageZh: "Chinese",
    settingsLanguageEn: "English",
    conflictTitle: "Resolve conflict",
    conflictLocal: "Local version",
    conflictRemote: "Remote version",
    conflictRemoteDeleted: "Remote entry deleted",
    conflictUseLocal: "Overwrite remote with local",
    conflictUseRemote: "Overwrite local with remote",
    syncNetwork: "Network",
    syncOnline: "Online",
    syncOffline: "Offline",
    syncServer: "Server",
    syncReachable: "Reachable",
    syncUnreachable: "Unreachable",
    syncPending: "Pending",
    syncLast: "Last sync",
    syncError: "Error",
    syncSyncing: "Syncing",
    syncNow: "Sync now",
    syncConflicts: "View conflicts",
    statusConflict: "Conflict",
    statusPending: "Pending",
    statusPendingOffline: "Saved locally",
    statusError: "Sync failed",
    statusSynced: "Synced",
    statusConflictShort: "Conflict",
    statusErrorShort: "Error",
    emptyContent: "(empty)",
    confirmDelete: "Delete this day's entry?",
    confirmClear: "This will clear all local data. Continue?",
    importUnsupported: "Unsupported backup version",
    loginNeedServer: "Please enter the server URL",
    loginNeedCreds: "Please enter username and password",
    loginSuccess: "Logged in",
    logoutSuccess: "Logged out",
    syncNeedLogin: "Please log in and configure server",
    calendarToggle: "Toggle calendar",
    syncStatusLabel: "Sync status"
  }
} as const;

type TranslationKey = keyof typeof translations.zh;

function detectLanguage(): "zh" | "en" {
  return navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en";
}

const view = ref<"editor" | "history" | "settings" | "conflict">("editor");
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
const authMessage = ref("");
const authMessageTone = ref<"success" | "error">("success");

const settings = reactive<Settings>({
  serverBaseUrl: "",
  username: "",
  token: null,
  rememberCredentials: false,
  autoSyncEnabled: true,
  language: "auto"
});

const syncStatus = reactive({
  online: navigator.onLine,
  serverReachable: false,
  syncing: false,
  lastSyncAt: "",
  lastError: ""
});

const resolvedLanguage = computed(() => {
  if (!settings.language || settings.language === "auto") {
    return detectLanguage();
  }
  return settings.language;
});
const localeTag = computed(() => (resolvedLanguage.value === "zh" ? "zh-CN" : "en-US"));
const t = (key: TranslationKey) => translations[resolvedLanguage.value][key];

const weekdayLabels = computed(() =>
  resolvedLanguage.value === "zh"
    ? ["日", "一", "二", "三", "四", "五", "六"]
    : ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
);

const calendarWeeks = computed(() => buildMonthCalendar(calendarCursor.value));
const calendarTitle = computed(() =>
  calendarCursor.value.toLocaleDateString(localeTag.value, { year: "numeric", month: "long" })
);

const headerTitle = computed(() => {
  if (view.value === "editor") {
    return selectedDate.value === todayString() ? t("titleToday") : selectedDate.value;
  }
  if (view.value === "history") {
    return t("titleHistory");
  }
  if (view.value === "settings") {
    return t("titleSettings");
  }
  return t("titleConflict");
});

const previewHtml = computed(() => marked.parse(editorContent.value || ""));

const entryMarkers = computed<Record<string, boolean>>(() => {
  const markers: Record<string, boolean> = {};
  for (const entry of entries.value) {
    markers[entry.date] = true;
  }
  return markers;
});

const historyList = computed(() => {
  if (!searchQuery.value) {
    return entries.value;
  }
  const keyword = searchQuery.value.toLowerCase();
  return entries.value.filter((entry) => entry.content.toLowerCase().includes(keyword));
});

const pendingCount = ref(0);
const conflictCount = ref(0);
const isLoggedIn = computed(() => !!activeToken());
const loggedInLabel = computed(
  () => `${t("settingsLoggedInAs")}${settings.username || t("settingsUnknownUser")}`
);
const pendingDaysLabel = computed(() => {
  const count = pendingCount.value;
  if (resolvedLanguage.value === "zh") {
    return `${count} 天`;
  }
  return `${count} day${count === 1 ? "" : "s"}`;
});
const conflictButtonLabel = computed(() => `${t("syncConflicts")} (${conflictCount.value})`);

const entryStatusLabel = computed(() => {
  if (!currentEntry.value) {
    return "";
  }
  if (currentEntry.value.syncStatus === "conflict") {
    return t("statusConflict");
  }
  if (currentEntry.value.syncStatus === "pending") {
    return syncStatus.online ? t("statusPending") : t("statusPendingOffline");
  }
  if (currentEntry.value.syncStatus === "error") {
    return t("statusError");
  }
  return t("statusSynced");
});

const editorModeLabel = computed(() => (editorMode.value === "edit" ? t("editorPreview") : t("editorEdit")));

const syncDotClass = computed(() => {
  if (conflictCount.value > 0) {
    return "status-dot--conflict";
  }
  if (pendingCount.value > 0) {
    return "status-dot--pending";
  }
  if (!settings.serverBaseUrl || !activeToken()) {
    return "status-dot--unauth";
  }
  if (!syncStatus.online) {
    return "status-dot--offline";
  }
  return syncStatus.syncing ? "status-dot--syncing" : "status-dot--ok";
});

let saveTimer: number | null = null;

function toggleCalendar() {
  if (window.matchMedia("(max-width: 768px)").matches) {
    calendarDrawerOpen.value = true;
    return;
  }
  isSidebarCollapsed.value = !isSidebarCollapsed.value;
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
  return normalized.length > 80 ? `${normalized.slice(0, 80)}...` : normalized || t("emptyContent");
}

function statusLabel(entry: EntryRecord) {
  if (entry.syncStatus === "pending") {
    return t("statusPending");
  }
  if (entry.syncStatus === "conflict") {
    return t("statusConflictShort");
  }
  if (entry.syncStatus === "error") {
    return t("statusErrorShort");
  }
  return t("statusSynced");
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
  if (!confirm(t("confirmDelete"))) {
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
  authMessage.value = "";
  if (!settings.serverBaseUrl) {
    syncStatus.lastError = t("loginNeedServer");
    authMessageTone.value = "error";
    authMessage.value = syncStatus.lastError;
    return;
  }
  if (!settings.username || !loginPassword.value) {
    syncStatus.lastError = t("loginNeedCreds");
    authMessageTone.value = "error";
    authMessage.value = syncStatus.lastError;
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
    authMessageTone.value = "success";
    authMessage.value = t("loginSuccess");
    if (settings.autoSyncEnabled && navigator.onLine) {
      await triggerSync();
    } else {
      await checkServerReachable();
    }
  } catch (err) {
    syncStatus.lastError = err instanceof Error ? err.message : "login_failed";
    authMessageTone.value = "error";
    authMessage.value = syncStatus.lastError;
  }
}

async function logout() {
  sessionToken.value = null;
  settings.token = null;
  await saveSettings({ ...settings });
  syncStatus.lastError = "";
  authMessageTone.value = "success";
  authMessage.value = t("logoutSuccess");
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
    syncStatus.lastError = t("syncNeedLogin");
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
    alert(t("importUnsupported"));
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
  if (!confirm(t("confirmClear"))) {
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
