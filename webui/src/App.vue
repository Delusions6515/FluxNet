<script setup>
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { MiuixIcon, MiuixNavigationBar, MiuixScrollArea, MiuixSnackbarHost, MiuixTopAppBar } from 'miuix-vue'
import { GridView, Playlist, Tune, Settings, Info } from 'miuix-vue/icons'
import { MODULE_VERSION, requestEdgeToEdge } from './api/module'

const pages = [
  defineAsyncComponent(() => import('./pages/OverviewPage.vue')),
  defineAsyncComponent(() => import('./pages/SubscriptionsPage.vue')),
  defineAsyncComponent(() => import('./pages/ProxyPage.vue')),
  defineAsyncComponent(() => import('./pages/SettingsPage.vue')),
  defineAsyncComponent(() => import('./pages/AboutPage.vue')),
]
const titles = ['概览', '订阅', '代理', '设置', '关于']
const icons = [GridView, Playlist, Tune, Settings, Info]
const index = ref(0)
const activePage = computed(() => pages[index.value])
const scroller = ref(null)
const scrollPositions = new Map()
const bottomBar = ref(null)
let observer

watch(index, (_, previous) => scrollPositions.set(previous, scroller.value?.getScrollTop?.() ?? 0), { flush: 'pre' })
function restoreScroll() { scroller.value?.setScrollTop?.(scrollPositions.get(index.value) ?? 0) }
function syncSnackbarInset() { document.documentElement.style.setProperty('--m-snackbar-inset-bottom', `${bottomBar.value?.offsetHeight ?? 0}px`) }

onMounted(() => {
  requestEdgeToEdge()
  if (bottomBar.value) { observer = new ResizeObserver(syncSnackbarInset); observer.observe(bottomBar.value) }
  syncSnackbarInset()
})
onBeforeUnmount(() => { observer?.disconnect(); document.documentElement.style.removeProperty('--m-snackbar-inset-bottom') })
</script>

<template>
  <div class="app-shell">
    <MiuixTopAppBar class="app-shell__top" title="FluxNet" :subtitle="MODULE_VERSION ? `v${MODULE_VERSION}` : titles[index]" />
    <MiuixScrollArea ref="scroller" class="app-shell__body">
      <Transition name="page" mode="out-in" @enter="restoreScroll"><KeepAlive><component :is="activePage" :key="index" /></KeepAlive></Transition>
    </MiuixScrollArea>
    <div ref="bottomBar" class="app-shell__bottom">
      <MiuixNavigationBar v-model="index" :items="titles.map((label) => ({ label }))">
        <template #icon="{ index: itemIndex }"><MiuixIcon :icon="icons[itemIndex]" :size="26" /></template>
      </MiuixNavigationBar>
    </div>
  </div>
  <MiuixSnackbarHost />
</template>
