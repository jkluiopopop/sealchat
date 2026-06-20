export interface ApplyCalendarActiveDayHighlightsOptions {
  panel: HTMLElement | null
  activeDaySetByMonth: Record<string, Set<string>>
}

const ACTIVE_DAY_CLASS = 'sc-export-active-day'
const ACTIVE_DAY_ATTR = 'data-sc-export-active-day'

const normalizeMonthKey = (date: Date) => {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  return `${year}-${month}`
}

const buildDayKey = (date: Date) => {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

export const parsePanelMonthKey = (text: string) => {
  const match = text.match(/(\d{4})\s*年\s*(\d{1,2})\s*月/)
  if (!match) {
    return ''
  }
  return `${match[1]}-${match[2].padStart(2, '0')}`
}

export const buildPanelDates = (monthKey: string, cellCount: number) => {
  const [yearText, monthText] = monthKey.split('-')
  const year = Number(yearText)
  const month = Number(monthText)
  if (!Number.isInteger(year) || !Number.isInteger(month)) {
    return []
  }
  const firstOfMonth = new Date(year, month - 1, 1)
  const firstVisible = new Date(firstOfMonth)
  firstVisible.setDate(firstVisible.getDate() - 1)
  let protectLastMonthDateIsShownFlag = true
  while (firstVisible.getDay() !== 0 || protectLastMonthDateIsShownFlag) {
    firstVisible.setDate(firstVisible.getDate() - 1)
    protectLastMonthDateIsShownFlag = false
  }
  firstVisible.setDate(firstVisible.getDate() + 1)
  const dates: Date[] = []
  for (let index = 0; index < cellCount; index += 1) {
    const next = new Date(firstVisible)
    next.setDate(firstVisible.getDate() + index)
    dates.push(next)
  }
  return dates
}

export const buildPanelMonthActiveDayMap = (input: Record<string, string[]>) => {
  const result: Record<string, Set<string>> = {}
  Object.entries(input).forEach(([month, days]) => {
    result[month] = new Set(days || [])
  })
  return result
}

export const collectPanelRelatedMonthKeys = (monthKey: string, cellCount: number) => {
  const related = new Set<string>()
  buildPanelDates(monthKey, cellCount).forEach((date) => {
    related.add(normalizeMonthKey(date))
  })
  return Array.from(related)
}

const clearCalendarActiveDayHighlights = (panel: HTMLElement) => {
  panel.querySelectorAll<HTMLElement>(`.${ACTIVE_DAY_CLASS}, [${ACTIVE_DAY_ATTR}]`).forEach((cell) => {
    cell.classList.remove(ACTIVE_DAY_CLASS)
    cell.removeAttribute(ACTIVE_DAY_ATTR)
  })
}

export const applyCalendarActiveDayHighlights = ({
  panel,
  activeDaySetByMonth,
}: ApplyCalendarActiveDayHighlightsOptions) => {
  if (!panel) {
    return false
  }
  clearCalendarActiveDayHighlights(panel)
  let highlighted = false
  const calendars = Array.from(panel.querySelectorAll<HTMLElement>('.n-date-panel-calendar'))
  calendars.forEach((calendar) => {
    const monthKey = parsePanelMonthKey(
      calendar.querySelector<HTMLElement>('.n-date-panel-month__month-year')?.textContent || ''
    )
    if (!monthKey) {
      return
    }
    const cells = Array.from(calendar.querySelectorAll<HTMLElement>('[data-n-date].n-date-panel-date'))
    const dates = buildPanelDates(monthKey, cells.length)
    cells.forEach((cell, index) => {
      const date = dates[index]
      if (!date) {
        return
      }
      const dateMonthKey = normalizeMonthKey(date)
      const activeDays = activeDaySetByMonth[dateMonthKey]
      if (!activeDays || activeDays.size === 0) {
        return
      }
      const dayKey = buildDayKey(date)
      if (!activeDays.has(dayKey)) {
        return
      }
      cell.classList.add(ACTIVE_DAY_CLASS)
      cell.setAttribute(ACTIVE_DAY_ATTR, 'true')
      highlighted = true
    })
  })
  return highlighted
}
