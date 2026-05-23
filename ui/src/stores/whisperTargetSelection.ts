import type { User } from '@satorijs/protocol';

export const addWhisperTargetUnique = (currentTargets: User[], target: User): User[] => {
  const targetId = String(target?.id || '').trim();
  if (!targetId) {
    return currentTargets;
  }

  const normalizedTarget = {
    ...target,
    id: targetId,
  } as User;

  const existingIndex = currentTargets.findIndex((item) => String(item?.id || '').trim() === targetId);
  if (existingIndex === -1) {
    return [...currentTargets, normalizedTarget];
  }

  const nextTargets = currentTargets.slice();
  nextTargets.splice(existingIndex, 1, normalizedTarget);
  return nextTargets;
};
