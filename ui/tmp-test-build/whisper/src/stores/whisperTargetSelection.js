"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.addWhisperTargetUnique = void 0;
const addWhisperTargetUnique = (currentTargets, target) => {
    const targetId = String(target?.id || '').trim();
    if (!targetId) {
        return currentTargets;
    }
    const normalizedTarget = {
        ...target,
        id: targetId,
    };
    const existingIndex = currentTargets.findIndex((item) => String(item?.id || '').trim() === targetId);
    if (existingIndex === -1) {
        return [...currentTargets, normalizedTarget];
    }
    const nextTargets = currentTargets.slice();
    nextTargets.splice(existingIndex, 1, normalizedTarget);
    return nextTargets;
};
exports.addWhisperTargetUnique = addWhisperTargetUnique;
