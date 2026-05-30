import { computed, h, nextTick, onBeforeUnmount, reactive, watch, type ComputedRef, type Ref, type VNodeChild } from 'vue'
import type { SelectOption } from 'naive-ui'
import { ensurePlatformFontLoaded } from './platformFontRegistry'
import type { PlatformFontAsset } from './platformFontTypes'

type PlatformFontSelectOption = SelectOption & {
  fontAsset: PlatformFontAsset
  rawLabel: string
  previewFamily?: string
}

type RenderOptionInfo = {
  node: VNodeChild
  option: SelectOption
}

type UsePlatformFontSelectPreviewOptions = {
  fonts: Ref<PlatformFontAsset[]>
  selectedId: Ref<string | null>
  menuClass: string
  immediateSelectedPreview?: boolean
}

const PREVIEW_ROOT_MARGIN = '120px'

const previewFamilyCache = reactive<Record<string, string>>({})
const scheduledLoads = new Set<string>()

const scheduleIdleTask = (task: () => void) => {
  if (typeof window !== 'undefined' && typeof window.requestIdleCallback === 'function') {
    window.requestIdleCallback(() => task(), { timeout: 240 })
    return
  }
  setTimeout(task, 16)
}

const resolveDisplayLabel = (item: { displayName?: string | null; family?: string | null }) => {
  const displayName = String(item.displayName || '').trim()
  const family = String(item.family || '').trim()
  return displayName && displayName !== family ? `${displayName} · ${family}` : (displayName || family)
}

const resolveInlineFontFamily = (item: { id?: string; family?: string | null }) => {
  const cachedFamily = item.id ? previewFamilyCache[item.id] : ''
  const family = String(cachedFamily || item.family || '').trim()
  return family ? `"${family}"` : undefined
}

const ensurePreviewFamilyLoaded = async (item: PlatformFontAsset) => {
  if (!item?.id) return ''
  const family = await ensurePlatformFontLoaded(item.id, item.family)
  previewFamilyCache[item.id] = family
  return family
}

const queuePreviewLoad = (item: PlatformFontAsset | null | undefined) => {
  const id = String(item?.id || '').trim()
  if (!id || scheduledLoads.has(id)) return
  scheduledLoads.add(id)
  scheduleIdleTask(() => {
    void ensurePreviewFamilyLoaded(item as PlatformFontAsset)
      .catch(() => {
        if (item?.family) {
          previewFamilyCache[id] = item.family
        }
      })
      .finally(() => {
        scheduledLoads.delete(id)
      })
  })
}

const buildLabelNode = (option: PlatformFontSelectOption) => {
  return h(
    'span',
    {
      class: 'platform-font-select-preview__value',
      style: option.previewFamily ? { fontFamily: option.previewFamily, fontWeight: option.fontAsset.weight, fontStyle: option.fontAsset.style } : undefined,
      title: option.rawLabel,
      onVnodeMounted: () => queuePreviewLoad(option.fontAsset),
      onMouseenter: () => queuePreviewLoad(option.fontAsset),
    },
    option.rawLabel,
  )
}

const buildOptionNodeWithPreview = (node: VNodeChild, option: PlatformFontSelectOption) => {
  return h(
    'div',
    {
      class: 'platform-font-select-preview__option',
      title: option.rawLabel,
      'data-platform-font-id': option.fontAsset.id,
      'data-platform-font-family': option.fontAsset.family,
      'data-platform-font-label': option.rawLabel,
      'data-platform-font-weight': option.fontAsset.weight,
      'data-platform-font-style': option.fontAsset.style,
      onMouseenter: () => queuePreviewLoad(option.fontAsset),
    },
    [
      h('div', { class: 'platform-font-select-preview__meta' }, [node]),
    ],
  )
}

export const createPlatformFontSelectPreviewController = ({
  fonts,
  selectedId,
  menuClass,
  immediateSelectedPreview = true,
}: UsePlatformFontSelectPreviewOptions) => {
  const observedMenus = new WeakSet<HTMLElement>()
  let menuObserver: IntersectionObserver | null = null
  let mutationObserver: MutationObserver | null = null

  const ensureMenuObserver = () => {
    if (menuObserver || typeof window === 'undefined' || typeof document === 'undefined' || typeof IntersectionObserver === 'undefined') {
      return
    }
    menuObserver = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) return
        const element = entry.target as HTMLElement
        const fontId = element.dataset.platformFontId?.trim()
        const family = element.dataset.platformFontFamily?.trim()
        if (!fontId || !family) return
        queuePreviewLoad({
          id: fontId,
          family,
          displayName: element.dataset.platformFontLabel || family,
          weight: element.dataset.platformFontWeight || '400',
          style: element.dataset.platformFontStyle || 'normal',
          status: 'ready',
          deliveryMode: 'single',
        } as PlatformFontAsset)
      })
    }, {
      rootMargin: PREVIEW_ROOT_MARGIN,
      threshold: 0,
    })

    const observeMenuNodes = () => {
      const menus = document.querySelectorAll<HTMLElement>(`.${menuClass}`)
      menus.forEach((menu) => {
        if (observedMenus.has(menu)) return
        observedMenus.add(menu)
        menu.querySelectorAll<HTMLElement>('[data-platform-font-id]').forEach((node) => {
          menuObserver?.observe(node)
        })
      })
    }

    observeMenuNodes()
    mutationObserver = new MutationObserver(() => observeMenuNodes())
    mutationObserver.observe(document.body, { childList: true, subtree: true })
  }

  const platformFontOptions: ComputedRef<PlatformFontSelectOption[]> = computed(() => {
    return fonts.value.map((item) => ({
      label: resolveDisplayLabel(item),
      rawLabel: resolveDisplayLabel(item),
      value: item.id,
      previewFamily: resolveInlineFontFamily(item),
      fontAsset: item,
    }))
  })

  const primeSelectedPreview = (fontId: string | null) => {
    const target = fonts.value.find((item) => item.id === fontId) || null
    if (!target) return
    queuePreviewLoad(target)
  }

  const handleDropdownVisible = (show: boolean) => {
    if (!show) return
    ensureMenuObserver()
    void nextTick(() => {
      const target = fonts.value.find((item) => item.id === selectedId.value) || null
      if (target) {
        queuePreviewLoad(target)
      }
    })
  }

  const renderPlatformFontLabel = (option: SelectOption) => {
    return buildLabelNode(option as PlatformFontSelectOption)
  }

  const renderPlatformFontOption = ({ node, option }: RenderOptionInfo) => {
    return buildOptionNodeWithPreview(node, option as PlatformFontSelectOption)
  }

  if (immediateSelectedPreview) {
    watch(selectedId, (fontId) => {
      primeSelectedPreview(fontId)
    }, { immediate: true })
  }

  onBeforeUnmount(() => {
    if (menuObserver) {
      menuObserver.disconnect()
      menuObserver = null
    }
    if (mutationObserver) {
      mutationObserver.disconnect()
      mutationObserver = null
    }
  })

  return {
    platformFontOptions,
    renderPlatformFontLabel,
    renderPlatformFontOption,
    handleDropdownVisible,
    primeSelectedPreview,
  }
}
