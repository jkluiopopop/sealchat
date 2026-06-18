export interface KeywordTooltipPresentationInput {
  isEmbeddedRuntime: boolean
  finePointer: boolean
  viewportWidth: number
  level: number
}

export interface KeywordTooltipPresentation {
  mode: 'floating' | 'embedded-wide'
}

export interface EmbeddedWideTooltipPlacementInput {
  targetTop: number
  targetBottom: number
  tooltipHeight: number
  viewportHeight: number
  topMargin: number
  bottomMargin: number
  gap: number
}

const EMBED_WIDE_BREAKPOINT = 768

export function resolveKeywordTooltipPresentation(
  input: KeywordTooltipPresentationInput,
): KeywordTooltipPresentation {
  if (
    input.isEmbeddedRuntime
    && input.finePointer
    && input.level === 0
    && input.viewportWidth > 0
    && input.viewportWidth <= EMBED_WIDE_BREAKPOINT
  ) {
    return { mode: 'embedded-wide' }
  }

  return { mode: 'floating' }
}

export function resolveEmbeddedWideTooltipTop(
  input: EmbeddedWideTooltipPlacementInput,
): number {
  const minTop = input.topMargin
  const maxTop = Math.max(minTop, input.viewportHeight - input.bottomMargin - input.tooltipHeight)

  const aboveTop = input.targetTop - input.gap - input.tooltipHeight
  const belowTop = input.targetBottom + input.gap
  const fitsAbove = aboveTop >= minTop
  const fitsBelow = belowTop <= maxTop

  if (fitsAbove) {
    return aboveTop
  }

  if (fitsBelow) {
    return belowTop
  }

  const aboveVisible = Math.max(0, input.targetTop - input.gap - minTop)
  const belowVisible = Math.max(0, input.viewportHeight - input.bottomMargin - (input.targetBottom + input.gap))

  if (belowVisible > aboveVisible) {
    return Math.min(maxTop, Math.max(minTop, belowTop))
  }

  return Math.min(maxTop, Math.max(minTop, aboveTop))
}
