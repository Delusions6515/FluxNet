<script setup>
import { computed, onActivated, onMounted, ref } from 'vue'
import { MiuixButton, MiuixCard, MiuixDropdownPreference, MiuixInput, MiuixSmallTitle, MiuixSwitchPreference, MiuixText, showSnackbar } from 'miuix-vue'
import { getInstalledApps, getSettings, replaceAppList, setSetting } from '@/api/module'

const settings = ref(null); const apps = ref([]); const query = ref(''); const selected = ref([]); const pinned = ref([]); const error = ref(''); const saving = ref(false)
const modes = ['tun', 'tproxy', 'redirect', 'ebpf']; const appModes = ['blacklist', 'whitelist']
const modeIndex = computed({ get: () => Math.max(0, modes.indexOf(settings.value?.proxy_mode)), set: (value) => saveSetting('proxy_mode', modes[value]) })
const appModeIndex = computed({ get: () => Math.max(0, appModes.indexOf(settings.value?.app_proxy_mode)), set: (value) => saveSetting('app_proxy_mode', appModes[value]) })
const visibleApps = computed(() => apps.value.filter((app) => `${app.appLabel} ${app.packageName}`.toLowerCase().includes(query.value.toLowerCase())).sort((a, b) => Number(pinned.value.includes(b.packageName)) - Number(pinned.value.includes(a.packageName))))
function showError(err) { showSnackbar({ message: err.message || '操作失败', withDismissAction: true }) }
async function load() { try { settings.value = await getSettings(); selected.value = [...(settings.value.app_proxy_mode === 'whitelist' ? settings.value.proxy_apps : settings.value.bypass_apps)]; pinned.value = [...selected.value]; apps.value = await getInstalledApps() } catch (err) { error.value = err.message; showError(err) } }
async function saveSetting(key, value) { try { settings.value = await setSetting(key, value); if (key === 'app_proxy_mode') { selected.value = [...(value === 'whitelist' ? settings.value.proxy_apps : settings.value.bypass_apps)]; pinned.value = [...selected.value] } showSnackbar({ message: '设置已保存，等待手动应用配置', withDismissAction: true }) } catch (err) { error.value = err.message; showError(err) } }
function toggle(packageName) { selected.value = selected.value.includes(packageName) ? selected.value.filter((item) => item !== packageName) : [...selected.value, packageName] }
async function saveApps() { saving.value = true; try { settings.value = await replaceAppList(settings.value.app_proxy_mode, selected.value); pinned.value = [...selected.value]; showSnackbar({ message: '应用名单已保存，等待手动应用配置', withDismissAction: true }) } catch (err) { error.value = err.message; showError(err) } finally { saving.value = false } }
onMounted(load); onActivated(load)
</script>

<template>
  <div class="page"><MiuixSmallTitle text="代理模式" /><MiuixCard class="section section--compact"><MiuixDropdownPreference v-model="modeIndex" title="代理模式" :items="modes" /></MiuixCard>
  <MiuixSmallTitle text="分应用代理" /><MiuixCard class="section section--compact"><MiuixSwitchPreference v-if="settings" :model-value="settings.app_proxy_enable" title="启用分应用代理" @update:model-value="(value) => saveSetting('app_proxy_enable', value ? '1' : '0')" /><MiuixDropdownPreference v-model="appModeIndex" title="规则模式" :items="['绕过所选应用', '仅代理所选应用']" /></MiuixCard>
  <MiuixCard class="section"><MiuixInput v-model="query" label="搜索已安装应用" /><div class="app-list"><button v-for="app in visibleApps" :key="app.packageName" class="app-row" :class="{ selected: selected.includes(app.packageName) }" @click="toggle(app.packageName)"><img class="app-row__icon" :src="`ksu://icon/${app.packageName}`" alt="" @error="(event) => { event.target.style.display = 'none' }" /><span><strong>{{ app.appLabel }}</strong><small>{{ app.packageName }}</small></span><b>{{ selected.includes(app.packageName) ? '已选' : '' }}</b></button></div><div class="page-actions"><MiuixButton :disabled="saving" @click="saveApps">{{ saving ? '保存中…' : '保存应用名单' }}</MiuixButton></div></MiuixCard><MiuixText v-if="error" class="section error" type="body2">{{ error }}</MiuixText></div>
</template>

<style scoped>
.app-list { max-height: 360px; overflow: auto; margin: 12px -4px; }.app-row { display: flex; width: 100%; align-items: center; gap: 12px; justify-content: space-between; padding: 12px 8px; border: 0; border-bottom: 1px solid var(--m-color-divider-line); background: transparent; color: inherit; text-align: left; }.app-row__icon { flex: none; width: 40px; height: 40px; border-radius: 8px; object-fit: cover; }.app-row.selected { color: var(--m-color-primary); }.app-row span { display: grid; flex: 1; gap: 3px; min-width: 0; overflow-wrap: anywhere; }.app-row small { color: var(--m-color-on-surface-variant-summary); }
</style>
