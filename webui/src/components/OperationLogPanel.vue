<script setup>
import { MiuixCard, MiuixSmallTitle, MiuixText } from 'miuix-vue'
defineProps({ entries: { type: Array, default: () => [] } })
</script>

<template>
  <MiuixSmallTitle text="最近日志" />
  <MiuixCard class="section log-panel">
    <MiuixText v-if="!entries.length" class="muted" type="body2">暂无日志</MiuixText>
    <div v-for="(entry, index) in entries" :key="`${entry.timestamp}-${index}`" class="log-row">
      <span class="log-row__time">{{ entry.timestamp }}</span>
      <span :class="entry.level === 'error' ? 'error' : ''">{{ entry.message || entry.event }}</span>
    </div>
  </MiuixCard>
</template>

<style scoped>
.log-panel { max-height: 260px; overflow: auto; }
.log-row { display: grid; gap: 3px; padding: 8px 0; border-bottom: 1px solid var(--m-color-divider-line); font: 12px/1.5 'JetBrains Mono Variable', monospace; overflow-wrap: anywhere; }
.log-row:last-child { border-bottom: 0; }
.log-row__time { color: var(--m-color-on-surface-variant-summary); }
</style>
