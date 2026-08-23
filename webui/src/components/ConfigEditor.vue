<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { Codemirror } from "vue-codemirror";
import { json } from "@codemirror/lang-json";
import { oneDark } from "@codemirror/theme-one-dark";
import { EditorView } from "@codemirror/view";
import {
  MiuixButton,
  MiuixCard,
  MiuixSmallTitle,
  MiuixText,
  showSnackbar,
  useTheme,
} from "miuix-vue";
import {
  readUserInbound,
  readUserTproxy,
  writeUserInbound,
  writeUserTproxy,
} from "@/api/module";

const props = defineProps({
  mode: { type: String, default: "" },
  target: { type: String, required: true },
});
const content = ref("");
const original = ref("");
const loading = ref(true);
const saving = ref(false);
const error = ref("");
const { theme } = useTheme();
const isInbound = computed(() => props.target === "inbound");
const title = computed(() =>
  isInbound.value ? `编辑用户入站: ${props.mode}` : "编辑用户 tproxy.conf",
);
const extensions = computed(() => [
  EditorView.lineWrapping,
  ...(isInbound.value ? [json()] : []),
  ...(theme.value === "dark" ? [oneDark] : []),
]);
const dirty = computed(() => content.value !== original.value);

function showError(err) {
  showSnackbar({ message: err.message || "操作失败", withDismissAction: true });
}
async function load() {
  loading.value = true;
  error.value = "";
  try {
    const data = isInbound.value
      ? await readUserInbound(props.mode)
      : await readUserTproxy();
    content.value = data.content;
    original.value = data.content;
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    loading.value = false;
  }
}
async function save() {
  if (isInbound.value) {
    try {
      const parsed = JSON.parse(content.value);
      if (!parsed || Array.isArray(parsed) || typeof parsed !== "object")
        throw new Error("入站必须是 JSON 对象");
    } catch (err) {
      error.value = `JSON 格式无效: ${err.message}`;
      return;
    }
  }
  saving.value = true;
  error.value = "";
  try {
    if (isInbound.value) await writeUserInbound(props.mode, content.value);
    else await writeUserTproxy(content.value);
    original.value = content.value;
    showSnackbar({
      message: "配置已保存，等待手动应用配置",
      withDismissAction: true,
    });
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    saving.value = false;
  }
}

onMounted(load);
watch(
  () => [props.target, props.mode],
  () => load(),
);
</script>

<template>
  <div class="config-editor">
    <MiuixSmallTitle :text="title" />
    <MiuixCard class="section config-editor__card">
      <Codemirror
        v-model="content"
        class="config-editor__code"
        :extensions="extensions"
        :disabled="loading || saving"
        placeholder="加载中…"
      />
      <MiuixText v-if="error" class="error config-editor__error" type="body2">
        {{ error }}
      </MiuixText>
      <div class="page-actions config-editor__actions">
        <MiuixButton
          type="secondary"
          :disabled="loading || saving || !dirty"
          @click="load"
        >
          重新加载
        </MiuixButton>
        <MiuixButton :disabled="loading || saving || !dirty" @click="save">
          {{ saving ? "保存中…" : "保存" }}
        </MiuixButton>
      </div>
    </MiuixCard>
  </div>
</template>

<style scoped>
.config-editor__code {
  flex: 1;
  min-height: 0;
}
.config-editor__code :deep(.cm-editor),
.config-editor__code :deep(.cm-scroller) {
  height: 100%;
  min-height: 0;
  font:
    13px/1.55 "JetBrains Mono Variable",
    monospace;
}
.config-editor {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}
.config-editor__card {
  display: flex;
  flex: 1;
  min-height: 0;
}
.config-editor__card :deep(.m-card) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}
.config-editor__error,
.config-editor__actions {
  margin-top: 12px;
}
</style>
