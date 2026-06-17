<script setup lang="ts">
import dayjs from 'dayjs'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useBattleReportStore } from '@/stores/battleReport'
import { useAIStore } from '@/stores/ai'
import { chatEvent } from '@/stores/chat'
import type { BattleReport } from '@/types'
import { copyTextWithFallback } from '@/utils/clipboard'
import { generateBattleReportEmbedLink } from '@/utils/battleReportEmbedLink'
import ActiveDayDateRangePicker from './export/ActiveDayDateRangePicker.vue'
import BattleReportEditorModal from './BattleReportEditorModal.vue'

interface Props {
  visible: boolean
  channelId?: string
  worldId?: string
}

interface Emits {
  (e: 'update:visible', value: boolean): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const message = useMessage()
const store = useBattleReportStore()
const aiStore = useAIStore()

const createVisible = ref(false)
const editorVisible = ref(false)
const editingReportId = ref('')
const draggedId = ref('')
const createMode = ref<'manual' | 'ai'>('ai')
const createForm = reactive({
  title: '',
  content: '',
  period: null as [number, number] | null,
  contextReportCount: 3,
})
let pollTimer: number | null = null

const reports = computed(() => props.channelId ? (store.itemsByChannel[props.channelId] || []) : [])
const editingReport = computed(() => editingReportId.value ? store.detailById[editingReportId.value] : null)
const hasGenerating = computed(() => reports.value.some((item) => item.status === 'generating'))

const formatPeriod = (item: BattleReport) => {
  if (!item.periodStart || !item.periodEnd) return '未设置周期'
  return `${dayjs(item.periodStart).format('YYYY-MM-DD HH:mm')} - ${dayjs(item.periodEnd).format('YYYY-MM-DD HH:mm')}`
}

const previewText = (item: BattleReport) => (item.contentPreview || item.content || '暂无内容').slice(0, 200)

const resetCreateForm = () => {
  createMode.value = 'ai'
  createForm.title = ''
  createForm.content = ''
  createForm.period = null
  createForm.contextReportCount = 3
}

const refresh = async () => {
  if (!props.channelId) return
  try {
    await store.list(props.channelId)
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '加载战报失败')
  }
}

const stopPolling = () => {
  if (pollTimer === null) return
  window.clearInterval(pollTimer)
  pollTimer = null
}

const syncPolling = () => {
  stopPolling()
  if (!props.visible || !props.channelId || !hasGenerating.value) return
  pollTimer = window.setInterval(() => {
    void refresh()
  }, 2500)
}

watch(
  () => [props.visible, props.channelId] as const,
  ([visible, channelId]) => {
    if (visible && channelId) {
      void refresh()
    } else {
      stopPolling()
    }
  },
  { immediate: true },
)

watch(hasGenerating, syncPolling)

const openCreate = () => {
  resetCreateForm()
  createVisible.value = true
}

const openEditor = async (item: BattleReport) => {
  editingReportId.value = item.id
  try {
    await store.get(item.id)
  } catch (error) {
    console.warn('加载战报详情失败', error)
  }
  editorVisible.value = true
}

const openEditorById = async (reportId: string) => {
  const normalized = String(reportId || '').trim()
  if (!normalized) return
  emit('update:visible', true)
  editingReportId.value = normalized
  try {
    await store.get(normalized)
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '加载战报详情失败')
    return
  }
  editorVisible.value = true
}

const handleBattleReportOpenEditor = (payload: any) => {
  const reportId = String(payload?.reportId || '').trim()
  const channelId = String(payload?.channelId || '').trim()
  if (channelId && props.channelId && channelId !== props.channelId) {
    return
  }
  void openEditorById(reportId)
}

onMounted(() => {
  chatEvent.on('battle-report-open-editor' as any, handleBattleReportOpenEditor)
})

onBeforeUnmount(() => {
  stopPolling()
  chatEvent.off('battle-report-open-editor' as any, handleBattleReportOpenEditor)
})

const copyReportLink = async (item: BattleReport) => {
  const worldId = props.worldId || item.worldId
  if (!worldId || !item.channelId || !item.id) {
    message.error('缺少战报链接参数')
    return
  }
  const link = generateBattleReportEmbedLink({ worldId, channelId: item.channelId, reportId: item.id })
  await copyTextWithFallback(link)
  message.success('战报嵌入链接已复制')
}

const createReport = async () => {
  if (!props.channelId) {
    message.error('未选择频道')
    return
  }
  if (!createForm.period) {
    message.error('请选择战报时间周期')
    return
  }
  const payload = {
    title: createForm.title.trim() || '新战报',
    content: createForm.content.trim(),
    periodStart: createForm.period[0],
    periodEnd: createForm.period[1],
    contextReportCount: createForm.contextReportCount,
    source: aiStore.currentSource,
  }
  try {
    if (createMode.value === 'ai') {
      await store.summarize(props.channelId, payload)
      message.success('AI 总结已开始')
    } else {
      await store.create(props.channelId, payload)
      message.success('战报已创建')
    }
    createVisible.value = false
    await refresh()
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '创建战报失败')
  }
}

const saveEditor = async (payload: { title: string; content: string }) => {
  const item = editingReport.value
  if (!item) return
  try {
    await store.update(item.id, {
      ...payload,
      periodStart: item.periodStart,
      periodEnd: item.periodEnd,
      contextReportCount: item.contextReportCount,
    })
    editorVisible.value = false
    message.success('战报已保存')
    await refresh()
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '保存战报失败')
  }
}

const deleteReport = async (item: BattleReport) => {
  if (!window.confirm(`删除战报“${item.title}”？`)) return
  try {
    await store.delete(item.id)
    message.success('战报已删除')
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '删除战报失败')
  }
}

const handleDragStart = (item: BattleReport, event: DragEvent) => {
  draggedId.value = item.id
  event.dataTransfer?.setData('text/plain', item.id)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

const handleDrop = async (target: BattleReport, event: DragEvent) => {
  event.preventDefault()
  const sourceId = draggedId.value || event.dataTransfer?.getData('text/plain') || ''
  draggedId.value = ''
  if (!props.channelId || !sourceId || sourceId === target.id) return
  const current = reports.value.slice()
  const sourceIndex = current.findIndex((item) => item.id === sourceId)
  const targetIndex = current.findIndex((item) => item.id === target.id)
  if (sourceIndex < 0 || targetIndex < 0) return
  const next = current.slice()
  const [moved] = next.splice(sourceIndex, 1)
  next.splice(targetIndex, 0, moved)
  store.setChannelItems(props.channelId, next)
  try {
    await store.reorder(props.channelId, next.map((item) => item.id))
    await refresh()
  } catch (error: any) {
    store.setChannelItems(props.channelId, current)
    message.error(error?.response?.data?.message || error?.message || '战报排序失败')
  }
}
</script>

<template>
  <n-drawer
    :show="visible"
    :width="620"
    placement="right"
    @update:show="emit('update:visible', $event)"
  >
    <n-drawer-content title="战报总结" closable>
      <div class="battle-report-toolbar">
        <div>
          <div class="battle-report-toolbar__title">频道战报</div>
          <div class="battle-report-toolbar__hint">新建后可手写，或交给 AI 按时间周期总结。</div>
        </div>
        <n-button size="small" type="primary" @click="openCreate">新建战报</n-button>
      </div>
      <n-spin :show="store.loading">
        <div v-if="!reports.length" class="battle-report-empty">
          暂无战报。点击上方新建，手写或交给 AI 总结。
        </div>
        <div v-else class="battle-report-timeline">
          <div
            v-for="item in reports"
            :key="item.id"
            class="battle-report-item"
            :class="`battle-report-item--${item.status}`"
            draggable="true"
            @dragstart="handleDragStart(item, $event)"
            @dragover.prevent
            @drop="handleDrop(item, $event)"
          >
            <n-tooltip trigger="hover">
              <template #trigger>
                <div class="battle-report-node">
                  <span v-if="item.status === 'generating'" class="battle-report-node__spinner"></span>
                </div>
              </template>
              {{ formatPeriod(item) }}
            </n-tooltip>
            <div class="battle-report-card" @dblclick="openEditor(item)">
              <div class="battle-report-card__main">
                <n-popover trigger="hover" placement="left" :width="280">
                  <template #trigger>
                    <button class="battle-report-title" type="button" @click="openEditor(item)">
                      {{ item.title || '未命名战报' }}
                    </button>
                  </template>
                  <div class="battle-report-preview">{{ previewText(item) }}</div>
                </n-popover>
                <span class="battle-report-meta">{{ formatPeriod(item) }}</span>
                <span v-if="item.status === 'failed'" class="battle-report-error">
                  {{ item.errorMessage || '生成失败' }}
                </span>
              </div>
              <div class="battle-report-actions" @click.stop @dblclick.stop>
                <n-button quaternary circle size="tiny" title="编辑战报" @click="openEditor(item)">✎</n-button>
                <n-button quaternary circle size="tiny" title="复制嵌入链接" @click="copyReportLink(item)">⧉</n-button>
                <n-button quaternary circle size="tiny" title="删除" @click="deleteReport(item)">×</n-button>
              </div>
            </div>
          </div>
        </div>
      </n-spin>
    </n-drawer-content>
  </n-drawer>

  <n-modal
    v-model:show="createVisible"
    preset="card"
    title="新建战报"
    class="battle-report-create-modal"
    :auto-focus="false"
  >
    <n-form label-placement="top">
      <n-form-item label="生成方式">
        <n-radio-group v-model:value="createMode">
          <n-radio-button value="ai">AI 总结</n-radio-button>
          <n-radio-button value="manual">手动创建</n-radio-button>
        </n-radio-group>
      </n-form-item>
      <n-form-item label="时间周期">
        <ActiveDayDateRangePicker
          v-model="createForm.period"
          :channel-id="props.channelId"
          placeholder="选择需要总结的活跃消息周期"
        />
      </n-form-item>
      <n-form-item label="前情提要">
        <n-input-number v-model:value="createForm.contextReportCount" :min="0" :max="20" />
        <template #feedback>AI 总结时引用多少篇之前的已完成战报。</template>
      </n-form-item>
      <n-form-item label="标题">
        <n-input v-model:value="createForm.title" maxlength="120" show-count placeholder="留空则使用默认标题" />
      </n-form-item>
      <n-form-item v-if="createMode === 'manual'" label="内容">
        <n-input
          v-model:value="createForm.content"
          type="textarea"
          :autosize="{ minRows: 8, maxRows: 18 }"
          placeholder="纯文本战报内容"
        />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="createVisible = false">取消</n-button>
        <n-button type="primary" :loading="store.saving" @click="createReport">
          {{ createMode === 'ai' ? '开始总结' : '创建战报' }}
        </n-button>
      </n-space>
    </template>
  </n-modal>

  <BattleReportEditorModal
    v-model:visible="editorVisible"
    :report="editingReport"
    @save="saveEditor"
  />
</template>

<style scoped>
.battle-report-empty {
  padding: 28px 12px;
  color: var(--text-color-3);
  text-align: center;
}

.battle-report-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
  padding: 12px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 14px;
  background: rgba(148, 163, 184, 0.08);
}

.battle-report-toolbar__title {
  font-weight: 800;
  color: var(--text-color-1);
}

.battle-report-toolbar__hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--text-color-3);
}

.battle-report-timeline {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 0 12px;
}

.battle-report-item {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 10px;
  cursor: grab;
}

.battle-report-item:active {
  cursor: grabbing;
}

.battle-report-node {
  position: relative;
  width: 28px;
  min-height: 58px;
}

.battle-report-node::before {
  content: "";
  position: absolute;
  top: 0;
  bottom: -8px;
  left: 13px;
  width: 2px;
  background: rgba(100, 116, 139, 0.28);
}

.battle-report-node::after {
  content: "";
  position: absolute;
  top: 18px;
  left: 8px;
  width: 12px;
  height: 12px;
  border-radius: 999px;
  background: #2563eb;
  box-shadow: 0 0 0 4px rgba(37, 99, 235, 0.14);
}

.battle-report-item--failed .battle-report-node::after {
  background: #dc2626;
  box-shadow: 0 0 0 4px rgba(220, 38, 38, 0.14);
}

.battle-report-node__spinner {
  position: absolute;
  z-index: 1;
  top: 15px;
  left: 5px;
  width: 18px;
  height: 18px;
  border: 2px solid rgba(37, 99, 235, 0.2);
  border-top-color: #2563eb;
  border-radius: 999px;
  animation: battle-report-spin 0.9s linear infinite;
}

.battle-report-card {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  padding: 12px 12px;
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 14px;
  background: rgba(148, 163, 184, 0.08);
}

.battle-report-card__main {
  min-width: 0;
}

.battle-report-title {
  display: block;
  max-width: 100%;
  padding: 0;
  border: 0;
  color: var(--text-color-1);
  background: transparent;
  font-weight: 700;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.battle-report-title:hover {
  color: var(--primary-color);
}

.battle-report-meta,
.battle-report-error {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-color-3);
}

.battle-report-error {
  color: #dc2626;
}

.battle-report-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.battle-report-preview {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.55;
}

.battle-report-create-modal {
  width: min(720px, calc(100vw - 32px));
}

@keyframes battle-report-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 720px) {
  :deep(.n-drawer) {
    width: 100vw !important;
  }
}
</style>
