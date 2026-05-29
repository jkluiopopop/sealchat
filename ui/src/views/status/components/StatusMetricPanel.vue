<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue';
import { type StatusHistoryPoint, type StatusMetricDefinition } from '../status-history';

const props = defineProps<{
  metric: StatusMetricDefinition;
  points: StatusHistoryPoint[];
  loading: boolean;
  expanded?: boolean;
  overlay?: boolean;
}>();

const emit = defineEmits<{
  (e: 'toggle-expand'): void;
}>();

const StatusMetricChart = defineAsyncComponent(() => import('./StatusMetricChart.vue'));

const chartAnchorRef = ref<HTMLDivElement | null>(null);
const chartVisible = ref(false);
let observer: IntersectionObserver | null = null;
const latestValueText = computed(() => {
  const latestPoint = props.points[props.points.length - 1];
  if (props.metric.series?.length) {
    return props.metric.series
      .map((item) => `${item.label} ${props.metric.format(Number(latestPoint?.[item.key] || 0))}`)
      .join(' / ');
  }
  return props.metric.format(latestPoint?.[props.metric.key] || 0);
});

const overlayCardStyle = computed(() => (props.overlay ? {
  width: '100vw',
  maxWidth: '100vw',
  minHeight: '100vh',
  height: '100vh',
  borderRadius: '0',
  borderLeft: 'none',
  borderRight: 'none',
  borderTop: 'none',
} : undefined));

const overlayContentStyle = computed(() => (
  props.overlay
    ? 'height:100%;display:flex;flex-direction:column;padding:18px 22px 22px;'
    : undefined
));

const overlayChartShellStyle = computed(() => (props.overlay ? {
  flex: '1 1 auto',
  minHeight: 'calc(100vh - 112px)',
  height: 'calc(100vh - 112px)',
  borderRadius: '1rem',
} : undefined));

onMounted(() => {
  if (props.overlay || typeof IntersectionObserver === 'undefined' || !chartAnchorRef.value) {
    chartVisible.value = true;
    return;
  }
  observer = new IntersectionObserver((entries) => {
    if (entries.some((entry) => entry.isIntersecting)) {
      chartVisible.value = true;
      observer?.disconnect();
      observer = null;
    }
  }, {
    rootMargin: '180px 0px',
  });
  observer.observe(chartAnchorRef.value);
});

onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
});
</script>

<template>
  <n-card
    class="status-metric-panel"
    :class="{
      'status-metric-panel--expanded': expanded,
      'status-metric-panel--overlay': overlay,
    }"
    :style="overlayCardStyle"
    :content-style="overlayContentStyle"
    size="small"
  >
    <div class="status-metric-panel__header">
      <div class="status-metric-panel__title-block">
        <strong>{{ metric.label }}</strong>
        <span>最新值：{{ latestValueText }}</span>
      </div>
      <button
        type="button"
        class="status-metric-panel__expand-button"
        :class="{ 'is-expanded': expanded }"
        :aria-label="expanded ? `收起${metric.label}图表` : `展开${metric.label}图表`"
        @click="emit('toggle-expand')"
      >
        +
      </button>
    </div>

    <n-spin :show="loading">
      <div v-if="points.length" class="status-metric-panel__body">
        <div
          ref="chartAnchorRef"
          class="status-metric-panel__chart-shell"
          :class="{
            'is-expanded': expanded,
            'is-overlay': overlay,
          }"
          :style="overlayChartShellStyle"
        >
          <component
            :is="StatusMetricChart"
            v-if="chartVisible"
            :metric="metric"
            :points="points"
            :expanded="expanded"
            :overlay="overlay"
          />
          <div v-else class="status-metric-panel__chart-placeholder">图表懒加载中...</div>
        </div>
      </div>

      <div v-else class="status-metric-panel__empty" role="status">暂无历史数据</div>
    </n-spin>
  </n-card>
</template>

<style scoped lang="scss">
.status-metric-panel {
  width: 100%;
  height: 100%;
  border-radius: 1rem;
  border: 1px solid var(--sc-border-mute);
  background:
    linear-gradient(
      180deg,
      color-mix(in srgb, var(--sc-bg-elevated) 92%, var(--sc-bg-surface) 8%) 0%,
      color-mix(in srgb, var(--sc-bg-elevated) 70%, var(--sc-bg-surface) 30%) 100%
    );
  transition:
    transform 0.34s cubic-bezier(0.22, 1, 0.36, 1),
    box-shadow 0.34s cubic-bezier(0.22, 1, 0.36, 1),
    border-color 0.26s ease,
    background 0.26s ease;
}

.status-metric-panel--expanded {
  border-color: color-mix(in srgb, var(--sc-border-strong) 64%, transparent);
  box-shadow:
    0 22px 40px color-mix(in srgb, var(--sc-border-strong) 18%, transparent),
    inset 0 1px 0 color-mix(in srgb, var(--sc-text-primary) 8%, transparent);
  transform: translateY(-2px);
}

.status-metric-panel--overlay {
  width: 100%;
  height: 100%;
  border-radius: 0;
  border-left: none;
  border-right: none;
  border-top: none;
  box-shadow:
    0 30px 60px color-mix(in srgb, black 38%, transparent),
    inset 0 1px 0 color-mix(in srgb, var(--sc-text-primary) 8%, transparent);
}

.status-metric-panel--overlay :deep(.n-card__content) {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 18px 22px 22px;
}

.status-metric-panel :deep(.n-card__content),
.status-metric-panel :deep(.n-spin-container),
.status-metric-panel :deep(.n-spin-content) {
  display: block;
  width: 100%;
  min-width: 0;
}

.status-metric-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.status-metric-panel__title-block {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.status-metric-panel__title-block strong {
  color: var(--sc-text-primary);
  font-size: 0.98rem;
}

.status-metric-panel__title-block span {
  color: var(--sc-text-secondary);
  font-size: 0.76rem;
  line-height: 1.4;
}

.status-metric-panel__expand-button {
  width: 1.9rem;
  height: 1.9rem;
  flex: 0 0 auto;
  border: 1px solid color-mix(in srgb, var(--sc-border-strong) 36%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--sc-bg-surface) 30%, transparent);
  color: var(--sc-text-primary);
  font-size: 1.15rem;
  line-height: 1;
  font-weight: 600;
  cursor: pointer;
  transition:
    transform 0.3s cubic-bezier(0.22, 1, 0.36, 1),
    background 0.22s ease,
    border-color 0.22s ease,
    box-shadow 0.22s ease;
}

.status-metric-panel__expand-button:hover {
  border-color: color-mix(in srgb, var(--sc-border-strong) 62%, transparent);
  background: color-mix(in srgb, var(--sc-bg-surface) 48%, transparent);
  box-shadow: 0 8px 16px color-mix(in srgb, var(--sc-border-strong) 12%, transparent);
}

.status-metric-panel__expand-button.is-expanded {
  transform: rotate(45deg) scale(1.05);
  border-color: color-mix(in srgb, var(--sc-border-strong) 74%, transparent);
  background: color-mix(in srgb, var(--sc-bg-surface) 56%, transparent);
}

.status-metric-panel__body {
  display: flex;
  flex-direction: column;
  gap: 0;
  width: 100%;
  min-width: 0;
}

.status-metric-panel--overlay .status-metric-panel__body,
.status-metric-panel--overlay :deep(.n-spin-container),
.status-metric-panel--overlay :deep(.n-spin-content) {
  flex: 1 1 auto;
}

.status-metric-panel__chart-shell {
  width: 100%;
  max-width: 100%;
  min-height: 248px;
  border-radius: 0.9rem;
  border: 1px solid color-mix(in srgb, var(--sc-border-mute) 78%, transparent);
  background: color-mix(in srgb, var(--sc-bg-surface) 22%, transparent);
  overflow: hidden;
  box-sizing: border-box;
  transition:
    min-height 0.34s cubic-bezier(0.22, 1, 0.36, 1),
    border-color 0.22s ease,
    background 0.22s ease;
}

.status-metric-panel__chart-shell.is-expanded {
  min-height: 340px;
  border-color: color-mix(in srgb, var(--sc-border-strong) 48%, transparent);
  background: color-mix(in srgb, var(--sc-bg-surface) 28%, transparent);
}

.status-metric-panel__chart-shell.is-overlay {
  flex: 1 1 auto;
  min-height: calc(100vh - 112px);
  height: calc(100vh - 112px);
  border-radius: 1rem;
}

.status-metric-panel__chart-placeholder,
.status-metric-panel__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 248px;
  color: var(--sc-text-secondary);
  font-size: 0.85rem;
}

.status-metric-panel__empty {
  min-height: 160px;
}

@media (max-width: 760px) {
  .status-metric-panel__header {
    flex-direction: column;
  }
}
</style>
