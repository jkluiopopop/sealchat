import dayjs from 'dayjs';

const PERFORMANCE_AUTOPLAY_WINDOW_SECONDS = 8;

const normalizeCreatedAt = (value: unknown): dayjs.Dayjs | null => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return dayjs(value);
  }
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) {
      return null;
    }
    if (/^\d+$/.test(trimmed)) {
      const numeric = Number(trimmed);
      if (Number.isFinite(numeric)) {
        return dayjs(numeric);
      }
    }
    const parsed = dayjs(trimmed);
    return parsed.isValid() ? parsed : null;
  }
  return null;
};

export const shouldAutoplayPerformanceMessage = (
  createdAt: unknown,
  _isSelf: boolean,
  sendStatus: 'sending' | 'sent' | 'failed',
  now = dayjs(),
): boolean => {
  if (sendStatus === 'sending') {
    return true;
  }
  const createdAtValue = normalizeCreatedAt(createdAt);
  if (!createdAtValue?.isValid()) {
    return false;
  }
  return Math.abs(now.diff(createdAtValue, 'second')) <= PERFORMANCE_AUTOPLAY_WINDOW_SECONDS;
};
