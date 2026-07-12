export type EffectiveDisplayPalette = 'day' | 'night'
export type StoredDisplayPalette = EffectiveDisplayPalette | 'auto'

interface MatchMediaLike {
  matches: boolean
}

interface WindowLike {
  matchMedia?: (query: string) => MatchMediaLike
}

const SYSTEM_DARK_QUERY = '(prefers-color-scheme: dark)'

export const resolveSystemDisplayPalette = (targetWindow?: WindowLike): EffectiveDisplayPalette => {
  try {
    if (targetWindow?.matchMedia?.(SYSTEM_DARK_QUERY)?.matches) {
      return 'night'
    }
    if (targetWindow?.matchMedia) {
      return 'day'
    }
  } catch {
    // Ignore matchMedia errors and fall back to a stable default.
  }
  return 'night'
}

export const resolveEffectiveDisplayPalette = (
  palette: StoredDisplayPalette,
  getSystemPalette?: () => EffectiveDisplayPalette,
): EffectiveDisplayPalette => {
  if (palette === 'auto') {
    return (getSystemPalette || (() => resolveSystemDisplayPalette(typeof window !== 'undefined' ? window : undefined)))()
  }
  return palette === 'day' ? 'day' : 'night'
}
