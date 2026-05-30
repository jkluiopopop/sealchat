<script setup lang="ts">
import { ref } from 'vue';
import { useMessage } from 'naive-ui';
import ChatInputRich from '@/views/chat/components/inputs/ChatInputRich.vue';
import { uploadImageAttachment } from '@/views/chat/composables/useAttachmentUploader';

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  maxlength?: number
}>(), {
  modelValue: '',
  placeholder: '用于聊天中的提示和解释（支持富文本格式）',
  maxlength: 2000,
});

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
}>();

const message = useMessage();
const richEditorRef = ref<InstanceType<typeof ChatInputRich> | null>(null);
const fileInputRef = ref<HTMLInputElement | null>(null);
const isUploading = ref(false);

const updateValue = (...args: unknown[]) => {
  const value = typeof args[0] === 'string' ? args[0] : '';
  emit('update:modelValue', value);
};

const insertImageFiles = async (files: File[]) => {
  if (isUploading.value) return;
  const imageFiles = files.filter((file) => file.type.startsWith('image/'));
  if (!imageFiles.length) {
    message.warning('当前仅支持插入图片文件');
    return;
  }

  const editor = richEditorRef.value?.getEditor?.();
  if (!editor) return;

  isUploading.value = true;
  try {
    for (const file of imageFiles) {
      const result = await uploadImageAttachment(file);
      if (!result.attachmentId) continue;
      const attachmentId = result.attachmentId.replace(/^id:/, '');
      const imageUrl = `/api/v1/attachment/${attachmentId}`;
      editor.chain().focus().setImage({ src: imageUrl, alt: file.name || '' }).run();
    }
  } catch (error: any) {
    message.error(error?.message || '图片上传失败');
  } finally {
    isUploading.value = false;
  }
};

const triggerFileSelect = () => {
  fileInputRef.value?.click();
};

const handleFileSelect = (event: Event) => {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files || []);
  if (files.length) {
    void insertImageFiles(files);
  }
  input.value = '';
};

const focus = () => {
  richEditorRef.value?.focus();
};

defineExpose({
  focus,
  getEditor: () => richEditorRef.value?.getEditor?.(),
  getJson: () => richEditorRef.value?.getJson?.(),
  triggerFileSelect,
  hasOpenOverlay: () => richEditorRef.value?.hasOpenOverlay?.() ?? false,
  hasRecentOverlayInteraction: (thresholdMs?: number) =>
    richEditorRef.value?.hasRecentOverlayInteraction?.(thresholdMs) ?? false,
});
</script>

<template>
  <div class="keyword-rich-editor">
    <input
      ref="fileInputRef"
      type="file"
      accept="image/*"
      multiple
      class="keyword-rich-editor__file-input"
      @change="handleFileSelect"
    />
    <ChatInputRich
      ref="richEditorRef"
      :model-value="props.modelValue"
      :placeholder="props.placeholder"
      :input-class="['keyword-rich-editor__input']"
      @update:model-value="updateValue"
      @paste-image="insertImageFiles($event.files)"
      @drop-files="insertImageFiles($event.files)"
      @upload-button-click="triggerFileSelect"
    />
  </div>
</template>

<style scoped>
.keyword-rich-editor {
  width: 100%;
}

.keyword-rich-editor__file-input {
  display: none;
}

:deep(.keyword-rich-editor__input) {
  min-height: 220px;
}

:deep(.keyword-rich-editor__input .tiptap-wrapper) {
  min-height: 220px;
}
</style>
