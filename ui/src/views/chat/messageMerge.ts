type WhisperComparable = {
  whisperToIds?: unknown;
  whisper_to_ids?: unknown;
  whisperTargets?: unknown;
  whisper_targets?: unknown;
  whisperTo?: unknown;
  whisper_to?: unknown;
  whisper_target?: unknown;
  whisperMeta?: {
    targetUserId?: unknown;
    targetUserIds?: unknown;
  } | null;
  isWhisper?: unknown;
  is_whisper?: unknown;
};

export type MergeComparableMessage = WhisperComparable & {
  roleKey?: unknown;
  senderRoleId?: unknown;
  sender_role_id?: unknown;
  sender_identity_id?: unknown;
  identity?: { id?: unknown } | null;
  member?: { id?: unknown; member_id?: unknown } | null;
  sender_member_id?: unknown;
  user?: { id?: unknown } | null;
  user_id?: unknown;
  sceneKey?: unknown;
  icMode?: unknown;
  ic_mode?: unknown;
  avatarMergeKey?: unknown;
};

const normalizeText = (value: unknown) => String(value || '').trim();

const normalizeWhisperFlag = (message?: WhisperComparable | null) => Boolean(
  message?.isWhisper ?? message?.is_whisper,
);

export const resolveWhisperTargetIdsForMerge = (message?: WhisperComparable | null): string[] => {
  if (!message) {
    return [];
  }
  const resolved = new Set<string>();
  const collect = (raw: unknown) => {
    if (!Array.isArray(raw)) {
      return;
    }
    raw.forEach((item) => {
      const candidate = typeof item === 'string'
        ? item
        : ((item as any)?.id || (item as any)?.userId || (item as any)?.user_id || '');
      const id = normalizeText(candidate);
      if (id) {
        resolved.add(id);
      }
    });
  };

  collect(message.whisperToIds);
  collect(message.whisper_to_ids);
  collect(message.whisperTargets);
  collect(message.whisper_targets);
  collect(message.whisperMeta?.targetUserIds);

  const directCandidates = [
    typeof message.whisperTo === 'string' ? message.whisperTo : (message.whisperTo as any)?.id,
    typeof message.whisper_to === 'string' ? message.whisper_to : (message.whisper_to as any)?.id,
    (message.whisper_target as any)?.id,
    message.whisperMeta?.targetUserId,
  ];
  directCandidates.forEach((candidate) => {
    const id = normalizeText(candidate);
    if (id) {
      resolved.add(id);
    }
  });

  return Array.from(resolved).sort();
};

export const resolveWhisperContextKey = (message?: WhisperComparable | null): string => {
  if (!normalizeWhisperFlag(message)) {
    return 'public';
  }
  const targetIds = resolveWhisperTargetIdsForMerge(message);
  if (targetIds.length > 0) {
    return `whisper:${targetIds.join(',')}`;
  }

  const fallback = [
    normalizeText(message?.whisperMeta?.targetUserId),
    normalizeText((message?.whisperTo as any)?.name),
    normalizeText((message?.whisperTo as any)?.nick),
    normalizeText((message?.whisper_to as any)?.name),
    normalizeText((message?.whisper_to as any)?.nick),
    normalizeText((message?.whisper_target as any)?.name),
    normalizeText((message?.whisper_target as any)?.nick),
  ].filter(Boolean);

  return fallback.length > 0 ? `whisper-fallback:${fallback.join('|')}` : 'whisper:unknown';
};

const resolveRoleKey = (message?: MergeComparableMessage | null) => normalizeText(
  message?.roleKey
  ?? message?.senderRoleId
  ?? message?.sender_role_id
  ?? message?.sender_identity_id
  ?? message?.identity?.id
  ?? message?.member?.id
  ?? message?.member?.member_id
  ?? message?.sender_member_id
  ?? message?.user?.id
  ?? message?.user_id,
);

const resolveSceneKey = (message?: MergeComparableMessage | null) => normalizeText(
  message?.sceneKey ?? message?.icMode ?? message?.ic_mode ?? 'ic',
).toLowerCase();

const resolveAvatarKey = (message?: MergeComparableMessage | null) => normalizeText(message?.avatarMergeKey);

export const shouldMergeNeighborMessages = (
  prev?: MergeComparableMessage | null,
  current?: MergeComparableMessage | null,
) => {
  if (!prev || !current) {
    return false;
  }
  if (normalizeWhisperFlag(prev) !== normalizeWhisperFlag(current)) {
    return false;
  }
  const prevRoleKey = resolveRoleKey(prev);
  const currentRoleKey = resolveRoleKey(current);
  if (!prevRoleKey || prevRoleKey !== currentRoleKey) {
    return false;
  }
  if (resolveSceneKey(prev) !== resolveSceneKey(current)) {
    return false;
  }
  if (resolveAvatarKey(prev) !== resolveAvatarKey(current)) {
    return false;
  }
  return resolveWhisperContextKey(prev) === resolveWhisperContextKey(current);
};

export const shouldRenderWhisperLabel = (
  message?: WhisperComparable | null,
  isMerged = false,
) => {
  if (!normalizeWhisperFlag(message)) {
    return false;
  }
  return !isMerged;
};
