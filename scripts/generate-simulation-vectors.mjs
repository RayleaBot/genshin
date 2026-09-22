// Records the Miao-Yunzai 模拟抽卡 five-star probability for every pull count,
// by running its probability method on each case.
//
// Usage: node scripts/generate-simulation-vectors.mjs <参考项目/2026-09-15>
// Writes internal/app/testdata/simulation-probability-vectors.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const ts = createRequire(path.join(root, 'ui/package.json'))('typescript')
const file = 'Miao-Yunzai/plugins/genshin/model/gachaData.js'
const text = await fs.readFile(path.join(path.resolve(process.argv[2]), file), 'utf8')
const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.JS)

let body
function visit(node) {
  if (ts.isMethodDeclaration(node) && node.name.getText(source) === 'probability') {
    body = node.body.getText(source)
  }
  ts.forEachChild(node, visit)
}
visit(source)
if (!body) {
  throw Error('Missing reference probability')
}
const probability = new vm.Script(`(function()${body})`).runInNewContext(Object.create(null), { timeout: 1000 })

const vectors = []
const types = { character: 'role', weapon: 'weapon', standard: 'permanent' }
for (const kind of ['character', 'weapon', 'standard']) {
  for (let five = 0; five <= 95; five++) {
    for (const weekly of [0, 1, 2]) {
      const type = types[kind]
      const self = { type, def: { chance5: 60, chanceW5: 70 }, user: { [type]: { num5: five }, week: { num: weekly } }, gachaData: { [type]: { num5: five } } }
      vectors.push({ game: 'genshin', kind, five, weekly, expected: Math.min(10000, probability.call(self)) })
    }
  }
}
await fs.writeFile(path.join(root, 'internal/app/testdata/simulation-probability-vectors.json'), JSON.stringify(vectors))
console.log(JSON.stringify({ reference_probability_cases: vectors.length }))
