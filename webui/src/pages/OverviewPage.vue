<script setup>
import { computed, onActivated, onMounted, ref } from 'vue'
import { MiuixButton, MiuixCard, MiuixSmallTitle, MiuixText, showSnackbar } from 'miuix-vue'
import OperationLogPanel from '@/components/OperationLogPanel.vue'
import { getOverview, serviceAction } from '@/api/module'

const data = ref(null); const loading = ref(false); const busy = ref(''); const error = ref('')
const status = computed(() => data.value?.service)
function showError(err) { showSnackbar({ message: err.message || '操作失败', withDismissAction: true }) }
async function refresh() { loading.value = true; error.value = ''; try { data.value = await getOverview() } catch (err) { error.value = err.message; showError(err) } finally { loading.value = false } }
async function run(action) { busy.value = action; try { await serviceAction(action); showSnackbar({ message: action === 'restart' ? '配置已应用并重启' : '服务操作完成', withDismissAction: true }); await refresh() } catch (err) { error.value = err.message; showError(err) } finally { busy.value = '' } }
onMounted(refresh); onActivated(refresh)
</script>

<template>
  <div class="page">
    <MiuixSmallTitle text="运行状态" />
    <MiuixCard class="section"><div class="status-grid"><div><span class="muted">服务</span><strong :class="status?.running ? 'ok' : 'error'">{{ status?.running ? '运行中' : '未运行' }}</strong></div><div><span class="muted">代理模式</span><strong>{{ status?.mode || '-' }}</strong></div><div><span class="muted">活跃订阅</span><strong>{{ data?.subscriptions?.active || '-' }}</strong></div><div><span class="muted">健康检查</span><strong :class="data?.health?.process_alive ? 'ok' : 'error'">{{ data?.health?.process_alive ? '正常' : '异常' }}</strong></div></div></MiuixCard>
    <MiuixSmallTitle text="服务控制" />
    <MiuixCard class="section"><div class="page-actions"><MiuixButton type="primary" :disabled="Boolean(busy) || status?.running" @click="run('start')">{{ busy === 'start' ? '启动中…' : '启动' }}</MiuixButton><MiuixButton type="secondary" :disabled="Boolean(busy) || !status?.running" @click="run('stop')">停止</MiuixButton><MiuixButton type="secondary" :disabled="Boolean(busy)" @click="run('restart')">应用并重启</MiuixButton><MiuixButton type="secondary" :disabled="loading" @click="refresh">刷新</MiuixButton></div></MiuixCard>
    <MiuixText v-if="error" class="section error" type="body2">{{ error }}</MiuixText>
    <OperationLogPanel :entries="data?.logs" />
  </div>
</template>

<style scoped>
.status-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }.status-grid div { display: grid; gap: 4px; }.ok { color: #36d167; }
</style>
