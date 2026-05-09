export type StatusRangeOption = '1h' | '6h' | '24h' | '7d';
export type StatusFilterMode = StatusRangeOption | 'custom';

export type StatusMetricKey =
  | 'concurrentConnections'
  | 'onlineUsers'
  | 'messagesPerMinute'
  | 'attachmentCount'
  | 'attachmentBytes';

export interface StatusHistoryPoint {
  timestamp: number;
  concurrentConnections: number;
  onlineUsers: number;
  messagesPerMinute: number;
  registeredUsers?: number;
  worldCount?: number;
  channelCount?: number;
  privateChannelCount?: number;
  messageCount?: number;
  attachmentCount: number;
  attachmentBytes: number;
}

export interface StatusMetricDefinition {
  key: StatusMetricKey;
  label: string;
  color: string;
  format: (value: number) => string;
}

export interface StatusMetricHistoryRow {
  timestamp: number;
  value: number;
  formattedValue: string;
}

export interface StatusHistoryRangePayload {
  range: StatusFilterMode | string;
  points: StatusHistoryPoint[];
}

export type StatusHistoryQuery =
  | { mode: StatusRangeOption }
  | { mode: 'custom'; start: number; end: number };

const numberFormatter = new Intl.NumberFormat('zh-CN');

export function formatStatusNumber(value?: number) {
  return numberFormatter.format(value || 0);
}

export function formatStatusBytes(value?: number) {
  const size = value || 0;
  if (size <= 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let cursor = size;
  let unitIndex = 0;
  while (cursor >= 1024 && unitIndex < units.length - 1) {
    cursor /= 1024;
    unitIndex += 1;
  }
  const precision = unitIndex === 0 ? 0 : cursor >= 100 ? 0 : cursor >= 10 ? 1 : 2;
  return `${cursor.toFixed(precision)} ${units[unitIndex]}`;
}

export const statusRangeOptions: Array<{ label: string; value: StatusRangeOption }> = [
  { label: '近 1 小时', value: '1h' },
  { label: '近 6 小时', value: '6h' },
  { label: '近 24 小时', value: '24h' },
  { label: '近 7 天', value: '7d' },
];

export const statusFilterOptions: Array<{ label: string; value: StatusFilterMode }> = [
  ...statusRangeOptions,
  { label: '自定义', value: 'custom' },
];

export const statusMetricDefinitions: Record<StatusMetricKey, StatusMetricDefinition> = {
  concurrentConnections: {
    key: 'concurrentConnections',
    label: '并发连接',
    color: '#2563eb',
    format: formatStatusNumber,
  },
  onlineUsers: {
    key: 'onlineUsers',
    label: '在线用户',
    color: '#059669',
    format: formatStatusNumber,
  },
  messagesPerMinute: {
    key: 'messagesPerMinute',
    label: '消息/分钟',
    color: '#f97316',
    format: formatStatusNumber,
  },
  attachmentCount: {
    key: 'attachmentCount',
    label: '附件数量',
    color: '#0ea5e9',
    format: formatStatusNumber,
  },
  attachmentBytes: {
    key: 'attachmentBytes',
    label: '附件总大小',
    color: '#ca8a04',
    format: formatStatusBytes,
  },
};

export const statusMetricList = Object.values(statusMetricDefinitions);

export function buildMetricHistoryRows(
  points: StatusHistoryPoint[],
  metric: StatusMetricDefinition,
  limit = 10,
): StatusMetricHistoryRow[] {
  return points
    .slice(-limit)
    .reverse()
    .map((point) => {
      const value = point[metric.key] || 0;
      return {
        timestamp: point.timestamp,
        value,
        formattedValue: metric.format(value),
      };
    });
}

export function buildStatusHistoryCacheKey(query: StatusHistoryQuery) {
  if (query.mode === 'custom') {
    return `custom:${query.start}:${query.end}`;
  }
  return `preset:${query.mode}`;
}

export function buildStatusHistoryParams(query: StatusHistoryQuery) {
  if (query.mode === 'custom') {
    return {
      start: query.start,
      end: query.end,
    };
  }
  return {
    range: query.mode,
  };
}

export function toggleExpandedMetricKey(
  current: StatusMetricKey | null,
  next: StatusMetricKey,
) {
  return current === next ? null : next;
}

export function createMetricHistoryStore(
  fetcher: (query: StatusHistoryQuery) => Promise<StatusHistoryRangePayload>,
) {
  const cache = new Map<string, Promise<StatusHistoryRangePayload>>();

  return {
    async getQueryData(query: StatusHistoryQuery) {
      const cacheKey = buildStatusHistoryCacheKey(query);
      if (!cache.has(cacheKey)) {
        const task = Promise.resolve(fetcher(query)).catch((err) => {
          cache.delete(cacheKey);
          throw err;
        });
        cache.set(cacheKey, task);
      }
      return await cache.get(cacheKey)!;
    },
    clearQuery(query: StatusHistoryQuery) {
      cache.delete(buildStatusHistoryCacheKey(query));
    },
    clearAll() {
      cache.clear();
    },
  };
}
