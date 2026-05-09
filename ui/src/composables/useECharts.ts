import type { Ref } from 'vue';
import { onBeforeUnmount } from 'vue';
import type * as echarts from 'echarts/core';
import { normalizeChartThemeMode } from './echarts-theme';

type EChartsModule = typeof echarts;
type EChartsInstance = echarts.ECharts;

export function useECharts(
  elRef: Ref<HTMLDivElement | null>,
  isDark: Ref<boolean>,
  echartsModule: EChartsModule,
) {
  let instance: EChartsInstance | null = null;
  let themeMode = normalizeChartThemeMode(isDark.value);

  const dispose = () => {
    if (instance) {
      instance.dispose();
      instance = null;
    }
  };

  const ensureInstance = () => {
    if (!elRef.value) {
      return null;
    }
    const nextThemeMode = normalizeChartThemeMode(isDark.value);
    if (!instance || themeMode !== nextThemeMode) {
      dispose();
      instance = echartsModule.init(elRef.value, nextThemeMode);
      themeMode = nextThemeMode;
    }
    return instance;
  };

  const resize = () => {
    instance?.resize();
  };

  onBeforeUnmount(() => {
    dispose();
  });

  return {
    dispose,
    ensureInstance,
    resize,
  };
}
