import { describe, expect, it } from 'vitest'
import { botAccountPath, historicalRecords, type APIRequest, type HistoricalRecord } from '../src/bots'

function row(bot_id: string, source_protocol = 'onebot11'): HistoricalRecord {
  return { kind: 'group', ref: bot_id || 'legacy', revision: '1', source_protocol, bot_id, label: 'group', assignable: !bot_id }
}

describe('historical account records', () => {
  it('checks complete accounts individually, including accounts beyond the directory first page', async () => {
    const requests: string[] = []
    const request: APIRequest = async <T>(path: string) => {
      requests.push(path)
      if (path.endsWith('/gone')) throw Object.assign(new Error('missing'), { status: 404 })
      return { state: 'no_connection' } as T
    }
    const records = [row(''), row('outside-first-page'), row('gone'), row('gone'), row('bot/id:opaque', 'qqofficial')]
    expect((await historicalRecords(records, request)).map(record => record.ref)).toEqual(['legacy', 'gone', 'gone'])
    expect(requests).toEqual(['/api/bot-accounts/onebot11/outside-first-page', '/api/bot-accounts/onebot11/gone', '/api/bot-accounts/qqofficial/bot%2Fid%3Aopaque'])
    expect(botAccountPath('qqofficial', 'bot:id')).toBe('/api/bot-accounts/qqofficial/bot%3Aid')
  })
  it('does not offer cleanup when the directory is unavailable', async () => {
    const failure = Object.assign(new Error('directory unavailable'), { status: 503 })
    await expect(historicalRecords([row('existing')], async () => { throw failure })).rejects.toBe(failure)
  })
  it('rechecks directory membership on refresh', async () => {
    let exists = false
    const request: APIRequest = async <T>() => {
      if (!exists) throw Object.assign(new Error('missing'), { status: 404 })
      return {} as T
    }
    expect(await historicalRecords([row('returned')], request)).toHaveLength(1)
    exists = true
    expect(await historicalRecords([row('returned')], request)).toHaveLength(0)
  })
})
