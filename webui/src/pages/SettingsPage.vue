<script setup>
import { computed, onActivated, onMounted, ref } from "vue";
import {
  MiuixButton,
  MiuixCard,
  MiuixDropdownPreference,
  MiuixProgressIndicator,
  MiuixSmallTitle,
  MiuixSwitchPreference,
  MiuixText,
  setThemeMode,
  showSnackbar,
  useTheme,
} from "miuix-vue";
import {
  getKernelStatus,
  getSettings,
  setKernelChannel,
  setSetting,
  upgradeKernel,
} from "@/api/module";

const settings = ref(null);
const kernel = ref(null);
const upgrading = ref(false);
const error = ref("");
const { mode } = useTheme();
const themeModes = ["system", "light", "dark"];
const themeIndex = computed({
  get: () => Math.max(0, themeModes.indexOf(mode.value)),
  set: (value) => {
    const next = themeModes[value] || "system";
    setThemeMode(next);
    localStorage.setItem("fluxnet-theme", next);
  },
});
const channels = [
  "delusions6515-pre",
  "delusions6515-stable",
  "ref1nd-pre",
  "ref1nd-stable",
  "official-stable",
  "official-pre",
];
const channelLabels = {
  "delusions6515-pre": "Delusions6515 预览版",
  "delusions6515-stable": "Delusions6515 稳定版",
  "ref1nd-pre": "reF1nd 预览版",
  "ref1nd-stable": "reF1nd 稳定版",
  "official-stable": "官方稳定版",
  "official-pre": "官方预览版",
};
const channelIndex = computed({
  get: () => Math.max(0, channels.indexOf(kernel.value?.channel || "")),
  set: async (value) => {
    await changeChannel(channels[value]);
  },
});
function showError(err) {
  showSnackbar({ message: err.message || "操作失败", withDismissAction: true });
}
async function load() {
  try {
    settings.value = await getSettings();
    kernel.value = await getKernelStatus();
  } catch (err) {
    error.value = err.message;
    showError(err);
  }
}
async function toggleAutostart(value) {
  try {
    settings.value = await setSetting("autostart", value ? "1" : "0");
  } catch (err) {
    error.value = err.message;
    showError(err);
  }
}
async function changeChannel(value) {
  try {
    kernel.value = await setKernelChannel(value, "arm64-v8a");
    showSnackbar({
      message: `内核渠道已切换到 ${channelLabels[value]}, 更新后生效`,
      withDismissAction: true,
    });
  } catch (err) {
    error.value = err.message;
    showError(err);
  }
}
async function updateKernel() {
  if (upgrading.value) return;
  upgrading.value = true;
  try {
    kernel.value = await upgradeKernel();
    showSnackbar({
      message: `内核已更新到 ${kernel.value.version}`,
      withDismissAction: true,
    });
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    upgrading.value = false;
  }
}
const stored = localStorage.getItem("fluxnet-theme");
if (stored) setThemeMode(stored);
onMounted(load);
onActivated(load);
</script>

<template>
  <div class="page">
    <MiuixSmallTitle text="外观" />
    <MiuixCard class="section section--compact">
      <MiuixDropdownPreference
        v-model="themeIndex"
        title="主题"
        :items="['跟随系统', '浅色', '深色']"
      />
    </MiuixCard>
    <MiuixSmallTitle text="启动" />
    <MiuixCard class="section section--compact">
      <MiuixSwitchPreference
        v-if="settings"
        :model-value="settings.autostart"
        title="开机自启"
        summary="下次开机自动启动 FluxNet"
        @update:model-value="toggleAutostart"
      />
    </MiuixCard>
    <MiuixSmallTitle text="内核" />
    <MiuixCard class="section section--compact">
      <MiuixDropdownPreference
        v-if="kernel"
        v-model="channelIndex"
        title="更新渠道"
        summary="渠道切换仅改变后续更新来源, 当前内核不受影响"
        :items="channels.map((c) => channelLabels[c])"
      />
    </MiuixCard>
    <MiuixCard class="section">
      <div class="kernel-row">
        <div class="kernel-meta">
          <MiuixText v-if="kernel" type="body2">
            {{
              kernel.installed ? `已安装 v${kernel.version}` : "内核尚未安装"
            }}
          </MiuixText>
        </div>
        <MiuixButton
          text
          :loading="upgrading"
          :disabled="upgrading"
          :style="{ minWidth: '100px' }"
          @click="updateKernel"
        >
          {{ upgrading ? "更新中" : "更新内核" }}
        </MiuixButton>
      </div>
      <MiuixProgressIndicator v-if="upgrading" class="kernel-progress" />
    </MiuixCard>
    <MiuixSmallTitle text="构建信息" />
    <MiuixCard class="section">
      <MiuixText type="body2">内核 ABI 固定为 arm64-v8a</MiuixText>
    </MiuixCard>
    <MiuixText v-if="error" class="section error" type="body2">
      {{ error }}
    </MiuixText>
  </div>
</template>

<style>
.kernel-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.kernel-progress {
  margin-top: 10px;
}
</style>
