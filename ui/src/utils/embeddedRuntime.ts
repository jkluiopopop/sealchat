export type EmbeddedRuntimeKind = 'top-level' | 'embed-route' | 'split-shell' | 'framed-app';

export interface EmbeddedRuntimeInfo {
  runtimeKind: EmbeddedRuntimeKind;
  isEmbeddedRuntime: boolean;
  thirdPartyFrameLikely: boolean;
  isSplitShell: boolean;
  isEmbedRoute: boolean;
  hasEmbedSignal: boolean;
}

function readHashQuery(): URLSearchParams {
  if (typeof window === 'undefined') {
    return new URLSearchParams();
  }
  const hash = window.location.hash || '';
  const queryIndex = hash.indexOf('?');
  if (queryIndex < 0) {
    return new URLSearchParams();
  }
  return new URLSearchParams(hash.slice(queryIndex + 1));
}

export function detectEmbeddedRuntime(): EmbeddedRuntimeInfo {
  if (typeof window === 'undefined') {
    return {
      runtimeKind: 'top-level',
      isEmbeddedRuntime: false,
      thirdPartyFrameLikely: false,
      isSplitShell: false,
      isEmbedRoute: false,
      hasEmbedSignal: false,
    };
  }

  let isFramed = false;
  let sameOriginTop = true;
  try {
    isFramed = window.self !== window.top;
    if (isFramed) {
      sameOriginTop = window.top?.location?.origin === window.location.origin;
    }
  } catch {
    isFramed = true;
    sameOriginTop = false;
  }

  const hash = window.location.hash || '';
  const hashPath = hash.startsWith('#') ? hash.slice(1).split('?')[0] : hash.split('?')[0];
  const search = new URLSearchParams(window.location.search || '');
  const hashQuery = readHashQuery();
  const isEmbedRoute = hashPath.startsWith('/embed');
  const isSplitShell = hashPath.startsWith('/split');
  const hasEmbedSignal = ['audioOwner', 'paneId', 'notifyOwner'].some((key) => {
    const value = hashQuery.get(key) || search.get(key) || '';
    return value.trim() !== '';
  });
  const isEmbeddedRuntime = isFramed || isEmbedRoute || hasEmbedSignal;

  let runtimeKind: EmbeddedRuntimeKind = 'top-level';
  if (isSplitShell) {
    runtimeKind = 'split-shell';
  } else if (isEmbedRoute) {
    runtimeKind = 'embed-route';
  } else if (isEmbeddedRuntime) {
    runtimeKind = 'framed-app';
  }

  return {
    runtimeKind,
    isEmbeddedRuntime,
    thirdPartyFrameLikely: isFramed && !sameOriginTop,
    isSplitShell,
    isEmbedRoute,
    hasEmbedSignal,
  };
}
