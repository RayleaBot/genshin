// Compares the packaged calculator with miao-plugin's own ProfileDmg on
// synthetic profiles and records the results as regression vectors. No
// account, application or external API is used.
//
// Usage: node scripts/generate-reference-build-vectors.mjs <参考项目/2026-09-15>
// Writes internal/assets/testdata/calc-vectors.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const refs = path.resolve(process.argv[2])
const ts = createRequire(path.join(root, 'ui/package.json'))('typescript')
const calc = path.join(root, 'internal/assets/calc')
const reference = path.join(root, 'internal/reference')
const catalog = JSON.parse(await fs.readFile(path.join(calc, 'catalog.json'), 'utf8'))

const cache = new Map()
async function script(file) {
  if (!cache.has(file)) {
    cache.set(file, new vm.Script(await fs.readFile(file, 'utf8'), { filename: file }))
  }
  return cache.get(file)
}

// original transpiles an upstream module to CommonJS without its imports.
async function original(file) {
  const text = (await fs.readFile(path.join(refs, 'miao-plugin', file), 'utf8')).replace(/^import .*$/gm, '')
  return ts.transpileModule(text, { compilerOptions: { target: ts.ScriptTarget.ES2019, module: ts.ModuleKind.CommonJS } }).outputText
}
const originalProfile = await original('models/ProfileDmg.js')
const originalBuffs = await original('models/dmg/DmgBuffs.js')

// syntheticInput builds a level 90 profile with the given constellation, a
// 角斗士的终幕礼 set and the character's highest talent levels up to 10.
const syntheticInput = `(() => {
  currentGame = inputRecord.game
  weaponCatalog = inputWeapons
  const char = characterObject(inputRecord)
  const weapon = Weapon.get(weaponID)
  const talents = {}
  for (const [key, item] of Object.entries(char.detail.talent || {})) {
    if (!['a', 'e', 'q', 't', 'me', 'mt', 'xe'].includes(key)) continue
    const values = Object.values(item.tables || {})[0]?.values
    talents[key] = Math.min(10, values?.length || 10)
  }
  const gear = Array.from({ length: 5 }, (_, i) => ({
    set_name: '角斗士的终幕礼',
    main: { key: i === 0 ? 'hpPlus' : i === 1 ? 'atkPlus' : 'atk', value: i === 0 ? 4000 : i === 1 ? 300 : 40 },
    sub: [{ key: 'cpct', value: 10 }, { key: 'cdmg', value: 15 }, { key: 'mastery', value: 20 }],
  }))
  const profile = {
    game: currentGame, char, level: 90, promote: 6, cons: rank, talent: talents,
    trees: Object.keys(char.detail.tree || {}),
    weapon: { id: weapon.id, name: weapon.name, level: 90, promote: 6, affix: 1, refinement: 1 },
    artis: artifactObject(gear),
  }
  const attributes = {}
  for (const [key, value] of Object.entries(new Attr(profile).calc())) {
    if (typeof value === 'number' && Number.isFinite(value)) attributes[key] = value
  }
  return { level: 90, promote: 6, rank, talents, trees: profile.trees, weapon: profile.weapon, equipment: gear, attributes, enemy_level: 103 }
})()`

// The upstream class pipeline runs apart from runner.js as the oracle for buff
// order, talent and condition checks, and damage.
const upstreamClasses = `
  const DmgBuffs = (() => { const exports = {}; ${originalBuffs}; return exports.default })()
  const ProfileDmg = (() => { const exports = {}; ${originalProfile}; return exports.default })()
  ArtifactSet.eachSet = (sets, fn) => {
    for (const [name, n] of Object.entries(sets)) {
      if (n >= 4) fn({ name }, 2)
      fn({ name }, n)
    }
  }`
const upstreamResult = `(() => {
  const char = characterObject(inputRecord)
  const profile = { id: char.id, game: currentGame, char, level: input.level, cons: input.rank, talent: input.talents, trees: input.trees, weapon: input.weapon, artis: artifactObject(input.equipment) }
  profile.promote = input.promote
  profile.attr = new Attr(profile).calc()
  const calculator = Object.create(ProfileDmg.prototype)
  calculator.game = currentGame
  calculator.profile = profile
  calculator.char = char
  calculator.getCalcRule = async () => characterRule
  return calculator.calcData({ enemyLv: input.enemy_level })
})()`

// compare fails when the packaged results differ from upstream's.
function compare(actual, expected) {
  if (actual.baseline.results.length !== expected.ret.length) {
    throw Error('oracle result count')
  }
  expected.ret.forEach((b, i) => {
    const a = actual.baseline.results[i]
    if (b.type === 'text') {
      if (a.text !== String(b.avg)) throw Error('oracle text difference')
      return
    }
    for (const [ak, bk] of [['expected', 'avg'], ['critical', 'dmg']]) {
      if (a[ak] == null && b[bk] == null) continue
      if (!Number.isFinite(b[bk]) || Math.abs(a[ak] - b[bk]) > 1e-8 * Math.max(1, Math.abs(b[bk]))) {
        throw Error(`oracle difference ${i} ${ak} ${a[ak]} ${b[bk]}`)
      }
    }
  })
}

const vectors = []
const failures = []
for (const record of catalog.characters.filter(record => record.script)) {
  const choices = catalog.weapons.filter(w => w.type === record.weapon_type)
  if (!choices.length) {
    failures.push({ key: record.key, reason: 'no weapon' })
    continue
  }
  const weaponID = choices.find(w => w.data.star === 5)?.id || choices[0].id
  for (const rank of [0, 6]) {
    const context = vm.createContext({ inputRecord: record, inputWeapons: catalog.weapons, rank, weaponID })
    const files = [
      path.join(reference, 'vendor/lodash.js'),
      path.join(reference, 'miao/bootstrap.js'),
      path.join(reference, 'miao/common.js'),
      path.join(calc, 'game.js'),
      path.join(calc, record.script),
      path.join(reference, 'miao/runner.js'),
    ]
    for (const file of files) {
      (await script(file)).runInContext(context, { timeout: 1000 })
    }
    try {
      const input = vm.runInContext(syntheticInput, context, { timeout: 1000 })
      context.input = input
      const actual = vm.runInContext('runBuild(inputRecord, inputWeapons, input)', context, { timeout: 1500 })
      vm.runInContext(upstreamClasses, context, { timeout: 1000 })
      const expected = await vm.runInContext(upstreamResult, context, { timeout: 1500 })
      compare(actual, expected)
      const results = expected.ret.map(r => ({ expected: r.type === 'text' ? null : r.avg, text: r.type === 'text' ? String(r.avg) : '', critical: r.dmg ?? null }))
      vectors.push({ key: record.key, input, results })
    } catch (error) {
      failures.push({ key: record.key, rank, reason: error.message })
    }
  }
}
await fs.writeFile(path.join(root, 'internal/assets/testdata/calc-vectors.json'), JSON.stringify(vectors) + '\n')
console.log(JSON.stringify({ vectors: vectors.length, characters: new Set(vectors.map(v => v.key)).size, failures: failures.length, sample: failures.slice(0, 12) }))
