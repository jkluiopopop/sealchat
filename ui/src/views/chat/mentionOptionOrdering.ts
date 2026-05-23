export type MentionableIdentityType = 'ic' | 'ooc' | 'user';

export interface MentionableMemberLite {
  userId: string;
  displayName: string;
  identityType: MentionableIdentityType;
}

export function shouldResetMentionOptionsOnSearchStart(prefix: string) {
  return prefix === '@';
}

function mentionableIdentityWeight(identityType: MentionableIdentityType, currentMode: 'ic' | 'ooc') {
  if (identityType === 'user') {
    return 2;
  }
  if (identityType === currentMode) {
    return 0;
  }
  return 1;
}

export function sortMentionableMembersByMode<T extends MentionableMemberLite>(
  items: T[],
  currentMode: 'ic' | 'ooc',
): T[] {
  if (!Array.isArray(items) || items.length <= 1) {
    return Array.isArray(items) ? items.slice() : [];
  }
  return items
    .map((item, index) => ({ item, index }))
    .sort((left, right) => {
      const leftWeight = mentionableIdentityWeight(left.item.identityType, currentMode);
      const rightWeight = mentionableIdentityWeight(right.item.identityType, currentMode);
      if (leftWeight !== rightWeight) {
        return leftWeight - rightWeight;
      }
      return left.index - right.index;
    })
    .map(({ item }) => item);
}
