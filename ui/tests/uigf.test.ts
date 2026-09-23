import { describe, expect, it } from 'vitest'
import { exportUIGF, parseImport } from '../src/uigf'

const sample = { id: '1844674407370955101', item_id: '10001', name: 'Synthetic item', time: '2026-09-01 08:00:00', rank_type: '5', count: '1' }

describe('UIGF records', () => {
  it('round-trips without changing large IDs or local time', () => {
    const input = { info: { version: 'v4.1', authkey: 'synthetic-must-not-transfer' }, hk4e: [{ uid: 100000001, timezone: 8, lang: 'zh-cn', list: [{ ...sample, gacha_type: '400', gacha_id: '9001', cookie_token: 'synthetic-must-not-transfer' }] }] }
    const [archive] = parseImport(JSON.stringify(input), -5)
    expect(archive!.timezone).toBe(8)
    expect(archive!.list[0]!.id).toBe(sample.id)
    expect(archive!.list[0]!.time).toBe(sample.time)
    const output = exportUIGF(archive!)
    expect(JSON.stringify(output)).not.toContain('synthetic-must-not-transfer')
    expect(parseImport(JSON.stringify(output), 0)).toEqual([archive])
  })

  it('keeps other games out of the selected archive', () => {
    expect(parseImport(JSON.stringify({ info: { version: 'v4.0' }, hkrpg: [] }), 8)).toEqual([])
    expect(parseImport(JSON.stringify({ info: { format: 'raylea-gacha', version: 1 }, game: 'zzz', archive: { uid: '100000001', timezone: 0, lang: 'zh-cn', list: [{ ...sample, gacha_type: '102' }] } }), 8)).toEqual([])
  })

  it('uses an explicit legacy timezone and rejects lossy numeric record IDs', () => {
    const legacy = { info: { uigf_version: 'v3.0', uid: '100000001' }, list: [{ ...sample, gacha_type: '301' }] }
    expect(parseImport(JSON.stringify(legacy), -5)[0]!.timezone).toBe(-5)
    legacy.list[0]!.id = 1844674407370955101 as unknown as string
    expect(() => parseImport(JSON.stringify(legacy), 8)).toThrow('字符串')
  })
})
