<script setup>
import { computed, onActivated, onMounted, ref } from 'vue'
import { MiuixCard, MiuixDropdownPreference, MiuixSmallTitle, MiuixSwitchPreference, MiuixText, setThemeMode, useTheme } from 'miuix-vue'
import { getSettings, setSetting } from '@/api/module'

const settings = ref(null); const error = ref(''); const { mode } = useTheme(); const themeModes = ['system', 'light', 'dark']
const themeIndex = computed({ get: () => Math.max(0, themeModes.indexOf(mode.value)), set: (value) => { const next = themeModes[value] || 'system'; setThemeMode(next); localStorage.setItem('fluxnet-theme', next) } })
async function load() { try { settings.value = await getSettings() } catch (err) { error.value = err.message } }
async function toggleAutostart(value) { try { settings.value = await setSetting('autostart', value ? '1' : '0') } catch (err) { error.value = err.message } }
const stored = localStorage.getItem('fluxnet-theme'); if (stored) setThemeMode(stored)
onMounted(load); onActivated(load)
</script>

<template>
  <div class="page"><MiuixSmallTitle text="外观" /><MiuixCard class="section section--compact"><MiuixDropdownPreference v-model="themeIndex" title="主题" :items="['跟随系统', '浅色', '深色']" /></MiuixCard><MiuixSmallTitle text="启动" /><MiuixCard class="section section--compact"><MiuixSwitchPreference v-if="settings" :model-value="settings.autostart" title="开机自启" summary="下次开机自动启动 FluxNet" @update:model-value="toggleAutostart" /></MiuixCard><MiuixText v-if="error" class="section error" type="body2">{{ error }}</MiuixText></div>
</template>
