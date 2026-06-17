import { defineStore } from 'pinia';
import { api } from './_config';
import type { BattleReport, BattleReportPayload } from '@/types';

interface BattleReportListResponse {
  items?: BattleReport[];
}

interface BattleReportItemResponse {
  item?: BattleReport;
}

interface BattleReportState {
  itemsByChannel: Record<string, BattleReport[]>;
  detailById: Record<string, BattleReport>;
  loading: boolean;
  saving: boolean;
}

const normalizeItems = (items?: BattleReport[]): BattleReport[] => Array.isArray(items) ? items : [];

export const useBattleReportStore = defineStore('battleReport', {
  state: (): BattleReportState => ({
    itemsByChannel: Object.create(null),
    detailById: Object.create(null),
    loading: false,
    saving: false,
  }),
  actions: {
    upsertItem(item?: BattleReport) {
      if (!item?.id) {
        return;
      }
      const existingDetail = this.detailById[item.id];
      this.detailById[item.id] = {
        ...existingDetail,
        ...item,
        content: item.content !== undefined ? item.content : existingDetail?.content,
      };
      const channelId = item.channelId;
      if (!channelId) {
        return;
      }
      const current = this.itemsByChannel[channelId] || [];
      const index = current.findIndex((candidate) => candidate.id === item.id);
      const mergeListItem = (candidate: BattleReport) => ({
        ...candidate,
        ...item,
        content: item.content !== undefined ? item.content : candidate.content,
      });
      const next = index >= 0
        ? current.map((candidate) => candidate.id === item.id ? mergeListItem(candidate) : candidate)
        : [item, ...current];
      this.itemsByChannel[channelId] = next;
    },
    async list(channelId: string) {
      this.loading = true;
      try {
        const resp = await api.get<BattleReportListResponse>(`api/v1/channels/${channelId}/battle-reports`);
        const items = normalizeItems(resp.data?.items).map((item) => {
          const existingDetail = this.detailById[item.id];
          return {
            ...item,
            content: item.content !== undefined ? item.content : existingDetail?.content,
          };
        });
        this.itemsByChannel[channelId] = items;
        items.forEach((item) => this.upsertItem(item));
        return this.itemsByChannel[channelId] || items;
      } finally {
        this.loading = false;
      }
    },
    async get(reportId: string) {
      this.loading = true;
      try {
        const resp = await api.get<BattleReportItemResponse>(`api/v1/battle-reports/${reportId}`);
        const item = resp.data?.item;
        this.upsertItem(item);
        return item;
      } finally {
        this.loading = false;
      }
    },
    async create(channelId: string, payload: BattleReportPayload) {
      this.saving = true;
      try {
        const resp = await api.post<BattleReportItemResponse>(`api/v1/channels/${channelId}/battle-reports`, payload);
        const item = resp.data?.item;
        this.upsertItem(item);
        return item;
      } finally {
        this.saving = false;
      }
    },
    async update(reportId: string, payload: BattleReportPayload) {
      this.saving = true;
      try {
        const resp = await api.patch<BattleReportItemResponse>(`api/v1/battle-reports/${reportId}`, payload);
        const item = resp.data?.item;
        this.upsertItem(item);
        return item;
      } finally {
        this.saving = false;
      }
    },
    async delete(reportId: string) {
      this.saving = true;
      try {
        await api.delete(`api/v1/battle-reports/${reportId}`);
        const existing = this.detailById[reportId];
        delete this.detailById[reportId];
        if (existing?.channelId) {
          this.itemsByChannel[existing.channelId] = (this.itemsByChannel[existing.channelId] || [])
            .filter((item) => item.id !== reportId);
        }
      } finally {
        this.saving = false;
      }
    },
    async reorder(channelId: string, ids: string[]) {
      await api.post(`api/v1/channels/${channelId}/battle-reports/reorder`, { ids });
    },
    async summarize(channelId: string, payload: BattleReportPayload) {
      this.saving = true;
      try {
        const resp = await api.post<BattleReportItemResponse>(`api/v1/channels/${channelId}/battle-reports/summarize`, payload);
        const item = resp.data?.item;
        this.upsertItem(item);
        return item;
      } finally {
        this.saving = false;
      }
    },
    setChannelItems(channelId: string, items: BattleReport[]) {
      this.itemsByChannel[channelId] = items.map((item) => {
        const existingDetail = this.detailById[item.id];
        return {
          ...item,
          content: item.content !== undefined ? item.content : existingDetail?.content,
        };
      });
      items.forEach((item) => {
        if (item?.id) {
          this.upsertItem(item);
        }
      });
    },
  },
});
