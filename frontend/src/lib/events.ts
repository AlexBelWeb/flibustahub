// Wails event channel names. Keep in sync with internal/events.
// Convention: domain:eventName (colon, then lowerCamel).
export const Events = {
  ImportProgress: 'import:progress',
  ImportCloseRequested: 'import:closeRequested',
} as const

export type EventName = (typeof Events)[keyof typeof Events]
