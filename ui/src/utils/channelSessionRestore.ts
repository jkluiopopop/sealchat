export type StoredChannelIcOocMode = 'ic' | 'ooc'

export interface ChannelRestorePreferenceOptions {
  currentIdentityId?: string | null
  storedMode?: string | null
  storedIdentityId?: string | null
  defaultIdentityId?: string | null
  icRoleId?: string | null
  oocRoleId?: string | null
  preferStoredIdentity?: boolean
  validIdentityIds?: string[]
}

export interface ChannelRestorePreferenceResult {
  mode: StoredChannelIcOocMode
  identityId: string
  preferIdentityModeMapping: boolean
}

export interface ChannelSessionRestoreStrategyOptions {
  splitFilter?: string | null
  storedMode?: string | null
}

export interface ChannelSessionRestoreStrategyResult {
  mode: StoredChannelIcOocMode
  useStoredIdentity: boolean
}

const normalizeId = (value: unknown): string => String(value || '').trim()

export const normalizeStoredChannelIcOocMode = (value: unknown): StoredChannelIcOocMode => (
  value === 'ooc' ? 'ooc' : 'ic'
)

export const resolveChannelSessionRestoreStrategy = (
  options: ChannelSessionRestoreStrategyOptions,
): ChannelSessionRestoreStrategyResult => {
  const splitFilter = options.splitFilter === 'ic' || options.splitFilter === 'ooc'
    ? options.splitFilter
    : 'all'
  if (splitFilter === 'ic' || splitFilter === 'ooc') {
    return {
      mode: splitFilter,
      useStoredIdentity: false,
    }
  }
  return {
    mode: normalizeStoredChannelIcOocMode(options.storedMode),
    useStoredIdentity: true,
  }
}

export const resolveChannelRestorePreference = (
  options: ChannelRestorePreferenceOptions,
): ChannelRestorePreferenceResult => {
  const mode = normalizeStoredChannelIcOocMode(options.storedMode)
  const validIdentityIds = Array.isArray(options.validIdentityIds)
    ? options.validIdentityIds.map((id) => normalizeId(id)).filter(Boolean)
    : []
  const validSet = new Set(validIdentityIds)
  const currentIdentityId = normalizeId(options.currentIdentityId)
  const mappedIdentityId = normalizeId(mode === 'ooc' ? options.oocRoleId : options.icRoleId)
  const storedIdentityId = normalizeId(options.storedIdentityId)
  const defaultIdentityId = normalizeId(options.defaultIdentityId)
  const preferredCurrentIdentityId = currentIdentityId && validSet.has(currentIdentityId) ? currentIdentityId : ''
  const preferredMappedIdentityId = mappedIdentityId && validSet.has(mappedIdentityId) ? mappedIdentityId : ''
  const preferredStoredIdentityId = storedIdentityId && validSet.has(storedIdentityId) ? storedIdentityId : ''
  const fallbackIdentityId = defaultIdentityId && validSet.has(defaultIdentityId)
    ? defaultIdentityId
    : (validIdentityIds[0] || '')
  const preferStoredIdentity = options.preferStoredIdentity === true
  const preservedCurrentIdentity = preferredCurrentIdentityId || ''

  return {
    mode,
    identityId: preservedCurrentIdentity || (
      preferStoredIdentity
        ? (preferredStoredIdentityId || preferredMappedIdentityId || fallbackIdentityId)
        : (preferredMappedIdentityId || preferredStoredIdentityId || fallbackIdentityId)
    ),
    preferIdentityModeMapping: preservedCurrentIdentity ? false : (preferStoredIdentity ? false : !!preferredMappedIdentityId),
  }
}
