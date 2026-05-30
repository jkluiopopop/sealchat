export interface SplitChannelDisplayLike {
  name?: string | null;
  permType?: string | null;
}

export const isSplitChannelNonPublic = (channel?: SplitChannelDisplayLike | null) => {
  const permType = typeof channel?.permType === 'string' ? channel.permType.trim().toLowerCase() : '';
  return permType === 'non-public';
};

export const formatSplitChannelDisplayName = (channel?: SplitChannelDisplayLike | null) => {
  const base = typeof channel?.name === 'string' && channel.name.trim()
    ? channel.name.trim()
    : '未命名频道';
  if (isSplitChannelNonPublic(channel)) {
    return `${base}[*]`;
  }
  return base;
};
