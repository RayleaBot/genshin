// Converts miao-plugin's material, banner and birthday data. Only the
// data-only modules are evaluated; no plugin code is loaded.
//
// Usage: node scripts/import-resource-data.mjs <参考项目/2026-09-15>
// Writes internal/assets/data/resources.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const meta = path.join(path.resolve(process.argv[2]), 'miao-plugin/resources/meta-gs')

// dataModule evaluates a module without imports and returns expression.
async function dataModule(file, expression) {
  let source = await fs.readFile(file, 'utf8')
  if (/^\s*import\s/m.test(source)) {
    throw Error('Data-only module required')
  }
  source = source.replace(/^export\s+(?=const\s)/gm, '').replace(/^export default/m, 'const defaultData =')
  const context = vm.createContext(Object.create(null), { codeGeneration: { strings: false, wasm: false } })
  return JSON.parse(new vm.Script(`${source}\nJSON.stringify(${expression})`).runInContext(context, { timeout: 1000 }))
}

// Talent books and weapon materials open on two weekdays and Sunday; daily.js
// lists each city's material per weekday, the cities in this order.
const daily = await dataModule(path.join(meta, 'material/daily.js'), 'defaultData')
const cities = ['蒙德', '璃月', '稻妻', '须弥', '枫丹', '纳塔', '挪德卡莱', '至冬']
const families = JSON.parse(await fs.readFile(path.join(meta, 'material/data.json'), 'utf8'))
const materials = new Map()
for (const family of Object.values(families)) {
  const key = family.type === 'talent' ? family.name.match(/「(.+)」/)?.[1] : family.name.slice(0, 4)
  let days = []
  let city = ''
  for (const [week, names] of Object.entries(daily[family.type] ?? {})) {
    const index = names.indexOf(key)
    if (index >= 0) {
      days = [Number(week), Number(week) + 3, 7]
      city = cities[index] ?? ''
    }
  }
  for (const item of [family, ...Object.values(family.items ?? {})]) {
    materials.set(item.name, { id: String(item.id ?? item.name), name: item.name, kind: item.type, rarity: item.star, family: family.name, days, city, sources: [], description: '' })
  }
}

const pools = await dataModule(path.join(meta, 'info/pool.js'),
  '[...poolDetail.map(p=>({...p,kind:"event"})),...mixPoolDetail.map(p=>({...p,kind:"chronicled"}))]')

const birthdays = []
for (const name of await fs.readdir(path.join(meta, 'character'))) {
  let raw
  try {
    raw = JSON.parse(await fs.readFile(path.join(meta, 'character', name, 'data.json'), 'utf8'))
  } catch (error) {
    if (error.code === 'ENOENT' || error.code === 'ENOTDIR') {
      continue
    }
    throw error
  }
  if (raw.birth && /^\d{1,2}-\d{1,2}$/.test(raw.birth)) {
    birthdays.push({ id: String(raw.id), name: raw.name, date: raw.birth.split('-').map(part => part.padStart(2, '0')).join('-') })
  }
}

const out = {
  version: 'miao-7f6f1c84c891',
  materials: [...materials.values()].sort((a, b) => a.name.localeCompare(b.name, 'zh')),
  pools: pools.map(p => ({ version: p.version, half: p.half, from: p.from, to: p.to, kind: p.kind, characters5: p.char5 ?? [], characters4: p.char4 ?? [], weapons5: p.weapon5 ?? [], weapons4: p.weapon4 ?? [] })),
  birthdays,
}
await fs.writeFile(path.join(root, 'internal/assets/data/resources.json'), JSON.stringify(out) + '\n')
console.log(JSON.stringify({ materials: materials.size, pools: pools.length, birthdays: birthdays.length }))
