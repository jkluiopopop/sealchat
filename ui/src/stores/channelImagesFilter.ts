export type ChannelImagesIcModeFilter = 'all' | 'ic' | 'ooc'
export type ChannelImagesSortOrder = 'desc' | 'asc'
export type ChannelImagesThumbnailMode = 'small' | 'large'

export const CHANNEL_IMAGES_IC_MODE_FILTER_STORAGE_KEY = 'sealchat.channelImages.icModeFilter'
export const CHANNEL_IMAGES_SORT_ORDER_STORAGE_KEY = 'sealchat.channelImages.sortOrder'
export const CHANNEL_IMAGES_THUMBNAIL_MODE_STORAGE_KEY = 'sealchat.channelImages.thumbnailMode'

export const normalizeChannelImagesIcModeFilter = (value: unknown): ChannelImagesIcModeFilter => {
  return value === 'ic' || value === 'ooc' || value === 'all' ? value : 'all'
}

export const nextChannelImagesIcModeFilter = (value: unknown): ChannelImagesIcModeFilter => {
  switch (normalizeChannelImagesIcModeFilter(value)) {
    case 'ic':
      return 'ooc'
    case 'ooc':
      return 'all'
    default:
      return 'ic'
  }
}

export const normalizeChannelImagesSortOrder = (value: unknown): ChannelImagesSortOrder => {
  return value === 'asc' ? 'asc' : 'desc'
}

export const normalizeChannelImagesThumbnailMode = (value: unknown): ChannelImagesThumbnailMode => {
  return value === 'small' ? 'small' : 'large'
}

export const nextChannelImagesSortOrder = (value: unknown): ChannelImagesSortOrder => {
  return normalizeChannelImagesSortOrder(value) === 'desc' ? 'asc' : 'desc'
}

const resolveDefaultStorage = (): Pick<Storage, 'getItem' | 'setItem'> | null => {
  if (typeof window === 'undefined') {
    return null
  }
  return window.localStorage
}

export const readChannelImagesIcModeFilter = (
  storage: Pick<Storage, 'getItem'> | null = resolveDefaultStorage(),
): ChannelImagesIcModeFilter => {
  try {
    return normalizeChannelImagesIcModeFilter(storage?.getItem(CHANNEL_IMAGES_IC_MODE_FILTER_STORAGE_KEY))
  } catch {
    return 'all'
  }
}

export const writeChannelImagesIcModeFilter = (
  storageOrValue: Pick<Storage, 'setItem'> | ChannelImagesIcModeFilter | null,
  maybeValue?: ChannelImagesIcModeFilter,
): void => {
  const storage = typeof storageOrValue === 'string' ? resolveDefaultStorage() : storageOrValue
  const value = typeof storageOrValue === 'string' ? storageOrValue : maybeValue
  try {
    storage?.setItem(CHANNEL_IMAGES_IC_MODE_FILTER_STORAGE_KEY, normalizeChannelImagesIcModeFilter(value))
  } catch {
    // Ignore storage failures in private mode or quota-limited environments.
  }
}

export const readChannelImagesSortOrder = (
  storage: Pick<Storage, 'getItem'> | null = resolveDefaultStorage(),
): ChannelImagesSortOrder => {
  try {
    return normalizeChannelImagesSortOrder(storage?.getItem(CHANNEL_IMAGES_SORT_ORDER_STORAGE_KEY))
  } catch {
    return 'desc'
  }
}

export const writeChannelImagesSortOrder = (
  storageOrValue: Pick<Storage, 'setItem'> | ChannelImagesSortOrder | null,
  maybeValue?: ChannelImagesSortOrder,
): void => {
  const storage = typeof storageOrValue === 'string' ? resolveDefaultStorage() : storageOrValue
  const value = typeof storageOrValue === 'string' ? storageOrValue : maybeValue
  try {
    storage?.setItem(CHANNEL_IMAGES_SORT_ORDER_STORAGE_KEY, normalizeChannelImagesSortOrder(value))
  } catch {
    // Ignore storage failures in private mode or quota-limited environments.
  }
}

export const readChannelImagesThumbnailMode = (
  storage: Pick<Storage, 'getItem'> | null = resolveDefaultStorage(),
): ChannelImagesThumbnailMode => {
  try {
    return normalizeChannelImagesThumbnailMode(storage?.getItem(CHANNEL_IMAGES_THUMBNAIL_MODE_STORAGE_KEY))
  } catch {
    return 'large'
  }
}

export const writeChannelImagesThumbnailMode = (
  storageOrValue: Pick<Storage, 'setItem'> | ChannelImagesThumbnailMode | null,
  maybeValue?: ChannelImagesThumbnailMode,
): void => {
  const storage = typeof storageOrValue === 'string' ? resolveDefaultStorage() : storageOrValue
  const value = typeof storageOrValue === 'string' ? storageOrValue : maybeValue
  try {
    storage?.setItem(CHANNEL_IMAGES_THUMBNAIL_MODE_STORAGE_KEY, normalizeChannelImagesThumbnailMode(value))
  } catch {
    // Ignore storage failures in private mode or quota-limited environments.
  }
}
