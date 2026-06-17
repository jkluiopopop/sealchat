<script setup lang="ts">
import dayjs from 'dayjs'
import { computed, onMounted, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useBattleReportStore } from '@/stores/battleReport'
import { copyTextWithFallback } from '@/utils/clipboard'
import { chatEvent } from '@/stores/chat'

interface Props {
  reportId: string
  rawLink?: string
}

const props = defineProps<Props>()
const store = useBattleReportStore()
const message = useMessage()
const loading = ref(false)
const failed = ref('')
const expanded = ref(false)
const report = computed(() => store.detailById[props.reportId])
const contentText = computed(() => report.value?.content || report.value?.contentPreview || '暂无内容')
const isLongContent = computed(() => contentText.value.length > 800 || contentText.value.split('\n').length > 16)
const periodText = computed(() => {
  const item = report.value
  if (!item?.periodStart || !item?.periodEnd) return ''
  return `${dayjs(item.periodStart).format('YYYY-MM-DD HH:mm')} - ${dayjs(item.periodEnd).format('YYYY-MM-DD HH:mm')}`
})

const load = async () => {
  if (!props.reportId) return
  loading.value = true
  failed.value = ''
  try {
    await store.get(props.reportId)
  } catch (error: any) {
    failed.value = error?.response?.data?.message || error?.message || '加载战报失败'
  } finally {
    loading.value = false
  }
}

const copyLink = async () => {
  if (!props.rawLink) return
  await copyTextWithFallback(props.rawLink)
  message.success('战报链接已复制')
}

const openEditor = (event?: MouseEvent) => {
  event?.preventDefault()
  event?.stopPropagation()
  chatEvent.emit('battle-report-open-editor' as any, {
    reportId: props.reportId,
    channelId: report.value?.channelId,
  })
}

onMounted(load)
watch(() => props.reportId, load)
</script>

<template>
  <div class="battle-report-embed-card" @dblclick="openEditor">
    <div class="battle-report-embed-card__head">
      <div>
        <div class="battle-report-embed-card__eyebrow">战报总结</div>
        <button class="battle-report-embed-card__title" type="button" @click="openEditor">
          {{ report?.title || (loading ? '加载中...' : '战报') }}
        </button>
        <div v-if="periodText" class="battle-report-embed-card__period">{{ periodText }}</div>
      </div>
      <n-button v-if="rawLink" quaternary circle size="tiny" title="复制链接" @click.stop="copyLink">⧉</n-button>
    </div>
    <n-spin :show="loading">
      <div v-if="failed" class="battle-report-embed-card__error">{{ failed }}</div>
      <div v-else-if="report?.status === 'generating'" class="battle-report-embed-card__hint">
        AI 总结生成中。
      </div>
      <div v-else-if="report?.status === 'failed'" class="battle-report-embed-card__error">
        {{ report.errorMessage || '生成失败' }}
      </div>
      <div
        v-else
        class="battle-report-embed-card__content"
        :class="{ 'battle-report-embed-card__content--collapsed': isLongContent && !expanded }"
      >
        {{ contentText }}
      </div>
      <n-button v-if="isLongContent" text size="small" class="battle-report-embed-card__expand" @click.stop="expanded = !expanded">
        {{ expanded ? '收起' : '展开完整战报' }}
      </n-button>
    </n-spin>
  </div>
</template>

<style scoped>
.battle-report-embed-card {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 16px;
  padding: 14px;
  background:
    radial-gradient(circle at top left, rgba(37, 99, 235, 0.12), transparent 42%),
    rgba(148, 163, 184, 0.08);
  color: var(--text-color-1);
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.08);
}

.battle-report-embed-card__head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.battle-report-embed-card__eyebrow {
  font-size: 12px;
  letter-spacing: 0.08em;
  color: var(--text-color-3);
}

.battle-report-embed-card__title {
  display: block;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--text-color-1);
  font-size: 16px;
  font-weight: 800;
  text-align: left;
  cursor: pointer;
}

.battle-report-embed-card__title:hover {
  color: var(--primary-color);
}

.battle-report-embed-card__period {
  margin-top: 2px;
  font-size: 12px;
  color: var(--text-color-3);
}

.battle-report-embed-card__content,
.battle-report-embed-card__hint,
.battle-report-embed-card__error {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
}

.battle-report-embed-card__content--collapsed {
  max-height: 320px;
  overflow: hidden;
  mask-image: linear-gradient(to bottom, #000 74%, transparent);
}

.battle-report-embed-card__expand {
  margin-top: 8px;
}

.battle-report-embed-card__hint {
  color: var(--text-color-3);
}

.battle-report-embed-card__error {
  color: #dc2626;
}
</style>
