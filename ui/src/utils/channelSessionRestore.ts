export type StoredChannelIcOocMode = 'ic' | 'ooc'

export interface ChannelRestorePreferenceOptions {
  storedMode?: string | null
  storedIdentityId?: string | null
  defaultIdentityId?: string | null
  icRoleId?: string | null
  oocRoleId?: string | null
  validIdentityIds?: string[]
}

export interface ChannelRestorePreferenceResult {
  mode: StoredChannelIcOocMode
  identityId: string
  preferIdentityModeMapping: boolean
}

const normalizeId = (value: unknown): string => String(value || '').trim()

export const normalizeStoredChannelIcOocMode = (value: unknown): StoredChannelIcOocMode => (
  value === 'ooc' ? 'ooc' : 'ic'
)

export const resolveChannelRestorePreference = (
  options: ChannelRestorePreferenceOptions,
): ChannelRestorePreferenceResult => {
  const mode = normalizeStoredChannelIcOocMode(options.storedMode)
  const validIdentityIds = Array.isArray(options.validIdentityIds)
    ? options.validIdentityIds.map((id) => normalizeId(id)).filter(Boolean)
    : []
  const validSet = new Set(validIdentityIds)
  const mappedIdentityId = normalizeId(mode === 'ooc' ? options.oocRoleId : options.icRoleId)
  const storedIdentityId = normalizeId(options.storedIdentityId)
  const defaultIdentityId = normalizeId(options.defaultIdentityId)
  const preferredMappedIdentityId = mappedIdentityId && validSet.has(mappedIdentityId) ? mappedIdentityId : ''
  const preferredStoredIdentityId = storedIdentityId && validSet.has(storedIdentityId) ? storedIdentityId : ''
  const fallbackIdentityId = defaultIdentityId && validSet.has(defaultIdentityId)
    ? defaultIdentityId
    : (validIdentityIds[0] || '')

  return {
    mode,
    identityId: preferredMappedIdentityId || preferredStoredIdentityId || fallbackIdentityId,
    preferIdentityModeMapping: !!preferredMappedIdentityId,
  }
}
