export interface BattleReportEmbedLinkParams {
  worldId: string;
  channelId: string;
  reportId: string;
}

export interface ParsedSingleBattleReportEmbedLink extends BattleReportEmbedLinkParams {
  rawLink: string;
}

const BATTLE_REPORT_LINK_EXACT_REGEX = /^(?:https?:\/\/[^\s<>"']*)?\/?#\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_-]+)\?([^\s#]+)$/;

const normalizeInput = (value: string) => value.replace(/&amp;/gi, '&').trim();

const resolveLinkBase = (base?: string): string => {
  const trimmed = (base || '').trim();
  if (trimmed) return trimmed.replace(/\/+$/, '');
  if (typeof window === 'undefined') return '';
  return window.location.origin;
};

export function generateBattleReportEmbedLink(
  params: BattleReportEmbedLinkParams,
  options?: { base?: string },
): string {
  const base = resolveLinkBase(options?.base);
  const search = new URLSearchParams({ battleReport: params.reportId });
  return `${base}/#/${params.worldId}/${params.channelId}?${search.toString()}`;
}

export function parseBattleReportEmbedLink(url: string): BattleReportEmbedLinkParams | null {
  if (!url || typeof url !== 'string') return null;
  const normalized = normalizeInput(url);
  const match = normalized.match(BATTLE_REPORT_LINK_EXACT_REGEX);
  if (!match) return null;
  const [, worldId, channelId, queryString] = match;
  const search = new URLSearchParams(queryString);
  const reportId = (search.get('battleReport') || '').trim();
  if (!worldId || !channelId || !reportId) return null;
  return { worldId, channelId, reportId };
}

export function parseSingleBattleReportEmbedLinkText(text: string): ParsedSingleBattleReportEmbedLink | null {
  if (!text || typeof text !== 'string') return null;
  const normalized = normalizeInput(text).replace(/\u00a0/g, ' ').trim();
  if (!normalized || /\s/.test(normalized)) return null;
  const parsed = parseBattleReportEmbedLink(normalized);
  if (!parsed) return null;
  return { ...parsed, rawLink: normalized };
}
