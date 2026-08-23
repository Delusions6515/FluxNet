<script setup>
import {
  computed,
  defineAsyncComponent,
  onActivated,
  onMounted,
  ref,
  watch,
} from "vue";
import {
  MiuixButton,
  MiuixCard,
  MiuixDropdownPreference,
  MiuixInput,
  MiuixSmallTitle,
  MiuixSwitchPreference,
  MiuixTabRow,
  MiuixText,
  showSnackbar,
} from "miuix-vue";
import {
  getInstalledApps,
  getProxyPackageCatalog,
  getSettings,
  partitionInstalledApps,
  replaceAppList,
  replaceAppLists,
  replaceForceAppList,
  setSetting,
  upgradeProxyPackageList,
} from "@/api/module";

const ConfigEditor = defineAsyncComponent(
  () => import("@/components/ConfigEditor.vue"),
);
const settings = ref(null);
const apps = ref([]);
const query = ref("");
const selected = ref([]);
const pinned = ref([]);
const error = ref("");
const savingApps = ref(false);
const upgrading = ref(false);
const generatingApps = ref(false);
const loadingApps = ref(false);
const tab = ref(0);
const editorVisited = ref(true);
const appListEditorOpen = ref(false);
const modes = ["tun", "tproxy", "redirect", "ebpf"];
const appModes = ["blacklist", "whitelist"];
const modeIndex = computed({
  get: () => Math.max(0, modes.indexOf(settings.value?.proxy_mode)),
  set: (value) => saveSetting("proxy_mode", modes[value]),
});
const appModeIndex = computed({
  get: () => Math.max(0, appModes.indexOf(settings.value?.app_proxy_mode)),
  set: (value) => saveSetting("app_proxy_mode", appModes[value]),
});
const tunStackIndex = computed({
  get: () =>
    Math.max(
      0,
      ["system", "gvisor", "mixed"].indexOf(settings.value?.tun_stack),
    ),
  set: (value) =>
    saveSetting("tun_stack", ["system", "gvisor", "mixed"][value]),
});
const isTun = computed(() => settings.value?.proxy_mode === "tun");
const usesTproxy = computed(() =>
  ["tproxy", "redirect"].includes(settings.value?.proxy_mode),
);
const automaticApps = computed(() => settings.value?.auto_mode);
const appListKind = computed(() => {
  if (automaticApps.value)
    return settings.value?.app_proxy_mode === "whitelist" ? "bypass" : "proxy";
  return settings.value?.app_proxy_mode === "whitelist" ? "proxy" : "bypass";
});
const appListTitle = computed(() => {
  if (automaticApps.value)
    return appListKind.value === "proxy" ? "强制代理名单" : "强制绕过名单";
  return appListKind.value === "proxy" ? "代理应用名单" : "绕过应用名单";
});
const visibleApps = computed(() =>
  apps.value
    .filter((app) =>
      `${app.appLabel} ${app.packageName}`
        .toLowerCase()
        .includes(query.value.toLowerCase()),
    )
    .sort(
      (a, b) =>
        Number(pinned.value.includes(b.packageName)) -
        Number(pinned.value.includes(a.packageName)),
    ),
);

function showError(err) {
  showSnackbar({ message: err.message || "操作失败", withDismissAction: true });
}
function currentSelection(nextSettings = settings.value) {
  if (!nextSettings) return [];
  const kind = nextSettings.auto_mode
    ? nextSettings.app_proxy_mode === "whitelist"
      ? "bypass"
      : "proxy"
    : nextSettings.app_proxy_mode === "whitelist"
      ? "proxy"
      : "bypass";
  const key = `${nextSettings.auto_mode ? "force_" : ""}${kind}_apps`;
  return [...(nextSettings[key] || [])];
}
function syncSelection(nextSettings = settings.value) {
  selected.value = currentSelection(nextSettings);
  pinned.value = [...selected.value];
}
async function load() {
  try {
    settings.value = await getSettings();
    syncSelection();
  } catch (err) {
    error.value = err.message;
    showError(err);
  }
}
async function openAppListEditor() {
  loadingApps.value = true;
  try {
    apps.value = await getInstalledApps();
    query.value = "";
    appListEditorOpen.value = true;
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    loadingApps.value = false;
  }
}
async function saveSetting(key, value) {
  try {
    settings.value = await setSetting(key, value);
    if (key === "app_proxy_mode" || key === "auto_mode") syncSelection();
    showSnackbar({
      message: "设置已保存，等待手动应用配置",
      withDismissAction: true,
    });
  } catch (err) {
    error.value = err.message;
    showError(err);
  }
}
function toggle(packageName) {
  selected.value = selected.value.includes(packageName)
    ? selected.value.filter((item) => item !== packageName)
    : [...selected.value, packageName];
}
async function saveApps() {
  savingApps.value = true;
  try {
    settings.value = automaticApps.value
      ? await replaceForceAppList(appListKind.value, selected.value)
      : await replaceAppList(appListKind.value, selected.value);
    pinned.value = [...selected.value];
    showSnackbar({
      message: "应用名单已保存，等待手动应用配置",
      withDismissAction: true,
    });
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    savingApps.value = false;
  }
}
async function generateApps() {
  generatingApps.value = true;
  try {
    const [catalog, installedApps] = await Promise.all([
      getProxyPackageCatalog(),
      getInstalledApps(),
    ]);
    const { proxyApps, bypassApps } = partitionInstalledApps(
      catalog.packages || [],
      installedApps,
    );
    settings.value = await replaceAppLists(proxyApps, bypassApps);
    syncSelection();
    showSnackbar({ message: "应用名单已生成", withDismissAction: true });
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    generatingApps.value = false;
  }
}
async function upgradeApps() {
  upgrading.value = true;
  try {
    await upgradeProxyPackageList();
    showSnackbar({ message: "预置名单已更新", withDismissAction: true });
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    upgrading.value = false;
  }
}

watch(tab, (value) => {
  if (value === 0) editorVisited.value = true;
});
onMounted(load);
onActivated(load);
</script>

<template>
  <div class="page">
    <MiuixCard class="section section--compact proxy-tabs">
      <MiuixTabRow
        v-model="tab"
        :tabs="['普通设置', '分应用代理设置']"
        contour
      />
    </MiuixCard>

    <div v-show="tab === 0">
      <MiuixSmallTitle text="代理模式" />
      <MiuixCard class="section section--compact">
        <MiuixDropdownPreference
          v-model="modeIndex"
          title="代理模式"
          :items="modes"
        />
      </MiuixCard>
      <template v-if="isTun">
        <MiuixSmallTitle text="Tun 设置" />
        <MiuixCard class="section section--compact">
          <MiuixDropdownPreference
            v-model="tunStackIndex"
            title="协议栈"
            :items="['system', 'gvisor', 'mixed']"
          />
          <MiuixSwitchPreference
            :model-value="settings?.auto_redirect"
            title="自动重定向"
            @update:model-value="
              (value) => saveSetting('auto_redirect', value ? '1' : '0')
            "
          />
          <MiuixSwitchPreference
            :model-value="settings?.tun_forward"
            title="热点共享转发"
            @update:model-value="
              (value) => saveSetting('tun_forward', value ? '1' : '0')
            "
          />
        </MiuixCard>
      </template>
      <ConfigEditor
        v-if="editorVisited && settings"
        v-show="tab === 0"
        target="inbound"
        :mode="settings.proxy_mode"
      />
      <ConfigEditor v-if="usesTproxy" target="tproxy" />
    </div>

    <div v-show="tab === 1 && appListEditorOpen" class="editor-page">
      <div class="section page-actions">
        <MiuixButton type="secondary" @click="appListEditorOpen = false">
          返回分应用代理设置
        </MiuixButton>
      </div>
      <MiuixSmallTitle :text="appListTitle" />
      <MiuixCard class="section">
        <MiuixInput v-model="query" label="搜索已安装应用" />
        <div class="app-list">
          <button
            v-for="app in visibleApps"
            :key="app.packageName"
            class="app-row"
            :class="{ selected: selected.includes(app.packageName) }"
            @click="toggle(app.packageName)"
          >
            <img
              class="app-row__icon"
              :src="`ksu://icon/${app.packageName}`"
              alt=""
              @error="
                (event) => {
                  event.target.style.display = 'none';
                }
              "
            />
            <span>
              <strong>{{ app.appLabel }}</strong>
              <small>{{ app.packageName }}</small>
            </span>
            <b>{{ selected.includes(app.packageName) ? "已选" : "" }}</b>
          </button>
        </div>
        <div class="page-actions">
          <MiuixButton
            :disabled="savingApps || generatingApps"
            @click="saveApps"
          >
            {{ savingApps ? "保存中…" : "保存应用名单" }}
          </MiuixButton>
        </div>
      </MiuixCard>
    </div>

    <div v-show="tab === 1 && !appListEditorOpen">
      <MiuixSmallTitle text="分应用代理" />
      <MiuixCard class="section section--compact">
        <MiuixSwitchPreference
          v-if="settings"
          :model-value="settings.app_proxy_enable"
          title="启用分应用代理"
          @update:model-value="
            (value) => saveSetting('app_proxy_enable', value ? '1' : '0')
          "
        />
        <MiuixDropdownPreference
          v-model="appModeIndex"
          title="规则模式"
          :items="['绕过所选应用', '仅代理所选应用']"
        />
        <MiuixSwitchPreference
          v-if="settings"
          :model-value="settings.auto_mode"
          title="自动模式"
          @update:model-value="
            (value) => saveSetting('auto_mode', value ? '1' : '0')
          "
        />
      </MiuixCard>
      <MiuixCard v-if="settings" class="section">
        <div class="page-actions">
          <MiuixButton
            type="secondary"
            :disabled="upgrading"
            @click="upgradeApps"
          >
            {{ upgrading ? "更新中…" : "更新预置名单" }}
          </MiuixButton>
          <MiuixButton
            type="secondary"
            :disabled="generatingApps"
            @click="generateApps"
          >
            {{ generatingApps ? "生成中…" : "从预置名单生成" }}
          </MiuixButton>
          <MiuixButton :disabled="loadingApps" @click="openAppListEditor">
            {{ loadingApps ? "加载中…" : `编辑${appListTitle}` }}
          </MiuixButton>
        </div>
      </MiuixCard>
    </div>
    <MiuixText v-if="error" class="section error" type="body2">
      {{ error }}
    </MiuixText>
  </div>
</template>

<style scoped>
.editor-page {
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.app-list {
  max-height: 360px;
  overflow: auto;
  margin: 12px -4px;
}
.app-row {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 12px;
  justify-content: space-between;
  padding: 12px 8px;
  border: 0;
  border-bottom: 1px solid var(--m-color-divider-line);
  background: transparent;
  color: inherit;
  text-align: left;
}
.app-row__icon {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 8px;
  object-fit: cover;
}
.app-row.selected {
  color: var(--m-color-primary);
}
.app-row span {
  display: grid;
  flex: 1;
  gap: 3px;
  min-width: 0;
  overflow-wrap: anywhere;
}
.app-row small {
  color: var(--m-color-on-surface-variant-summary);
}
</style>
