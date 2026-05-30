<script setup lang="ts">
const mode = defineModel<'builtin' | 'bot' | 'disabled'>('mode', { default: 'builtin' });
const botId = defineModel<string>('botId', { default: '' });
const botIds = defineModel<string[]>('botIds', { default: [] });
const eventBotIds = defineModel<string[]>('eventBotIds', { default: [] });

import { computed, watch } from 'vue';

const props = defineProps<{
  botOptionsLoading?: boolean;
  botSelectOptions: Array<{ label: string; value: string }>;
  disabled?: boolean;
  title?: string;
  hint?: string;
  botPlaceholder?: string;
}>();

const diceModeOptions = [
  { label: '内置掷骰', value: 'builtin' },
  { label: 'BOT掷骰', value: 'bot' },
  { label: '关闭掷骰', value: 'disabled' },
] as const;

const primaryBotSelectOptions = computed(() => {
  const selected = new Set(botIds.value);
  return props.botSelectOptions.filter((item) => selected.has(String(item.value || '')));
});

const eventBotSelectOptions = computed(() => {
  const selected = new Set(botIds.value);
  return props.botSelectOptions.filter((item) => selected.has(String(item.value || '')));
});

watch(botIds, (nextBotIds) => {
  const normalized = Array.from(new Set((nextBotIds || []).map((id) => String(id || '').trim()).filter(Boolean)));
  if (normalized.length !== (nextBotIds || []).length || normalized.some((id, index) => id !== nextBotIds[index])) {
    botIds.value = normalized;
    return;
  }
  if (botId.value && !normalized.includes(botId.value)) {
    botId.value = normalized[0] || '';
  } else if (!botId.value && normalized.length > 0) {
    botId.value = normalized[0];
  }
  eventBotIds.value = (eventBotIds.value || []).filter((id) => normalized.includes(id));
});

watch(botId, (nextBotId) => {
  const normalized = String(nextBotId || '').trim();
  if (normalized !== nextBotId) {
    botId.value = normalized;
    return;
  }
  if (!normalized) {
    return;
  }
  if (!botIds.value.includes(normalized)) {
    botIds.value = [...botIds.value, normalized];
  }
});
</script>

<template>
  <div class="dice-default-fields">
    <div class="dice-default-fields__block">
      <div v-if="title" class="dice-default-fields__title">{{ title }}</div>
      <n-radio-group v-model:value="mode" :disabled="props.disabled">
        <n-space>
          <n-radio
            v-for="item in diceModeOptions"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </n-radio>
        </n-space>
      </n-radio-group>
    </div>
    <div v-if="mode === 'bot'" class="dice-default-fields__bot-config">
      <div class="dice-default-fields__intro">
        BOT 掷骰分 3 步：先绑定 BOT，再选主控 BOT，最后选择接收频道事件的 BOT。
      </div>
      <div class="dice-default-fields__field">
        <div class="dice-default-fields__field-title">1. 频道已绑定 BOT</div>
        <div class="dice-default-fields__field-desc">新建频道启用 BOT 掷骰后，这些 BOT 会自动加入频道。至少选择 1 个。</div>
        <n-select
          v-model:value="botIds"
          :options="botSelectOptions"
          :loading="botOptionsLoading"
          :disabled="props.disabled"
          :placeholder="botPlaceholder || '选择要绑定到新频道的 BOT'"
          clearable
          multiple
        />
      </div>
      <template v-if="botIds.length > 0">
        <div class="dice-default-fields__field">
          <div class="dice-default-fields__field-title">2. 主控 BOT</div>
          <div class="dice-default-fields__field-desc">负责处理掷骰命令，以及频道里的角色卡相关能力。</div>
          <n-select
            v-model:value="botId"
            :options="primaryBotSelectOptions"
            :loading="botOptionsLoading"
            :disabled="props.disabled"
            placeholder="选择新频道默认主控 BOT"
            clearable
          />
        </div>
        <div class="dice-default-fields__field">
          <div class="dice-default-fields__field-title">3. 接收频道事件的 BOT</div>
          <div class="dice-default-fields__field-desc">决定哪些 BOT 会收到频道事件。留空时，默认全部已绑定 BOT 都会接收。</div>
          <n-select
            v-model:value="eventBotIds"
            :options="eventBotSelectOptions"
            :loading="botOptionsLoading"
            :disabled="props.disabled"
            placeholder="选择接收频道事件的 BOT"
            clearable
            multiple
          />
        </div>
      </template>
    </div>
    <div v-else-if="mode === 'disabled'" class="dice-default-fields__hint">
      新建频道会关闭所有掷骰，输入栏不会显示掷骰入口。
    </div>
    <div v-else class="dice-default-fields__hint">
      新建频道默认使用内置掷骰。
    </div>
    <div v-if="hint" class="dice-default-fields__hint">{{ hint }}</div>
  </div>
</template>

<style scoped>
.dice-default-fields {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dice-default-fields__block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.dice-default-fields__title {
  color: var(--sc-text-primary);
  font-size: 13px;
  font-weight: 600;
}

.dice-default-fields__hint {
  color: var(--sc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.dice-default-fields__bot-config {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dice-default-fields__intro {
  color: var(--sc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.dice-default-fields__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.dice-default-fields__field-title {
  color: var(--sc-text-primary);
  font-size: 13px;
  font-weight: 600;
}

.dice-default-fields__field-desc {
  color: var(--sc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}
</style>
