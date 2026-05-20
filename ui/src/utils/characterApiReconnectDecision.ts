export interface CharacterApiReconnectDecisionInput {
  isBotCommand: boolean
  botFeatureEnabled?: boolean
  isBotPrivateChat?: boolean
  characterApiReady?: boolean
  hadSuccessfulCharacterApiSession?: boolean
}

export const shouldAttemptCharacterApiReconnect = (
  input: CharacterApiReconnectDecisionInput,
): boolean => {
  if (!input.isBotCommand) {
    return false
  }
  if (input.botFeatureEnabled !== true && input.isBotPrivateChat !== true) {
    return false
  }
  if (input.characterApiReady === true) {
    return false
  }
  return input.hadSuccessfulCharacterApiSession === true
}
