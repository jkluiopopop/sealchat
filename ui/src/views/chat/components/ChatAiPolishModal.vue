<script setup lang="ts">
const props = defineProps<{
  visible: boolean
  loading?: boolean
  sourceText: string
  resultText: string
}>()

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
  (event: 'update:resultText', value: string): void
  (event: 'retry'): void
  (event: 'apply'): void
}>()
</script>

<template>
  <n-modal
    :show="props.visible"
    preset="card"
    title="AI 润色"
    class="sc-fluid-modal sc-fluid-modal--wide"
    :auto-focus="false"
    @update:show="emit('update:visible', $event)"
  >
    <n-form label-placement="top">
      <n-form-item label="原文">
        <n-input :value="sourceText" type="textarea" :rows="4" readonly />
      </n-form-item>
      <n-form-item label="润色结果">
        <n-spin :show="props.loading" class="chat-ai-polish-modal__result-spin">
          <n-input
            :value="resultText"
            type="textarea"
            :rows="8"
            :readonly="props.loading"
            :placeholder="props.loading ? 'AI 正在润色，请稍候…' : '润色结果将显示在这里'"
            @update:value="emit('update:resultText', $event)"
          />
        </n-spin>
      </n-form-item>
    </n-form>

    <template #footer>
      <n-space justify="end">
        <n-button @click="emit('update:visible', false)">取消</n-button>
        <n-button tertiary :loading="props.loading" @click="emit('retry')">重新生成</n-button>
        <n-button type="primary" :disabled="!props.resultText.trim()" @click="emit('apply')">覆盖输入</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.chat-ai-polish-modal__result-spin {
  display: block;
  width: 100%;
}

.chat-ai-polish-modal__result-spin :deep(.n-spin-container),
.chat-ai-polish-modal__result-spin :deep(.n-spin-content) {
  width: 100%;
}
</style>
