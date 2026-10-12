import type { BotAccount } from '@rayleabot/plugin-ui'

export type APIRequest = <T>(path: string) => Promise<T>
export interface HistoricalRecord {
  kind: string
  ref: string
  revision: string
  source_protocol: string
  bot_id: string
  label: string
  assignable: boolean
}

export function botAccountPath(protocol: string, id: string): string {
  return `/api/bot-accounts/${encodeURIComponent(protocol)}/${encodeURIComponent(id)}`
}

// Only a detail endpoint's 404 means an account is absent.
export async function historicalRecords(records: HistoricalRecord[], request: APIRequest): Promise<HistoricalRecord[]> {
  const present = new Map<string, boolean>()
  const result: HistoricalRecord[] = []
  for (const record of records) {
    if (!record.source_protocol || !record.bot_id) {
      result.push(record)
      continue
    }
    const path = botAccountPath(record.source_protocol, record.bot_id)
    if (!present.has(path)) {
      try {
        await request<BotAccount>(path)
        present.set(path, true)
      } catch (cause) {
        if (!(cause instanceof Error) || !('status' in cause) || cause.status !== 404) throw cause
        present.set(path, false)
      }
    }
    if (!present.get(path)) result.push(record)
  }
  return result
}
