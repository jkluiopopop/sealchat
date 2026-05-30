export type DiceMode = 'builtin' | 'bot' | 'disabled';

export const resolveDiceMode = (input: {
  builtInDiceEnabled: boolean;
  botFeatureEnabled: boolean;
  isBotPrivateChatChannel?: boolean;
}): DiceMode => {
  if (input.isBotPrivateChatChannel) {
    return 'bot';
  }
  if (input.botFeatureEnabled) {
    return 'bot';
  }
  if (input.builtInDiceEnabled) {
    return 'builtin';
  }
  return 'disabled';
};

export const shouldShowDiceTrayTrigger = (input: {
  builtInDiceEnabled: boolean;
  botFeatureEnabled: boolean;
  isBotPrivateChatChannel?: boolean;
}): boolean => resolveDiceMode(input) !== 'disabled';

export const getDiceModeLabel = (input: {
  builtInDiceEnabled: boolean;
  botFeatureEnabled: boolean;
  isBotPrivateChatChannel?: boolean;
}): string => {
  const mode = resolveDiceMode(input);
  if (mode === 'bot') return 'BOT掷骰';
  if (mode === 'disabled') return '已关闭掷骰';
  return '内置掷骰';
};
