export const PAGE_DESCRIPTION_MAX_LENGTH = 60

export const normalizePageDescription = (value?: string | null) => {
  const trimmed = value?.trim() || ''
  return Array.from(trimmed).slice(0, PAGE_DESCRIPTION_MAX_LENGTH).join('')
}
