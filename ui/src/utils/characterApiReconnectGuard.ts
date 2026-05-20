import { isBotCommandLikeContent } from './botCommand'
import { shouldAttemptCharacterApiReconnect } from './characterApiReconnectDecision'

export interface CharacterApiReconnectGuardInput {
  content: string
  botCommandPrefixes?: unknown
  botFeatureEnabled?: boolean
  isBotPrivateChat?: boolean
  characterApiReady?: boolean
  hadSuccessfulCharacterApiSession?: boolean
}

export const shouldAttemptCharacterApiReconnectBeforeBotCommand = (
  input: CharacterApiReconnectGuardInput,
): boolean => {
  return shouldAttemptCharacterApiReconnect({
    isBotCommand: isBotCommandLikeContent(input.content, input.botCommandPrefixes),
    botFeatureEnabled: input.botFeatureEnabled,
    isBotPrivateChat: input.isBotPrivateChat,
    characterApiReady: input.characterApiReady,
    hadSuccessfulCharacterApiSession: input.hadSuccessfulCharacterApiSession,
  })
}
