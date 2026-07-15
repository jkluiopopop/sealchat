import type {
  BridgeCharacterAppearance,
  BridgeCharacterDecoration,
  BridgeCharacterVariant,
  BridgeImageRef,
  BridgeRoleSnapshot,
  SealChatBridgeMessageEvent,
  SealChatBridgeMessagePayload,
} from './sealchatBridgeProtocol'
import { isSafeStageImageUrl } from '../views/theater/shared/stage-types'

type AvatarDecorationLike = {
  id?: string
  enabled: boolean
  decorationId?: string
  resourceAttachmentId?: string
  fallbackAttachmentId?: string
  settings?: object
}

const inlineImageTokenPattern = /\[\[(?:图片:[^\]]+|img:[^\]]+)\]\]/gi
const botStateWidgetPrefix = '[[STATE_WIDGET]]'

type TipTapNode = {
  type?: string
  text?: string
  attrs?: Record<string, unknown>
  content?: TipTapNode[]
}

const isMentionNodeType = (value: unknown): boolean => {
  const normalized = String(value || '').trim().toLowerCase()
  return normalized === 'mention' || normalized === 'satorimention'
}

type IdentityLike = {
  id?: string
  displayName?: string
  color?: string
  avatarAttachmentId?: string
  avatarAttachment?: string
  isTemporary?: boolean
  icOocOnActivate?: '' | 'ic' | 'ooc'
  avatarDecoration?: AvatarDecorationLike | null
  avatarDecorations?: AvatarDecorationLike[] | null
}

type VariantLike = {
  id?: string
  keyword?: string
  selectorEmoji?: string
  note?: string
  enabled?: boolean
  displayName?: string
  color?: string
  avatarAttachmentId?: string
  appearance?: Record<string, unknown>
  updatedAt?: string
}

type ResolvedAppearanceLike = {
  displayName?: string
  color?: string
  avatarAttachmentId?: string
  avatarDecorations?: AvatarDecorationLike[] | null
}

type BridgeMessageLike = {
  id?: string
  content?: string
  createdAt?: number
  icMode?: string
  ic_mode?: string
  isWhisper?: boolean
  is_whisper?: boolean
  identity?: IdentityLike | null
  senderRoleId?: string
  sender_role_id?: string
  sender_identity_name?: string
  sender_identity_color?: string
  sender_identity_avatar_id?: string
}

const isTipTapJson = (content: string): boolean => {
  if (!content || typeof content !== 'string') {
    return false
  }
  try {
    const parsed = JSON.parse(content)
    return Boolean(parsed && typeof parsed === 'object' && parsed.type === 'doc')
  } catch {
    return false
  }
}

const extractTipTapText = (node: TipTapNode | null | undefined): string => {
  if (!node) {
    return ''
  }
  if (typeof node.text === 'string') {
    return node.text
  }
  if (node.type === 'hardBreak') {
    return '\n'
  }
  if (isMentionNodeType(node.type)) {
    const mentionId = String(node.attrs?.id || '').trim()
    const mentionName = String(node.attrs?.name || '').trim()
    return `@${mentionName || mentionId || '用户'}`
  }
  if (Array.isArray(node.content) && node.content.length > 0) {
    const joined = node.content.map((child) => extractTipTapText(child)).join('')
    if (node.type === 'paragraph' || node.type === 'heading' || node.type === 'listItem') {
      return `${joined}\n`
    }
    return joined
  }
  return ''
}

const tiptapJsonToPlainText = (content: string): string => {
  try {
    const parsed = JSON.parse(content) as TipTapNode
    return extractTipTapText(parsed).replace(/\n+$/, '')
  } catch {
    return ''
  }
}

export const normalizeBridgePlainText = (raw: string): string => {
  const content = String(raw || '')
  if (!content) {
    return ''
  }

  const plainText = isTipTapJson(content) ? tiptapJsonToPlainText(content) : content
  const trimmedStart = plainText.trimStart()
  const withoutStateWidget = trimmedStart.startsWith(botStateWidgetPrefix)
    ? `${plainText.slice(0, plainText.length - trimmedStart.length)}${trimmedStart.slice(botStateWidgetPrefix.length).replace(/^\s*/, '')}`
    : plainText

  return withoutStateWidget.replace(inlineImageTokenPattern, '[图片]')
}

const resolveAvatarUrl = (
  resolveAttachmentUrl: (token?: string) => string,
  ...tokens: Array<string | undefined>
): string => {
  const normalizeAbsoluteUrl = (value: string): string => {
    if (!value.startsWith('//')) {
      return value
    }
    const protocol = typeof globalThis.location?.protocol === 'string' && globalThis.location.protocol
      ? globalThis.location.protocol
      : 'https:'
    return `${protocol}${value}`
  }

  for (const token of tokens) {
    if (typeof token === 'string' && token.trim()) {
      const resolved = normalizeAbsoluteUrl(resolveAttachmentUrl(token))
      if (resolved && isSafeStageImageUrl(resolved)) return resolved
    }
  }
  return ''
}

const buildImageRef = (
  token: string | undefined,
  resolveAttachmentUrl: (token?: string) => string,
  alt?: string,
): BridgeImageRef | null => {
  const resourceId = String(token || '').trim()
  if (!resourceId) return null
  const url = resolveAvatarUrl(resolveAttachmentUrl, resourceId)
  if (!url) return null
  return { resourceId, url, ...(alt ? { alt } : {}) }
}

const buildFirstImageRef = (
  tokens: Array<string | undefined>,
  resolveAttachmentUrl: (token?: string) => string,
  alt?: string,
): BridgeImageRef | null => {
  for (const token of tokens) {
    const image = buildImageRef(token, resolveAttachmentUrl, alt)
    if (image) return image
  }
  return null
}

const buildDecorations = (
  decorations: AvatarDecorationLike[] | null | undefined,
  resolveAttachmentUrl: (token?: string) => string,
): BridgeCharacterDecoration[] => (Array.isArray(decorations) ? decorations : [])
  .map((decoration, index) => {
    const primaryToken = String(decoration.resourceAttachmentId || '').trim()
    const fallbackToken = String(decoration.fallbackAttachmentId || '').trim()
    const resource = buildFirstImageRef([primaryToken, fallbackToken], resolveAttachmentUrl)
    if (!resource) return null
    const fallbackResource = fallbackToken && fallbackToken !== resource.resourceId
      ? buildImageRef(fallbackToken, resolveAttachmentUrl)
      : null
    return {
      id: String(decoration.id || decoration.decorationId || `decoration-${index}`).trim(),
      resource,
      enabled: decoration.enabled === true,
      zIndex: Number.isFinite((decoration.settings as { zIndex?: number } | undefined)?.zIndex)
        ? Number((decoration.settings as { zIndex?: number }).zIndex)
        : 1,
      settings: { ...(decoration.settings || {}) },
      extensions: {
        ...(decoration.decorationId ? { decorationId: decoration.decorationId } : {}),
        ...(fallbackResource ? { fallbackResource } : {}),
      },
    } satisfies BridgeCharacterDecoration
  })
  .filter((item): item is BridgeCharacterDecoration => Boolean(item))

const buildAppearance = ({
  displayName,
  color,
  avatarAttachmentId,
  avatarFallbackAttachmentIds,
  decorations,
  resolveAttachmentUrl,
}: {
  displayName?: string
  color?: string
  avatarAttachmentId?: string
  avatarFallbackAttachmentIds?: Array<string | undefined>
  decorations?: AvatarDecorationLike[] | null
  resolveAttachmentUrl: (token?: string) => string
}): BridgeCharacterAppearance => ({
  displayName: String(displayName || ''),
  color: String(color || ''),
  avatar: buildFirstImageRef(
    [avatarAttachmentId, ...(avatarFallbackAttachmentIds || [])],
    resolveAttachmentUrl,
    displayName,
  ),
  decorations: buildDecorations(decorations, resolveAttachmentUrl),
  extensions: {},
})

const clonePublicRecord = (value: Record<string, unknown>): Record<string, unknown> => {
  try {
    return JSON.parse(JSON.stringify(value)) as Record<string, unknown>
  } catch {
    return {}
  }
}

const buildVariantSnapshot = (
  variant: VariantLike,
  resolveAttachmentUrl: (token?: string) => string,
): BridgeCharacterVariant => {
  const avatar = buildImageRef(variant.avatarAttachmentId, resolveAttachmentUrl, variant.displayName)
  return {
    variantId: String(variant.id || ''),
    keyword: String(variant.keyword || ''),
    selectorEmoji: String(variant.selectorEmoji || ''),
    note: String(variant.note || ''),
    enabled: variant.enabled !== false,
    appearancePatch: {
      ...(variant.displayName ? { displayName: variant.displayName } : {}),
      ...(variant.color ? { color: variant.color } : {}),
      ...(avatar ? { avatar } : {}),
    },
    extensions: {
      ...(variant.appearance ? { appearance: clonePublicRecord(variant.appearance) } : {}),
      ...(variant.updatedAt ? { updatedAt: variant.updatedAt } : {}),
    },
  }
}

export const buildRoleSnapshot = ({
  identity,
  variant,
  variants = [],
  resolvedAppearance,
  isActive = false,
  revision = 0,
  updatedAt = Date.now(),
  resolveAttachmentUrl,
}: {
  identity: IdentityLike
  variant?: VariantLike | null
  variants?: VariantLike[]
  resolvedAppearance?: ResolvedAppearanceLike | null
  isActive?: boolean
  revision?: number
  updatedAt?: number
  resolveAttachmentUrl: (token?: string) => string
}): BridgeRoleSnapshot => {
  const identityDecorations = identity.avatarDecorations
    || (identity.avatarDecoration ? [identity.avatarDecoration] : [])
  const resolvedDisplayName = resolvedAppearance?.displayName || variant?.displayName || identity.displayName || ''
  const resolvedColor = resolvedAppearance?.color || variant?.color || identity.color || ''
  const resolvedAvatarAttachmentId = resolvedAppearance?.avatarAttachmentId
    || variant?.avatarAttachmentId
    || identity.avatarAttachmentId
    || identity.avatarAttachment
  const resolvedDecorations = resolvedAppearance?.avatarDecorations || identityDecorations
  const baseAppearance = buildAppearance({
    displayName: identity.displayName,
    color: identity.color,
    avatarAttachmentId: identity.avatarAttachmentId || identity.avatarAttachment,
    decorations: identityDecorations,
    resolveAttachmentUrl,
  })
  const finalAppearance = buildAppearance({
    displayName: resolvedDisplayName,
    color: resolvedColor,
    avatarAttachmentId: resolvedAvatarAttachmentId,
    avatarFallbackAttachmentIds: [
      variant?.avatarAttachmentId,
      identity.avatarAttachmentId,
      identity.avatarAttachment,
    ],
    decorations: resolvedDecorations,
    resolveAttachmentUrl,
  })
  return {
    identityId: String(identity.id || ''),
    displayName: resolvedDisplayName,
    color: resolvedColor,
    avatarUrl: finalAppearance.avatar?.url || '',
    isTemporary: Boolean(identity.isTemporary),
    icOocOnActivate: identity.icOocOnActivate || '',
    activeVariantId: variant?.id || null,
    activeVariantDisplayName: variant?.displayName || '',
    activeVariantColor: variant?.color || '',
    activeVariantAvatarUrl: buildImageRef(variant?.avatarAttachmentId, resolveAttachmentUrl)?.url || '',
    isActive,
    revision,
    updatedAt,
    baseAppearance,
    variants: variants.map((item) => buildVariantSnapshot(item, resolveAttachmentUrl)),
    resolvedAppearance: finalAppearance,
    extensions: {},
  }
}

export const buildBridgeMessagePayload = ({
  event,
  worldId,
  channelId,
  message,
  liveIdentity,
  liveVariant,
  resolveAttachmentUrl,
}: {
  event: SealChatBridgeMessageEvent
  worldId: string
  channelId: string
  message: BridgeMessageLike
  liveIdentity?: IdentityLike | null
  liveVariant?: VariantLike | null
  resolveAttachmentUrl: (token?: string) => string
}): SealChatBridgeMessagePayload => {
  const rawContent = String(message.content || '')
  const identity = message.identity || null
  const displayIdentity = liveIdentity || identity
  const normalizedMode = String(message.icMode ?? message.ic_mode ?? 'ic').toLowerCase() === 'ooc' ? 'ooc' : 'ic'

  return {
    type: 'sealchat.bridge.message',
    event,
    worldId,
    channelId,
    messageId: String(message.id || ''),
    createdAt: typeof message.createdAt === 'number' ? message.createdAt : undefined,
    icMode: normalizedMode,
    isWhisper: Boolean(message.isWhisper ?? message.is_whisper),
    identityId: displayIdentity?.id || message.senderRoleId || message.sender_role_id || null,
    displayName: liveVariant?.displayName || displayIdentity?.displayName || message.sender_identity_name || '',
    color: liveVariant?.color || displayIdentity?.color || message.sender_identity_color || '',
    avatarUrl: resolveAvatarUrl(
      resolveAttachmentUrl,
      liveVariant?.avatarAttachmentId,
      displayIdentity?.avatarAttachment,
      displayIdentity?.avatarAttachmentId,
      message.sender_identity_avatar_id,
    ),
    contentRaw: rawContent,
    contentText: normalizeBridgePlainText(rawContent),
  }
}
