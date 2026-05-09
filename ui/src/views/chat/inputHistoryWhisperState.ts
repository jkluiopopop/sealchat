export interface WhisperSnapshotTarget {
  id: string;
  nick?: string;
  name?: string;
  avatar?: string;
  color?: string;
  nick_color?: string;
  nickColor?: string;
  whisperIcDisplayName?: string;
  whisperOocDisplayName?: string;
  whisperUserDisplayName?: string;
  whisperIcColor?: string;
  whisperOocColor?: string;
}

export interface WhisperSnapshot {
  enabled: boolean;
  targets: WhisperSnapshotTarget[];
}

export interface WhisperSnapshotController {
  clearWhisperTargets: () => void;
  toggleWhisperTarget: (target: WhisperSnapshotTarget) => void;
}

const normalizeText = (value: unknown) => String(value || '').trim();

const normalizeOptionalText = (value: unknown): string | undefined => {
  const normalized = normalizeText(value);
  return normalized || undefined;
};

const normalizeWhisperSnapshotTarget = (raw: any): WhisperSnapshotTarget | null => {
  const id = normalizeText(raw?.id);
  if (!id) {
    return null;
  }
  const target: WhisperSnapshotTarget = { id };
  const apply = (key: keyof WhisperSnapshotTarget, value: unknown) => {
    const normalized = normalizeOptionalText(value);
    if (normalized) {
      target[key] = normalized as never;
    }
  };
  apply('nick', raw?.nick);
  apply('name', raw?.name);
  apply('avatar', raw?.avatar);
  apply('color', raw?.color);
  apply('nick_color', raw?.nick_color);
  apply('nickColor', raw?.nickColor);
  apply('whisperIcDisplayName', raw?.whisperIcDisplayName);
  apply('whisperOocDisplayName', raw?.whisperOocDisplayName);
  apply('whisperUserDisplayName', raw?.whisperUserDisplayName);
  apply('whisperIcColor', raw?.whisperIcColor);
  apply('whisperOocColor', raw?.whisperOocColor);
  return target;
};

const normalizeWhisperSnapshotTargets = (raw: unknown): WhisperSnapshotTarget[] => {
  if (!Array.isArray(raw)) {
    return [];
  }
  return raw
    .map((target) => normalizeWhisperSnapshotTarget(target))
    .filter((target): target is WhisperSnapshotTarget => !!target);
};

export const captureWhisperSnapshot = (targets: unknown): WhisperSnapshot => {
  const normalizedTargets = normalizeWhisperSnapshotTargets(targets);
  return {
    enabled: normalizedTargets.length > 0,
    targets: normalizedTargets,
  };
};

export const normalizeWhisperSnapshot = (raw: unknown): WhisperSnapshot => {
  if (!raw || typeof raw !== 'object') {
    return {
      enabled: false,
      targets: [],
    };
  }
  const targets = normalizeWhisperSnapshotTargets((raw as any).targets);
  const enabled = (raw as any).enabled === true && targets.length > 0;
  return {
    enabled,
    targets: enabled ? targets : [],
  };
};

const buildWhisperContextKey = (snapshot?: WhisperSnapshot | null): string => {
  const normalized = normalizeWhisperSnapshot(snapshot);
  if (!normalized.enabled || normalized.targets.length === 0) {
    return 'public';
  }
  const ids = normalized.targets.map((target) => target.id).filter(Boolean).sort();
  if (ids.length > 0) {
    return `whisper:${ids.join(',')}`;
  }
  return 'whisper:unknown';
};

export const buildInputHistorySignature = (
  mode: 'plain' | 'rich',
  content: string,
  snapshot?: WhisperSnapshot | null,
) => `${mode}:${buildWhisperContextKey(snapshot)}:${content}`;

export const restoreWhisperSnapshot = (
  controller: WhisperSnapshotController,
  snapshot?: WhisperSnapshot | null,
) => {
  controller.clearWhisperTargets();
  const normalized = normalizeWhisperSnapshot(snapshot);
  if (!normalized.enabled) {
    return;
  }
  normalized.targets.forEach((target) => {
    controller.toggleWhisperTarget({ ...target });
  });
};
