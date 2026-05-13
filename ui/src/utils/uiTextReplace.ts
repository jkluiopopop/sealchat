import type { UITextReplaceConfig, UITextReplaceRule } from '../types'

export type PreparedUITextReplaceRule = {
  id: string
  searchText: string
  replaceText: string
}

export type UITextReplaceIgnoredContext = {
  tagName?: string
  classNames?: string[]
  ancestorSelectors?: string[]
  isContentEditable?: boolean
}

type IdleWindow = Window & {
  requestIdleCallback?: (callback: IdleRequestCallback, options?: IdleRequestOptions) => number
  cancelIdleCallback?: (handle: number) => void
}

const DEFAULT_RULES: UITextReplaceRule[] = [
  { id: 'default-world-lobby', searchText: '世界大厅', replaceText: '世界大厅', enabled: true },
  { id: 'default-world-manage', searchText: '世界管理', replaceText: '世界管理', enabled: true },
  { id: 'default-glossary-manage', searchText: '术语管理', replaceText: '术语管理', enabled: true },
  { id: 'default-announcement', searchText: '公告', replaceText: '公告', enabled: true },
]

const TRACKED_ATTRIBUTE_NAMES = ['placeholder', 'title', 'aria-label', 'aria-placeholder', 'alt'] as const

const SHARED_IGNORE_SELECTORS = [
  '[data-ui-text-replace-ignore]',
  '[data-message-id]',
  '.message-row__grid',
  '.message-list',
  '.message-item',
  '.chat-input-wrapper',
  '.chat-input-container',
  '.chat-input-area',
  '.chat-input-editor-main',
  '.chat-input-plain-wrapper',
  '.hybrid-input',
  '.tiptap-editor',
  '.announcement-rich-html',
  '.sticky-note-editor__wrapper',
  '.sticky-note__rich-input',
  '.ProseMirror',
]

const TEXT_IGNORE_SELECTOR = [
  ...SHARED_IGNORE_SELECTORS,
  'input',
  'textarea',
  '[contenteditable="true"]',
].join(', ')

const ATTRIBUTE_IGNORE_SELECTOR = [
  ...SHARED_IGNORE_SELECTORS,
  '[contenteditable="true"]',
].join(', ')

const TRACKED_ATTRIBUTE_SELECTOR = TRACKED_ATTRIBUTE_NAMES.map((item) => `[${item}]`).join(', ')

const IGNORE_CLASS_NAMES = new Set([
  'chat-input-wrapper',
  'chat-input-container',
  'chat-input-area',
  'chat-input-editor-main',
  'chat-input-plain-wrapper',
  'hybrid-input',
  'tiptap-editor',
  'announcement-rich-html',
  'sticky-note-editor__wrapper',
  'sticky-note__rich-input',
  'message-list',
  'message-item',
  'message-row__grid',
])

const IGNORE_ANCESTOR_MATCHES = new Set([
  '[data-message-id]',
  '.chat-input-wrapper',
  '.chat-input-container',
  '.chat-input-area',
  '.chat-input-editor-main',
  '.chat-input-plain-wrapper',
  '.hybrid-input',
  '.tiptap-editor',
  '.announcement-rich-html',
  '.sticky-note-editor__wrapper',
  '.sticky-note__rich-input',
])

const cloneDefaultRules = () => DEFAULT_RULES.map((item) => ({ ...item }))

export const normalizeUITextReplaceConfig = (value?: UITextReplaceConfig | null): UITextReplaceConfig => {
  const sourceRules = Array.isArray(value?.rules) && value!.rules.length > 0 ? value!.rules : cloneDefaultRules()
  const rules = sourceRules
    .map((item, index) => ({
      id: String(item?.id || '').trim() || `ui-text-replace-${index + 1}`,
      searchText: String(item?.searchText || '').trim(),
      replaceText: String(item?.replaceText || '').trim(),
      enabled: item?.enabled !== false,
    }))
    .filter((item) => item.searchText.length > 0)
  return {
    enabled: value?.enabled === true,
    rules: rules.length > 0 ? rules : cloneDefaultRules(),
  }
}

export const prepareUITextReplaceRules = (config?: UITextReplaceConfig | null): PreparedUITextReplaceRule[] => {
  const normalized = normalizeUITextReplaceConfig(config)
  return normalized.rules
    .filter((item) => item.enabled && item.searchText && item.searchText !== item.replaceText)
    .map((item) => ({
      id: item.id,
      searchText: item.searchText,
      replaceText: item.replaceText,
    }))
    .sort((a, b) => b.searchText.length - a.searchText.length)
}

export const applyUITextReplaceRules = (text: string, rules: PreparedUITextReplaceRule[]): string => {
  let nextText = text
  for (const rule of rules) {
    if (!rule.searchText || nextText.includes(rule.searchText) === false) continue
    nextText = nextText.split(rule.searchText).join(rule.replaceText)
  }
  return nextText
}

export const isUITextReplaceIgnoredContext = (context: UITextReplaceIgnoredContext): boolean => {
  const tagName = String(context.tagName || '').toUpperCase()
  if (tagName === 'INPUT' || tagName === 'TEXTAREA' || tagName === 'SCRIPT' || tagName === 'STYLE') {
    return true
  }
  if (context.isContentEditable) {
    return true
  }
  if ((context.classNames || []).some((item) => IGNORE_CLASS_NAMES.has(item))) {
    return true
  }
  return (context.ancestorSelectors || []).some((item) => IGNORE_ANCESTOR_MATCHES.has(item))
}

const getRootElement = (): HTMLElement | null => {
  if (typeof document === 'undefined') return null
  return document.body || document.getElementById('app')
}

const getIdleWindow = (): IdleWindow | null => {
  if (typeof window === 'undefined') return null
  return window as IdleWindow
}

const matchesIgnoredTextDomContext = (element: Element | null): boolean => {
  if (!element) return true
  if (element instanceof HTMLElement && isUITextReplaceIgnoredContext({
    tagName: element.tagName,
    classNames: Array.from(element.classList || []),
    ancestorSelectors: [],
    isContentEditable: element.isContentEditable,
  })) {
    return true
  }
  return Boolean(element.closest(TEXT_IGNORE_SELECTOR))
}

const matchesIgnoredAttributeDomContext = (element: Element | null): boolean => {
  if (!element) return true
  if (element instanceof HTMLElement && element.isContentEditable) {
    return true
  }
  return Boolean(element.closest(ATTRIBUTE_IGNORE_SELECTOR))
}

class UITextReplaceRuntime {
  private observer: MutationObserver | null = null
  private idleHandle: number | null = null
  private flushTimer: number | null = null
  private pendingNodes = new Set<Node>()
  private trackedNodes = new Set<Text>()
  private trackedAttributeElements = new Set<Element>()
  private originalTextMap = new WeakMap<Text, string>()
  private originalAttributeMap = new WeakMap<Element, Map<string, string | null>>()
  private selfMutatingNodes = new WeakSet<Text>()
  private selfMutatingAttributes = new WeakMap<Element, Set<string>>()
  private activeRules: PreparedUITextReplaceRule[] = []
  private configSignature = ''

  apply(config?: UITextReplaceConfig | null) {
    const normalized = normalizeUITextReplaceConfig(config)
    const nextSignature = JSON.stringify(normalized)
    if (nextSignature === this.configSignature) return

    this.restoreTrackedNodes()
    this.restoreTrackedAttributes()
    this.configSignature = nextSignature
    this.activeRules = prepareUITextReplaceRules(normalized)

    if (!normalized.enabled || this.activeRules.length === 0) {
      this.stopObserver()
      return
    }

    this.startObserver()
    this.scheduleFullScan()
  }

  private scheduleFullScan() {
    const root = getRootElement()
    if (!root) return
    this.cancelIdleWork()
    const idleWindow = getIdleWindow()
    if (idleWindow?.requestIdleCallback) {
      this.idleHandle = idleWindow.requestIdleCallback(() => {
        this.idleHandle = null
        this.queueNode(root)
      }, { timeout: 400 })
      return
    }
    this.idleHandle = window.setTimeout(() => {
      this.idleHandle = null
      this.queueNode(root)
    }, 180)
  }

  private startObserver() {
    const root = getRootElement()
    if (!root || this.observer) return
    this.observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        if (mutation.type === 'characterData') {
          const target = mutation.target
          if (!(target instanceof Text)) continue
          if (this.selfMutatingNodes.has(target)) {
            this.selfMutatingNodes.delete(target)
            continue
          }
          if (this.trackedNodes.has(target)) {
            this.originalTextMap.set(target, target.nodeValue || '')
          }
          this.queueNode(target)
          continue
        }
        if (mutation.type === 'attributes') {
          const target = mutation.target
          const attributeName = mutation.attributeName
          if (!(target instanceof Element) || !attributeName) continue
          if (this.consumeSelfMutatingAttribute(target, attributeName)) {
            continue
          }
          if (this.hasOriginalAttributeValue(target, attributeName)) {
            this.setOriginalAttributeValue(target, attributeName, target.getAttribute(attributeName))
          }
          this.queueNode(target)
          continue
        }
        mutation.addedNodes.forEach((node) => this.queueNode(node))
      }
    })
    this.observer.observe(root, {
      attributes: true,
      attributeFilter: TRACKED_ATTRIBUTE_NAMES as unknown as string[],
      childList: true,
      characterData: true,
      subtree: true,
    })
  }

  private stopObserver() {
    this.cancelIdleWork()
    this.cancelFlushWork()
    this.pendingNodes.clear()
    if (this.observer) {
      this.observer.disconnect()
      this.observer = null
    }
  }

  private cancelIdleWork() {
    if (this.idleHandle === null) return
    const idleWindow = getIdleWindow()
    if (idleWindow?.cancelIdleCallback) {
      idleWindow.cancelIdleCallback(this.idleHandle)
    } else {
      window.clearTimeout(this.idleHandle)
    }
    this.idleHandle = null
  }

  private cancelFlushWork() {
    if (this.flushTimer === null) return
    window.clearTimeout(this.flushTimer)
    this.flushTimer = null
  }

  private queueNode(node: Node | null) {
    if (!node || this.activeRules.length === 0) return
    this.pendingNodes.add(node)
    if (this.flushTimer !== null) return
    this.flushTimer = window.setTimeout(() => {
      this.flushTimer = null
      this.flushPendingNodes()
    }, 32)
  }

  private flushPendingNodes() {
    if (this.activeRules.length === 0 || this.pendingNodes.size === 0) return
    const nodes = Array.from(this.pendingNodes)
    this.pendingNodes.clear()
    for (const node of nodes) {
      this.processNode(node)
    }
  }

  private processNode(node: Node) {
    if (node instanceof Text) {
      this.processTextNode(node)
      return
    }
    if (!(node instanceof Element) && !(node instanceof DocumentFragment)) {
      return
    }

    if (node instanceof Element) {
      this.processElementAttributes(node)
    }

    if (node instanceof Element || node instanceof DocumentFragment) {
      this.processSubtreeAttributes(node)
    }

    if (node instanceof Element && matchesIgnoredTextDomContext(node)) {
      return
    }

    const walker = document.createTreeWalker(node, NodeFilter.SHOW_TEXT)
    let current = walker.nextNode()
    while (current) {
      if (current instanceof Text) {
        this.processTextNode(current)
      }
      current = walker.nextNode()
    }
  }

  private processTextNode(node: Text) {
    const parentElement = node.parentElement
    if (!parentElement || matchesIgnoredTextDomContext(parentElement)) return

    const currentText = node.nodeValue || ''
    if (!currentText.trim()) return

    const baseText = this.originalTextMap.get(node) ?? currentText
    const nextText = applyUITextReplaceRules(baseText, this.activeRules)

    if (nextText === baseText) {
      if (currentText !== baseText) {
        this.writeTextNode(node, baseText)
      }
      this.trackedNodes.delete(node)
      return
    }

    this.originalTextMap.set(node, baseText)
    this.trackedNodes.add(node)
    if (currentText !== nextText) {
      this.writeTextNode(node, nextText)
    }
  }

  private writeTextNode(node: Text, value: string) {
    this.selfMutatingNodes.add(node)
    node.nodeValue = value
  }

  private processSubtreeAttributes(root: Element | DocumentFragment) {
    if (!TRACKED_ATTRIBUTE_SELECTOR) return
    const candidates = root.querySelectorAll(TRACKED_ATTRIBUTE_SELECTOR)
    candidates.forEach((element) => this.processElementAttributes(element))
  }

  private processElementAttributes(element: Element) {
    for (const attributeName of TRACKED_ATTRIBUTE_NAMES) {
      this.processElementAttribute(element, attributeName)
    }
  }

  private processElementAttribute(element: Element, attributeName: string) {
    if (matchesIgnoredAttributeDomContext(element)) return

    const hasAttribute = element.hasAttribute(attributeName)
    const currentValue = hasAttribute ? element.getAttribute(attributeName) : null
    const hasOriginalValue = this.hasOriginalAttributeValue(element, attributeName)
    const baseValue = hasOriginalValue ? this.getOriginalAttributeValue(element, attributeName) : currentValue

    if (baseValue === undefined) return
    if (typeof baseValue === 'string' && !baseValue.trim()) return

    const nextValue = typeof baseValue === 'string'
      ? applyUITextReplaceRules(baseValue, this.activeRules)
      : baseValue

    if (nextValue === baseValue) {
      if (currentValue !== baseValue) {
        this.writeElementAttribute(element, attributeName, baseValue)
      }
      this.deleteOriginalAttributeValue(element, attributeName)
      return
    }

    this.setOriginalAttributeValue(element, attributeName, baseValue)
    if (currentValue !== nextValue) {
      this.writeElementAttribute(element, attributeName, nextValue)
    }
  }

  private writeElementAttribute(element: Element, attributeName: string, value: string | null) {
    this.markSelfMutatingAttribute(element, attributeName)
    if (value === null) {
      element.removeAttribute(attributeName)
      return
    }
    element.setAttribute(attributeName, value)
  }

  private markSelfMutatingAttribute(element: Element, attributeName: string) {
    let current = this.selfMutatingAttributes.get(element)
    if (!current) {
      current = new Set<string>()
      this.selfMutatingAttributes.set(element, current)
    }
    current.add(attributeName)
  }

  private consumeSelfMutatingAttribute(element: Element, attributeName: string): boolean {
    const current = this.selfMutatingAttributes.get(element)
    if (!current?.has(attributeName)) return false
    current.delete(attributeName)
    return true
  }

  private hasOriginalAttributeValue(element: Element, attributeName: string): boolean {
    const current = this.originalAttributeMap.get(element)
    return current?.has(attributeName) === true
  }

  private getOriginalAttributeValue(element: Element, attributeName: string): string | null | undefined {
    const current = this.originalAttributeMap.get(element)
    if (!current?.has(attributeName)) return undefined
    return current.get(attributeName) ?? null
  }

  private setOriginalAttributeValue(element: Element, attributeName: string, value: string | null) {
    let current = this.originalAttributeMap.get(element)
    if (!current) {
      current = new Map<string, string | null>()
      this.originalAttributeMap.set(element, current)
    }
    current.set(attributeName, value)
    this.trackedAttributeElements.add(element)
  }

  private deleteOriginalAttributeValue(element: Element, attributeName: string) {
    const current = this.originalAttributeMap.get(element)
    if (!current?.has(attributeName)) return
    current.delete(attributeName)
    if (current.size === 0) {
      this.trackedAttributeElements.delete(element)
    }
  }

  private restoreTrackedNodes() {
    for (const node of Array.from(this.trackedNodes)) {
      if (!node.isConnected) {
        this.trackedNodes.delete(node)
        continue
      }
      const original = this.originalTextMap.get(node)
      if (typeof original === 'string' && node.nodeValue !== original) {
        this.writeTextNode(node, original)
      }
      this.trackedNodes.delete(node)
    }
  }

  private restoreTrackedAttributes() {
    for (const element of Array.from(this.trackedAttributeElements)) {
      if (!element.isConnected) {
        this.trackedAttributeElements.delete(element)
        continue
      }
      const current = this.originalAttributeMap.get(element)
      if (!current) {
        this.trackedAttributeElements.delete(element)
        continue
      }
      for (const [attributeName, originalValue] of current.entries()) {
        const currentValue = element.getAttribute(attributeName)
        if (currentValue !== originalValue) {
          this.writeElementAttribute(element, attributeName, originalValue)
        }
      }
      current.clear()
      this.trackedAttributeElements.delete(element)
    }
  }
}

const runtime = new UITextReplaceRuntime()

export const applyUITextReplaceConfig = (config?: UITextReplaceConfig | null) => {
  runtime.apply(config)
}
