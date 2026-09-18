// Wails event channel names. Keep in sync with internal/events.
// Convention: domain:eventName (colon, then lowerCamel).
export const Events = {
  ImportProgress: 'import:progress',
  ImportCloseRequested: 'import:closeRequested',
  DBUpdated: 'db:updated',
  SearchIndexReady: 'search:indexReady',
} as const

export type EventName = (typeof Events)[keyof typeof Events]
