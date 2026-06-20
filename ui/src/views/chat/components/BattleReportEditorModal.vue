<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import type { BattleReport } from '@/types'

interface Props {
  visible: boolean
  report?: BattleReport | null
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'save', payload: { title: string; content: string }): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const form = reactive({
  title: '',
  content: '',
})
let loadedReportId = ''
let loadedSignature = ''

const isFailed = computed(() => props.report?.status === 'failed')
const isGenerating = computed(() => props.report?.status === 'generating')

watch(
  () => [
    props.visible,
    props.report?.id,
    props.report?.title,
    props.report?.content,
    props.report?.updatedAt,
  ] as const,
  () => {
    if (!props.visible) return
    const reportId = props.report?.id || ''
    const signature = [
      reportId,
      props.report?.title || '',
      props.report?.content || '',
      props.report?.updatedAt || 0,
    ].join('\u0001')
    if (reportId === loadedReportId && signature === loadedSignature) return
    form.title = props.report?.title || ''
    form.content = props.report?.content || ''
    loadedReportId = reportId
    loadedSignature = signature
  },
  { immediate: true },
)

const close = () => emit('update:visible', false)
const save = () => emit('save', {
  title: form.title.trim() || '未命名战报',
  content: form.content.trim(),
})
</script>

<template>
  <n-modal
    :show="visible"
    preset="card"
    class="battle-report-editor-modal"
    title="编辑战报"
    :auto-focus="false"
    @update:show="emit('update:visible', $event)"
  >
    <n-alert v-if="isFailed" type="error" :show-icon="false" class="battle-report-editor-modal__alert">
      {{ report?.errorMessage || '生成失败，可编辑后保存，或重新新建总结。' }}
    </n-alert>
    <n-alert v-else-if="isGenerating" type="info" :show-icon="false" class="battle-report-editor-modal__alert">
      AI 总结仍在生成中，完成后会自动填充内容。
    </n-alert>
    <n-form label-placement="top">
      <n-form-item label="标题">
        <n-input v-model:value="form.title" maxlength="120" show-count placeholder="战报标题" />
      </n-form-item>
      <n-form-item label="内容">
        <n-input
          v-model:value="form.content"
          type="textarea"
          :autosize="{ minRows: 18, maxRows: 36 }"
          placeholder="纯文本战报内容"
        />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="close">取消</n-button>
        <n-button type="primary" :disabled="isGenerating" @click="save">保存</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.battle-report-editor-modal {
  width: min(960px, calc(100vw - 32px));
}

.battle-report-editor-modal__alert {
  margin-bottom: 12px;
}

@media (max-width: 720px) {
  .battle-report-editor-modal {
    width: calc(100vw - 16px);
  }

  :deep(.n-card-header),
  :deep(.n-card__content),
  :deep(.n-card__footer) {
    padding-left: 14px;
    padding-right: 14px;
  }
}
</style>
