// Converts miao-plugin's artifact metadata into the table the cloud panel
// reader uses. Only the fixed metadata modules are read; no plugin runtime is
// loaded.
//
// Usage: node scripts/import-cloud-panel-data.mjs <参考项目/2026-09-15>
// Writes internal/assets/data/cloud-panels.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const artifactDir = path.join(path.resolve(process.argv[2]), 'miao-plugin/resources/meta-gs/artifact')

// moduleValue evaluates extra.js without its imports and returns expression.
async function moduleValue(file, expression) {
  const source = (await fs.readFile(path.join(artifactDir, file), 'utf8'))
    .replace(/^import .*$/gm, '')
    .replace(/^export\s+(?=const\s)/gm, '')
    .replace(/^export \{.*\}\s*$/gm, '')
  const context = vm.createContext({
    lodash: { forEach: (object, fn) => Object.entries(object).forEach(([key, value]) => fn(value, key)) },
    Format: { pct: String, comma: String },
  }, { codeGeneration: { strings: false, wasm: false } })
  return JSON.parse(new vm.Script(`${source}\nJSON.stringify(${expression})`).runInContext(context, { timeout: 1000 }))
}

const { attrMap, mainIdMap, attrIdMap } = await moduleValue('extra.js', '{attrMap,mainIdMap,attrIdMap}')
const attrs = Object.fromEntries(Object.entries(attrMap).map(([key, attr]) => [key, { title: attr.title, format: attr.format, value: attr.value }]))

const items = {}
const sets = JSON.parse(await fs.readFile(path.join(artifactDir, 'data.json'), 'utf8'))
for (const set of Object.values(sets)) {
  for (const [slot, item] of Object.entries(set.idxs)) {
    items[item.name] = { name: item.name, set: set.name, slot: Number(slot) }
  }
}

const out = { version: 'miao-7f6f1c84c891', attrMap: attrs, mainIdMap, attrIdMap, items }
await fs.writeFile(path.join(root, 'internal/assets/data/cloud-panels.json'), JSON.stringify(out) + '\n')
console.log(JSON.stringify({ items: Object.keys(items).length }))
