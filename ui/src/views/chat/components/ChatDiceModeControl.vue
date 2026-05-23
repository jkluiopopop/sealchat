<script setup lang="ts">
import { computed } from 'vue'
import { Settings } from '@vicons/ionicons5'

interface SelectOption {
  label: string
  value: string
}

interface Props {
  visible: boolean
  showStatus?: boolean
  showSettings?: boolean
  isMobile?: boolean
  modeLabel: string
  modeTooltip: string
  builtInDiceEnabled: boolean
  botFeatureEnabled: boolean
  diceFeatureUpdating?: boolean
  channelBotSelection: string
  botSelectOptions: SelectOption[]
  botOptionsLoading?: boolean
  channelBotsLoading?: boolean
  syncingChannelBot?: boolean
  hasBotOptions?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showStatus: false,
  showSettings: false,
  isMobile: false,
  diceFeatureUpdating: false,
  botOptionsLoading: false,
  channelBotsLoading: false,
  syncingChannelBot: false,
  hasBotOptions: false,
})

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'toggle-built-in', value: boolean): void
  (e: 'toggle-bot', value: boolean): void
  (e: 'select-bot', value: string | null): void
  (e: 'open-channel-member-settings'): void
}>()

const panelVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const handleOpenChannelMemberSettings = () => {
  panelVisible.value = false
  emit('open-channel-member-settings')
}
</script>

<template>
  <template v-if="props.showStatus">
    <template v-if="props.isMobile">
      <n-tooltip trigger="hover">
        <template #trigger>
          <div class="dice-mode-status">
            <span class="dice-mode-status__label">{{ props.modeLabel }}</span>
            <n-button
              v-if="props.showSettings"
              quaternary
              size="tiny"
              circle
              class="dice-tray-settings-trigger"
              :class="{ 'dice-tray-settings-trigger--active': panelVisible }"
              @click.stop="panelVisible = true"
            >
              <n-icon :component="Settings" size="14" />
            </n-button>
          </div>
        </template>
        {{ props.modeTooltip }}
      </n-tooltip>
      <n-modal
        v-if="props.showSettings"
        v-model:show="panelVisible"
        preset="card"
        class="dice-settings-modal-mobile"
        :mask-closable="true"
        :closable="false"
        :bordered="false"
        title="掷骰设置"
      >
        <div class="dice-settings-panel dice-settings-panel--modal">
          <div class="dice-settings-panel__section">
            <div class="dice-settings-panel__row">
              <div>
                <p class="dice-settings-panel__title">内置骰点</p>
                <p class="dice-settings-panel__desc">自动解析输入并生成骰点结果。</p>
              </div>
              <n-switch
                size="small"
                :value="props.builtInDiceEnabled"
                :disabled="props.diceFeatureUpdating"
                @update:value="emit('toggle-built-in', $event)"
              />
            </div>
          </div>
          <div class="dice-settings-panel__section">
            <div class="dice-settings-panel__row">
              <div>
                <p class="dice-settings-panel__title">机器人骰点</p>
                <p class="dice-settings-panel__desc">交由机器人处理掷骰，避免与内置功能冲突。</p>
              </div>
              <n-switch
                size="small"
                :value="props.botFeatureEnabled"
                :disabled="props.diceFeatureUpdating"
                @update:value="emit('toggle-bot', $event)"
              />
            </div>
            <div v-if="props.botFeatureEnabled" class="dice-settings-panel__body">
              <n-select
                :value="props.channelBotSelection"
                class="dice-settings-panel__select"
                :options="props.botSelectOptions"
                :loading="props.botOptionsLoading || props.channelBotsLoading || props.syncingChannelBot"
                :disabled="props.syncingChannelBot || !props.hasBotOptions"
                placeholder="选择主控 BOT（不会移除其他已绑定 BOT）"
                clearable
                @update:value="emit('select-bot', $event)"
              />
              <div class="dice-settings-panel__hint" v-if="!props.botOptionsLoading && !props.hasBotOptions">
                暂无可用机器人，请先在后台创建令牌。
              </div>
              <div class="dice-settings-panel__hint" v-else>
                这里只切换主控 BOT；如需绑定多个 BOT，请前往频道设置。
              </div>
            </div>
            <div class="dice-settings-panel__footer">
              <n-button text size="tiny" @click="handleOpenChannelMemberSettings">前往成员管理</n-button>
            </div>
          </div>
        </div>
      </n-modal>
    </template>
    <template v-else>
      <n-popover
        v-if="props.showSettings"
        trigger="manual"
        placement="bottom-end"
        :show="panelVisible"
        @clickoutside="panelVisible = false"
      >
        <template #trigger>
          <n-tooltip trigger="hover">
            <template #trigger>
              <div class="dice-mode-status">
                <span class="dice-mode-status__label">{{ props.modeLabel }}</span>
                <n-button
                  quaternary
                  size="tiny"
                  circle
                  class="dice-tray-settings-trigger"
                  :class="{ 'dice-tray-settings-trigger--active': panelVisible }"
                  @click.stop="panelVisible = !panelVisible"
                >
                  <n-icon :component="Settings" size="14" />
                </n-button>
              </div>
            </template>
            {{ props.modeTooltip }}
          </n-tooltip>
        </template>
        <div class="dice-settings-panel">
          <div class="dice-settings-panel__section">
            <div class="dice-settings-panel__row">
              <div>
                <p class="dice-settings-panel__title">内置骰点</p>
                <p class="dice-settings-panel__desc">自动解析输入并生成骰点结果。</p>
              </div>
              <n-switch
                size="small"
                :value="props.builtInDiceEnabled"
                :disabled="props.diceFeatureUpdating"
                @update:value="emit('toggle-built-in', $event)"
              />
            </div>
          </div>
          <div class="dice-settings-panel__section">
            <div class="dice-settings-panel__row">
              <div>
                <p class="dice-settings-panel__title">机器人骰点</p>
                <p class="dice-settings-panel__desc">交由机器人处理掷骰，避免与内置功能冲突。</p>
              </div>
              <n-switch
                size="small"
                :value="props.botFeatureEnabled"
                :disabled="props.diceFeatureUpdating"
                @update:value="emit('toggle-bot', $event)"
              />
            </div>
            <div v-if="props.botFeatureEnabled" class="dice-settings-panel__body">
              <n-select
                :value="props.channelBotSelection"
                class="dice-settings-panel__select"
                :options="props.botSelectOptions"
                :loading="props.botOptionsLoading || props.channelBotsLoading || props.syncingChannelBot"
                :disabled="props.syncingChannelBot || !props.hasBotOptions"
                placeholder="选择主控 BOT（不会移除其他已绑定 BOT）"
                clearable
                @update:value="emit('select-bot', $event)"
              />
              <div class="dice-settings-panel__hint" v-if="!props.botOptionsLoading && !props.hasBotOptions">
                暂无可用机器人，请先在后台创建令牌。
              </div>
              <div class="dice-settings-panel__hint" v-else>
                这里只切换主控 BOT；如需绑定多个 BOT，请前往频道设置。
              </div>
            </div>
            <div class="dice-settings-panel__footer">
              <n-button text size="tiny" @click="handleOpenChannelMemberSettings">前往成员管理</n-button>
            </div>
          </div>
        </div>
      </n-popover>
      <n-tooltip v-else trigger="hover">
        <template #trigger>
          <div class="dice-mode-status">
            <span class="dice-mode-status__label">{{ props.modeLabel }}</span>
          </div>
        </template>
        {{ props.modeTooltip }}
      </n-tooltip>
    </template>
  </template>
</template>

<style scoped>
.dice-mode-status {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  cursor: pointer;
}

.dice-mode-status__label {
  font-size: 11px;
  color: var(--sc-text-tertiary, #94a3b8);
  white-space: nowrap;
}

:root[data-display-palette='night'] .dice-mode-status__label {
  color: rgba(148, 163, 184, 0.85);
}

.dice-tray-settings-trigger {
  width: 1.5rem;
  height: 1.5rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: var(--sc-text-secondary);
  border: 1px solid transparent;
  transition: color 0.15s ease, border-color 0.15s ease, background-color 0.15s ease;
}

:root[data-display-palette='night'] .dice-tray-settings-trigger {
  color: rgba(226, 232, 240, 0.8);
}

.dice-tray-settings-trigger--active {
  color: var(--sc-primary-color, #2563eb);
  border-color: rgba(37, 99, 235, 0.4);
  background-color: rgba(37, 99, 235, 0.08);
}

:root[data-display-palette='night'] .dice-tray-settings-trigger--active {
  color: rgba(147, 197, 253, 0.95);
  border-color: rgba(147, 197, 253, 0.35);
  background-color: rgba(59, 130, 246, 0.18);
}

.dice-settings-panel {
  min-width: 260px;
  max-width: 320px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.dice-settings-panel--modal {
  min-width: 0;
  width: 100%;
  max-width: 100%;
  padding-right: 0;
}

.dice-settings-panel--modal .dice-settings-panel__section {
  padding-right: 0;
}

.dice-settings-panel--modal .dice-settings-panel__footer {
  padding-right: 0;
}

.dice-settings-modal-mobile :deep(.n-card) {
  width: min(360px, 92vw);
}

.dice-settings-modal-mobile :deep(.n-card__content) {
  padding-top: 0;
  max-height: min(70vh, 520px);
  overflow-y: auto;
}

.dice-settings-panel__section {
  border: 1px solid var(--sc-border-strong);
  border-radius: 0.75rem;
  padding: 0.65rem 0.75rem;
  background-color: var(--sc-bg-elevated);
}

.dice-settings-panel__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.dice-settings-panel__title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--sc-text-primary);
  margin: 0;
}

.dice-settings-panel__desc {
  font-size: 0.75rem;
  color: var(--sc-text-secondary);
  margin: 0.1rem 0 0;
}

.dice-settings-panel__body {
  margin-top: 0.65rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.dice-settings-panel__select {
  width: 100%;
}

.dice-settings-panel__hint {
  font-size: 0.75rem;
  color: var(--sc-text-secondary);
}

.dice-settings-panel__footer {
  margin-top: 0.35rem;
  display: flex;
  justify-content: flex-end;
}
</style>
