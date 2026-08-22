<script setup>
import {
  computed,
  defineAsyncComponent,
  onActivated,
  onMounted,
  ref,
} from "vue";
import {
  MiuixButton,
  MiuixCard,
  MiuixDialog,
  MiuixInput,
  MiuixSmallTitle,
  MiuixText,
  showSnackbar,
} from "miuix-vue";
import {
  addRemoteSubscription,
  createLocalSubscription,
  getSubscriptions,
  removeSubscription,
  switchSubscription,
  updateSubscription,
} from "@/api/module";

const JsonEditor = defineAsyncComponent(
  () => import("@/components/JsonEditor.vue"),
);
const index = ref({ active: "", subscriptions: [] });
const error = ref("");
const busy = ref("");
const creating = ref(false);
const editorName = ref("");
const editorSourceName = ref("");
const editorVisited = ref(false);
const dialogOpen = ref(false);
const dialogType = ref("");
const name = ref("");
const url = ref("");
const active = computed(() => index.value.active);
function showError(err) {
  showSnackbar({ message: err.message || "操作失败", withDismissAction: true });
}
function openDialog(type) {
  if (busy.value || creating.value) return;
  dialogType.value = type;
  dialogOpen.value = true;
}
function closeDialog() {
  dialogOpen.value = false;
  dialogType.value = "";
  name.value = "";
  url.value = "";
}
function openEditor(sourceName) {
  editorSourceName.value = sourceName;
  editorName.value = sourceName;
  editorVisited.value = true;
}
async function load() {
  try {
    index.value = await getSubscriptions();
  } catch (err) {
    error.value = err.message;
    showError(err);
  }
}
async function action(kind, item) {
  if (busy.value || creating.value) return;
  busy.value = `${kind}:${item.name}`;
  error.value = "";
  try {
    if (kind === "switch") await switchSubscription(item.name);
    if (kind === "update") await updateSubscription(item.name);
    if (kind === "remove") await removeSubscription(item.name);
    await load();
    showSnackbar({
      message: "订阅已更新，等待手动应用配置",
      withDismissAction: true,
    });
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    busy.value = "";
  }
}
async function create() {
  if (creating.value || busy.value) return;
  creating.value = true;
  error.value = "";
  try {
    if (dialogType.value === "local") {
      await createLocalSubscription(name.value);
      openEditor(name.value);
    } else await addRemoteSubscription(name.value, url.value);
    closeDialog();
    await load();
  } catch (err) {
    error.value = err.message;
    showError(err);
  } finally {
    creating.value = false;
  }
}
onMounted(load);
onActivated(load);
</script>

<template>
  <div class="page">
    <div v-show="editorName" class="editor-page">
      <div class="section page-actions">
        <MiuixButton type="secondary" @click="editorName = ''"
          >返回订阅列表</MiuixButton
        >
      </div>
      <Suspense v-if="editorVisited"
        ><template #default
          ><JsonEditor :name="editorSourceName" @saved="load" /></template
        ><template #fallback
          ><MiuixCard class="section editor-loading"
            ><MiuixText type="body2" class="muted"
              >正在加载编辑器…</MiuixText
            ></MiuixCard
          ></template
        ></Suspense
      >
    </div>
    <div v-show="!editorName">
      <MiuixSmallTitle text="订阅" /><MiuixCard class="section"
        ><MiuixText type="body2" class="muted">当前使用</MiuixText
        ><MiuixText type="body1">{{ active || "未选择" }}</MiuixText>
        <div class="page-actions subscription-add">
          <MiuixButton
            type="primary"
            :disabled="Boolean(busy) || creating"
            @click="openDialog('remote')"
            >添加远程订阅</MiuixButton
          ><MiuixButton
            type="secondary"
            :disabled="Boolean(busy) || creating"
            @click="openDialog('local')"
            >新建本地订阅</MiuixButton
          >
        </div></MiuixCard
      >
      <MiuixCard class="section section--compact"
        ><div
          v-for="item in index.subscriptions"
          :key="item.name"
          class="subscription-row"
        >
          <div>
            <strong>{{ item.name }}</strong
            ><span class="muted">{{
              item.type === "remote" ? item.url : "本地 JSON 配置"
            }}</span>
          </div>
          <div class="subscription-row__actions">
            <MiuixButton
              v-if="item.type === 'local'"
              type="secondary"
              :disabled="Boolean(busy) || creating"
              @click="openEditor(item.name)"
              >编辑</MiuixButton
            ><MiuixButton
              v-if="item.type === 'remote'"
              type="secondary"
              :disabled="Boolean(busy) || creating"
              @click="action('update', item)"
              >{{
                busy === `update:${item.name}` ? "更新中…" : "更新"
              }}</MiuixButton
            ><MiuixButton
              v-if="item.name !== active"
              type="secondary"
              :disabled="Boolean(busy) || creating"
              @click="action('switch', item)"
              >{{
                busy === `switch:${item.name}` ? "切换中…" : "切换"
              }}</MiuixButton
            ><MiuixButton
              v-if="item.name !== active"
              type="secondary"
              :disabled="Boolean(busy) || creating"
              @click="action('remove', item)"
              >{{
                busy === `remove:${item.name}` ? "删除中…" : "删除"
              }}</MiuixButton
            >
          </div>
        </div></MiuixCard
      >
      <MiuixText v-if="error" class="section error" type="body2">{{
        error
      }}</MiuixText>
      <MiuixDialog
        v-model="dialogOpen"
        :title="dialogType === 'remote' ? '添加远程订阅' : '新建本地订阅'"
        ><template #default
          ><div class="subscription-dialog__fields">
            <MiuixInput v-model="name" label="名称" /><MiuixInput
              v-if="dialogType === 'remote'"
              v-model="url"
              label="订阅 URL"
            />
          </div>
          <div class="subscription-dialog__actions">
            <MiuixButton
              class="subscription-dialog__action"
              type="secondary"
              :disabled="creating"
              @click="closeDialog"
              >取消</MiuixButton
            ><MiuixButton
              class="subscription-dialog__action"
              :disabled="creating || !name || (dialogType === 'remote' && !url)"
              @click="create"
              >{{ creating ? "保存中…" : "保存" }}</MiuixButton
            >
          </div></template
        ></MiuixDialog
      >
    </div>
  </div>
</template>

<style scoped>
.editor-page {
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.editor-loading {
  flex: 1;
}
.subscription-add {
  margin-top: 12px;
}
.subscription-row {
  padding: 14px 16px;
  border-bottom: 1px solid var(--m-color-divider-line);
}
.subscription-row:last-child {
  border-bottom: 0;
}
.subscription-row > div:first-child {
  display: grid;
  gap: 4px;
  overflow-wrap: anywhere;
}
.subscription-row__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}
.subscription-dialog__fields {
  display: grid;
  gap: 16px;
}
.subscription-dialog__actions {
  display: flex;
  gap: 12px;
  margin-top: 24px;
}
.subscription-dialog__action {
  flex: 1;
}
</style>
