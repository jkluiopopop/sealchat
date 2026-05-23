type CharacterApiRuntimeStateCarrier = {
  id?: string;
  characterApiEnabled?: boolean;
  characterApiReason?: string;
  children?: CharacterApiRuntimeStateCarrier[];
};

type CharacterApiRuntimeSnapshot = {
  hasEnabled: boolean;
  enabled?: boolean;
  hasReason: boolean;
  reason?: string;
};

const collectCharacterApiRuntimeSnapshot = (
  nodes: CharacterApiRuntimeStateCarrier[] = [],
  bucket = new Map<string, CharacterApiRuntimeSnapshot>(),
) => {
  for (const node of nodes) {
    if (!node || !node.id) {
      continue;
    }
    bucket.set(node.id, {
      hasEnabled: typeof node.characterApiEnabled === 'boolean',
      enabled: node.characterApiEnabled,
      hasReason: typeof node.characterApiReason === 'string',
      reason: node.characterApiReason,
    });
    if (Array.isArray(node.children) && node.children.length > 0) {
      collectCharacterApiRuntimeSnapshot(node.children, bucket);
    }
  }
  return bucket;
};

export const mergeCharacterApiRuntimeStateIntoChannels = <T extends CharacterApiRuntimeStateCarrier>(
  channels: T[] = [],
  previousTree: CharacterApiRuntimeStateCarrier[] = [],
): T[] => {
  const snapshot = collectCharacterApiRuntimeSnapshot(previousTree);
  const mergeNode = (node: T): T => {
    const merged = { ...node } as T;
    const state = node.id ? snapshot.get(node.id) : undefined;
    if (state) {
      if (typeof merged.characterApiEnabled !== 'boolean' && state.hasEnabled) {
        merged.characterApiEnabled = state.enabled;
      }
      if (typeof merged.characterApiReason !== 'string' && state.hasReason) {
        merged.characterApiReason = state.reason;
      }
    }
    if (Array.isArray(node.children)) {
      merged.children = node.children.map((child) => mergeNode(child as T));
    }
    return merged;
  };
  return channels.map((channel) => mergeNode(channel));
};
