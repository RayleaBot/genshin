// Bundle the pinned miao-plugin calculation modules for Genshin Impact. Only
// numerical modules are transpiled; nothing here starts Yunzai, reads
// configuration, fetches resources or loads user code.
//
// Usage: node scripts/bundle-reference-calculation.mjs <参考项目/2026-09-15>
//
// Writes the miao runtime to internal/reference/miao/common.js and the calc
// directory internal/assets/calc: catalog.json, game.js (weapon and set
// effects plus the artifact scoring tables), characters/<id>-<name>.js from
// calc.js and scores/<id>-<name>.js from artis.js. Characters with data but no
// calc.js enter the catalog without a damage script so they can still be
// scored. The artifact set names are also written to
// internal/assets/catalog.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { calcScriptName, transpile, walk } from './calc-bundle.mjs'

const references = process.argv[2] && path.resolve(process.argv[2])
if (!references) throw new Error('Usage: bundle-reference-calculation.mjs <references>')
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const typescript = createRequire(path.join(root, 'ui/package.json'))('typescript')
const miao = path.join(references, 'miao-plugin')
const game = 'gs'

const allowedImports = ['lodash', '#miao', './DmgCalcMeta.js', './DmgMastery.js', './AttrItem.js', '../Base.js', '../index.js', '#miao.models', './AttrData.js', './common/Elem.js', './ArtisMark.js', './ArtisMarkCfg.js', '../../artifact/artis-mark.js']

async function module(file, provided) {
  const source = provided ?? await fs.readFile(path.join(miao, file), 'utf8')
  return transpile(typescript, file, source, allowedImports)
}

let common = '// Fixed miao-plugin 7f6f1c84; MIT, see LICENSES. Generated numeric modules.\n'
common += `const Elem=${await module('components/common/Elem.js')}.default;\nFormat=${await module('components/Format.js')}.default;\n`
for (const [name, file] of [['AttrItem', 'dmg/AttrItem'], ['DmgMastery', 'dmg/DmgMastery'], ['DmgAttr', 'dmg/DmgAttr'], ['DmgCalc', 'dmg/DmgCalc'], ['AttrData', 'attr/AttrData'], ['Attr', 'attr/Attr']]) {
  if (name === 'DmgMastery') common += `const {erType,erTitle,eleBaseDmg,breakBaseDmg,cryBaseDmg,elationBaseDmg}=${await module('models/dmg/DmgCalcMeta.js')};\n`
  common += `const ${name}=${await module('models/' + file + '.js')}.default;\n`
}
for (const name of ['ArtisMarkCfg', 'ArtisMark']) common += `const ${name}=${await module('models/artis/' + name + '.js')}.default;\n`
await fs.writeFile(path.join(root, 'internal/reference/miao/common.js'), common)

const resources = path.join(miao, 'resources/meta-' + game)
const calc = path.join(root, 'internal/assets/calc')
for (const directory of ['characters', 'scores']) {
  await fs.rm(path.join(calc, directory), { recursive: true, force: true })
  await fs.mkdir(path.join(calc, directory), { recursive: true })
}
const catalog = { version: 'miao-7f6f1c84-reference-v1', characters: [], weapons: [], coverage: [] }

const sets = JSON.parse(await fs.readFile(path.join(resources, 'artifact/data.json'), 'utf8'))
const setNames = {}
// Genshin showcase answers name a piece only by ID, and 面板换装 swaps a
// piece's set: the set and slot give the piece name.
const pieces = {}
for (const set of Object.values(sets)) {
  for (const [slot, part] of Object.entries(set.idxs || {})) {
    if (part.id) setNames[String(part.id)] = set.name
    if (part.name) setNames[part.name] = set.name
    if (part.name) (pieces[set.name] ||= [])[slot - 1] = part.name
  }
}
const pluginCatalog = path.join(root, 'internal/assets/catalog.json')
const local = JSON.parse(await fs.readFile(pluginCatalog, 'utf8'))
local.artifact_sets = setNames
local.artifact_pieces = pieces
await fs.writeFile(pluginCatalog, JSON.stringify(local, null, 2) + '\n')

let gameSource = `const artifactBuffs=${await module('resources/meta-' + game + '/artifact/calc.js')}.default;\nconst weaponBuffs={};\n`
// The tables Meta.getMeta(game, 'arti') serves to the scoring modules.
const artifact = 'resources/meta-' + game + '/artifact/'
gameSource += `const artiMeta=(()=>{const attrs=${await module(artifact + 'extra.js')};const alias=${await module(artifact + 'alias.js')};const mark=${await module(artifact + 'artis-mark.js')};return {mainAttr:attrs.mainAttr,subAttr:attrs.subAttr,attrMap:attrs.attrMap,usefulAttr:mark.usefulAttr,setAbbr:alias.setAbbr}})();\nconst usefulAttr=artiMeta.usefulAttr;\n`
for (const file of (await walk(path.join(resources, 'weapon'))).filter(f => path.basename(f) === 'calc.js')) {
  gameSource += `Object.assign(weaponBuffs,loadWeaponDefinitions(${await module(path.relative(miao, file).replaceAll('\\', '/'))}.default));\n`
}
await fs.writeFile(path.join(calc, 'game.js'), gameSource)

// Upstream keeps one artis.js per character directory; every Traveler
// element reads the one in 旅行者/, so the lookup walks up from the element
// directory and names the script after the directory that holds it.
const scoreScripts = new Set()
const scoreScript = async directory => {
  let file
  for (; directory.startsWith(path.join(resources, 'character', path.sep)); directory = path.dirname(directory)) {
    try {
      await fs.access(path.join(directory, 'artis.js'))
      file = path.join(directory, 'artis.js')
      break
    } catch {}
  }
  if (!file) return undefined
  const owner = JSON.parse(await fs.readFile(path.join(directory, 'data.json'), 'utf8'))
  const script = calcScriptName(game + '_' + owner.id, owner.name).replace(/^characters\//, 'scores/')
  if (!scoreScripts.has(script)) {
    scoreScripts.add(script)
    await fs.writeFile(path.join(calc, script), `const scoreRule=${await module(path.relative(miao, file).replaceAll('\\', '/'))};\n`)
  }
  return script
}
// Upstream names characters through its index, which a per-character
// data.json can disagree with. Travelers keep their own names.
const index = JSON.parse(await fs.readFile(path.join(resources, 'character/data.json'), 'utf8'))
const addCharacter = async (key, id, data, element, source, code, authored, score) => {
  if (catalog.characters.some(c => c.key === key)) throw new Error('Duplicate reference key ' + key)
  if (!/^[a-z0-9_]+$/i.test(key)) throw new Error('Invalid reference key ' + key)
  const name = [7, 10000005, 10000007].includes(Number(id)) ? data.name : (index[id]?.name || data.name)
  let script = ''
  if (code) {
    script = calcScriptName(key, name)
    await fs.writeFile(path.join(calc, script), `const characterRule=${code};\n`)
    catalog.coverage.push(authored ? { key, source, authored: true } : { key, source })
  }
  catalog.characters.push({ key, game, id: String(id), name, element, weapon_type: data.weapon, script, score_script: score, data })
}
const calculated = new Set()
for (const file of (await walk(path.join(resources, 'character'))).filter(f => path.basename(f) === 'calc.js')) {
  let parent = path.dirname(file)
  let data
  while (parent.startsWith(path.join(resources, 'character'))) {
    try {
      data = JSON.parse(await fs.readFile(path.join(parent, 'data.json'), 'utf8'))
      break
    } catch (error) {
      if (error.code !== 'ENOENT') throw error
      parent = path.dirname(parent)
    }
  }
  if (!data?.id) throw new Error('Missing character metadata: ' + file)
  // Every Traveler element shares id 7; each element keeps its own rule.
  const variant = data.id === 7 ? data.elem : (path.dirname(file) !== parent ? path.basename(path.dirname(file)) : '')
  const id = data.id === 7 ? 10000007 : data.id
  const source = path.relative(miao, file).replaceAll('\\', '/')
  const key = game + '_' + id + (variant ? '_' + variant : '')
  await addCharacter(key, id, data, variant || data.elem, source, await module(source), false, await scoreScript(path.dirname(file)))
  calculated.add(parent)
}
for (const file of (await walk(path.join(resources, 'weapon'))).filter(f => path.basename(f) === 'data.json')) {
  const data = JSON.parse(await fs.readFile(file, 'utf8'))
  if (!data.id || !data.name || !data.attr) continue
  const type = path.relative(path.join(resources, 'weapon'), file).split(path.sep)[0]
  catalog.weapons.push({ game, id: String(data.id), name: data.name, type, data })
}
// Characters upstream can score but not calculate yet. The Traveler root and
// the placeholders 空 and 荧 are covered by the element variants above.
for (const file of (await walk(path.join(resources, 'character'))).filter(f => path.basename(f) === 'data.json')) {
  const directory = path.dirname(file)
  if (calculated.has(directory) || path.dirname(directory) !== path.join(resources, 'character')) continue
  if ([...calculated].some(d => d.startsWith(directory + path.sep))) continue
  const data = JSON.parse(await fs.readFile(file, 'utf8'))
  if (!data.id || !data.name || [10000005, 10000007].includes(data.id)) continue
  const key = game + '_' + data.id
  await addCharacter(key, data.id, data, data.elem, '', '', false, await scoreScript(directory))
}
await fs.writeFile(path.join(calc, 'catalog.json'), JSON.stringify(catalog) + '\n')
console.log(JSON.stringify({ characters: catalog.characters.length, weapons: catalog.weapons.length, version: catalog.version }))
