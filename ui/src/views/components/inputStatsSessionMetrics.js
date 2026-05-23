const toTimestamp = (value) => {
  return value instanceof Date ? value.getTime() : new Date(value).getTime()
}

export function buildInputStatSessions(messages, thresholdMinutes) {
  if (!Array.isArray(messages) || messages.length === 0) {
    return []
  }

  const thresholdMs = thresholdMinutes * 60 * 1000
  const result = []

  let sessionStart = 0
  let sessionChars = 0
  let sessionMsgs = 0

  for (let i = 0; i < messages.length; i++) {
    const curTime = toTimestamp(messages[i].createdAt)

    if (i === 0) {
      sessionStart = curTime
      sessionChars = messages[i].charCount || 0
      sessionMsgs = 1
      continue
    }

    const prevTime = toTimestamp(messages[i - 1].createdAt)
    const gap = curTime - prevTime

    if (gap > thresholdMs) {
      const durationMin = (prevTime - sessionStart) / 60000
      result.push({
        index: result.length + 1,
        startTime: new Date(sessionStart).toLocaleString(),
        endTime: new Date(prevTime).toLocaleString(),
        duration: Math.round(durationMin * 10) / 10,
        occupiedDuration: Math.round((durationMin + thresholdMinutes) * 10) / 10,
        totalChars: sessionChars,
        totalMessages: sessionMsgs,
        typingSpeed: durationMin > 0 ? Math.round(sessionChars / durationMin * 10) / 10 : 0,
      })
      sessionStart = curTime
      sessionChars = messages[i].charCount || 0
      sessionMsgs = 1
    } else {
      sessionChars += messages[i].charCount || 0
      sessionMsgs++
    }
  }

  if (sessionMsgs > 0) {
    const endTime = toTimestamp(messages[messages.length - 1].createdAt)
    const durationMin = (endTime - sessionStart) / 60000
    result.push({
      index: result.length + 1,
      startTime: new Date(sessionStart).toLocaleString(),
      endTime: new Date(endTime).toLocaleString(),
      duration: Math.round(durationMin * 10) / 10,
      occupiedDuration: Math.round((durationMin + thresholdMinutes) * 10) / 10,
      totalChars: sessionChars,
      totalMessages: sessionMsgs,
      typingSpeed: durationMin > 0 ? Math.round(sessionChars / durationMin * 10) / 10 : 0,
    })
  }

  result.forEach((session, index) => {
    session.index = index + 1
  })

  return result
}

export function calcSessionSummary(sessions) {
  return sessions.reduce((acc, session) => {
    acc.duration += session.duration
    acc.totalChars += session.totalChars
    acc.totalMessages += session.totalMessages
    return acc
  }, {
    duration: 0,
    totalChars: 0,
    totalMessages: 0,
  })
}

export function calcOccupiedWindowSummary(messages, thresholdMinutes) {
  return {
    duration: calcOccupiedWindowMinutes(messages, thresholdMinutes),
    totalChars: (messages || []).reduce((sum, message) => sum + (message?.charCount || 0), 0),
    totalMessages: Array.isArray(messages) ? messages.length : 0,
  }
}

export function calcOccupiedWindowMinutes(messages, thresholdMinutes) {
  if (!Array.isArray(messages) || messages.length === 0) {
    return 0
  }

  const thresholdMs = Math.max(0, thresholdMinutes) * 60 * 1000
  const windows = messages
    .map((message) => {
      const start = toTimestamp(message.createdAt)
      return { start, end: start + thresholdMs }
    })
    .sort((a, b) => a.start - b.start)

  let totalMs = 0
  let currentStart = windows[0].start
  let currentEnd = windows[0].end

  for (let i = 1; i < windows.length; i++) {
    const next = windows[i]
    if (next.start <= currentEnd) {
      currentEnd = Math.max(currentEnd, next.end)
      continue
    }
    totalMs += currentEnd - currentStart
    currentStart = next.start
    currentEnd = next.end
  }

  totalMs += currentEnd - currentStart
  return Math.round(totalMs / 60000)
}
