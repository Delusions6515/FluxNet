<script setup>
import { computed, onMounted, ref } from 'vue'
import { Codemirror } from 'vue-codemirror'
import { json } from '@codemirror/lang-json'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'
import { MiuixButton, MiuixCard, MiuixSmallTitle, MiuixText, showSnackbar, useTheme } from 'miuix-vue'
import { readLocalSubscription, writeLocalSubscription } from '@/api/module'

const props = defineProps({ name: { type: String, required: true } })
const emit = defineEmits(['saved'])
const content = ref('')
const original = ref('')
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const { theme } = useTheme()
const extensions = computed(() => [json(), EditorView.lineWrapping, ...(theme.value === 'dark' ? [oneDark] : [])])
const dirty = computed(() => content.value !== original.value)

async function load() {
  loading.value = true; error.value = ''
  try { const data = await readLocalSubscription(props.name); content.value = data.content; original.value = data.content } catch (err) { error.value = err.message } finally { loading.value = false }
}
async function save() {
  try { JSON.parse(content.value) } catch (err) { error.value = `JSON 格式无效: ${err.message}`; return }
  saving.value = true; error.value = ''
  try { await writeLocalSubscription(props.name, content.value); original.value = content.value; showSnackbar({ message: '本地订阅已保存', withDismissAction: true }); emit('saved') } catch (err) { error.value = err.message } finally { saving.value = false }
}
onMounted(load)
</script>

<template>
  <div class="json-editor">
    <MiuixSmallTitle :text="`编辑本地订阅: ${name}`" />
    <MiuixCard class="section json-editor__card">
      <Codemirror v-model="content" class="json-editor__code" :extensions="extensions" :disabled="loading || saving" placeholder="加载中…" />
      <MiuixText v-if="error" class="error json-editor__error" type="body2">{{ error }}</MiuixText>
      <div class="page-actions json-editor__actions"><MiuixButton type="secondary" :disabled="loading || saving || !dirty" @click="load">重新加载</MiuixButton><MiuixButton :disabled="loading || saving || !dirty" @click="save">{{ saving ? '保存中…' : '保存' }}</MiuixButton></div>
    </MiuixCard>
  </div>
</template>

<style scoped>
.json-editor { display: flex; flex: 1; min-height: 0; flex-direction: column; }
.json-editor__card { display: flex; flex: 1; min-height: 0; flex-direction: column; }
.json-editor__card :deep(.m-card) { display: flex; flex: 1; min-height: 0; flex-direction: column; }
.json-editor__code { flex: 1; min-height: 300px; }
.json-editor__code :deep(.cm-editor), .json-editor__code :deep(.cm-scroller) { height: 100%; min-height: 300px; font: 13px/1.55 'JetBrains Mono Variable', monospace; }
.json-editor__error, .json-editor__actions { margin-top: 12px; }
</style>
