export interface PageTitleState {
  unreadCount: number;
  currentChannelName: string;
  displayedTitle: string;
}

const normalizeTitle = (value?: string | null) => (typeof value === 'string' ? value.trim() : '');

const resolveChannelOrDefaultTitle = (channelName: string, defaultTitle: string) => {
  const normalizedChannelName = normalizeTitle(channelName);
  const normalizedDefaultTitle = normalizeTitle(defaultTitle);
  return normalizedChannelName || normalizedDefaultTitle;
};

export const createPageTitleState = (defaultTitle: string, channelName = ''): PageTitleState => ({
  unreadCount: 0,
  currentChannelName: normalizeTitle(channelName),
  displayedTitle: resolveChannelOrDefaultTitle(channelName, defaultTitle),
});

export const setChannelTitleState = (
  state: PageTitleState,
  channelName: string,
  defaultTitle: string,
): PageTitleState => {
  const nextChannelName = normalizeTitle(channelName);
  return {
    unreadCount: state.unreadCount,
    currentChannelName: nextChannelName,
    displayedTitle: state.unreadCount > 0
      ? state.displayedTitle
      : resolveChannelOrDefaultTitle(nextChannelName, defaultTitle),
  };
};

export const updateUnreadTitleState = (
  state: PageTitleState,
  count: number,
  channelName: string,
  defaultTitle: string,
): PageTitleState => {
  const unreadCount = Number.isFinite(count) ? Math.max(0, Math.trunc(count)) : 0;
  const nextChannelName = normalizeTitle(channelName) || state.currentChannelName;
  if (unreadCount > 0 && nextChannelName) {
    return {
      unreadCount,
      currentChannelName: nextChannelName,
      displayedTitle: `有${unreadCount}条新消息 | ${nextChannelName}`,
    };
  }
  return {
    unreadCount: 0,
    currentChannelName: nextChannelName,
    displayedTitle: resolveChannelOrDefaultTitle(nextChannelName, defaultTitle),
  };
};

export const clearUnreadTitleState = (state: PageTitleState, defaultTitle: string): PageTitleState => ({
  unreadCount: 0,
  currentChannelName: state.currentChannelName,
  displayedTitle: resolveChannelOrDefaultTitle(state.currentChannelName, defaultTitle),
});

export const replaceChannelTitleState = (
  state: PageTitleState,
  channelName: string,
  defaultTitle: string,
): PageTitleState => {
  const nextState = clearUnreadTitleState(state, defaultTitle);
  return setChannelTitleState(nextState, channelName, defaultTitle);
};
