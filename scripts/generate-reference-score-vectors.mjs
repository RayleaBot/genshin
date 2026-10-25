// Regenerate scoring vectors from the pinned upstream rules and check the
// packaged scorer against them. The synthetic panel contexts stay unchanged.
// Usage: node scripts/generate-reference-score-vectors.mjs <references>
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const refs = path.resolve(process.argv[2])
const ts = createRequire(path.join(root, 'ui/package.json'))('typescript')
const calc = path.join(root, 'internal/assets/calc')
const fixturePath = path.join(root, 'internal/assets/testdata/score-vectors.json')
const fixture = JSON.parse(await fs.readFile(fixturePath, 'utf8'))
const catalog = JSON.parse(await fs.readFile(path.join(calc, 'catalog.json'), 'utf8'))
const scripts = new Map()
async function execute(context, file) {
  if (!scripts.has(file)) scripts.set(file, new vm.Script(await fs.readFile(file, 'utf8'), { filename: file }))
  scripts.get(file).runInContext(context, { timeout: 1000 })
}
async function original(file) {
  const source = (await fs.readFile(path.join(refs, file), 'utf8')).replace(/^import .*$/gm, '')
  const compiled = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2019, module: ts.ModuleKind.CommonJS } }).outputText
  return '(() => { const exports = {}; ' + compiled + '; return exports.default })()'
}
const equal = (a, b) => Number.isFinite(a) && Number.isFinite(b) && Math.abs(a - b) <= 1e-8 * Math.max(1, Math.abs(b))

const oracle = 'const OriginalMark=' + await original('miao-plugin/models/artis/ArtisMark.js') +
  ';const OriginalCfg=((ArtisMark)=>' + await original('miao-plugin/models/artis/ArtisMarkCfg.js') + ')(OriginalMark);'
const vectors = []
for (const record of catalog.characters) {
  for (const [index, template] of fixture.contexts.entries()) {
    const input = structuredClone(template)
    const main = index === 0 ? record.element : undefined
    input.equipment = input.equipment.filter(g => {
      if (g.main.key !== '$dmg') return true
      if (!main) return false
      g.main.key = main
      return true
    })
    const context = vm.createContext({ record, weapons: catalog.weapons, input })
    for (const name of ['vendor/lodash.js', 'miao/bootstrap.js', 'miao/common.js']) await execute(context, path.join(root, 'internal/reference', name))
    await execute(context, path.join(calc, 'game.js'))
    if (record.score_script) await execute(context, path.join(calc, record.score_script))
    await execute(context, path.join(root, 'internal/reference/miao/runner.js'))
    const actual = vm.runInContext('runScore(record, weapons, input)', context, { timeout: 1000 })
    vm.runInContext(oracle, context, { timeout: 1000 })
    const expected = vm.runInContext(`(() => {
      const char = {name:record.name, abbr:record.data.abbr||record.name, game:'gs', isGs:true, baseAttr:record.data.baseAttr, getArtisCfg:()=>typeof scoreRule==='undefined'?false:scoreRule.default};
      const weapon = input.weapon.id ? Weapon.get(input.weapon.id) : false;
      const artis = scoringArtifacts(input.equipment);
      const profile = {game:'gs',char,id:Number(record.id),elem:record.element,attr:input.attributes,cons:input.rank,artis,
        weapon:{name:weapon?.name||input.weapon.name||'',affix:input.weapon.refinement,bonusKey:weapon?.detail?.attr?.bonusKey}};
      const cfg = OriginalCfg.getCfg(profile);
      return input.equipment.map(g=>({slot:g.slot,score:OriginalMark.getMark({charCfg:cfg,idx:g.slot,arti:artis.pieces[g.slot],elem:profile.elem,game:'gs',id:profile.id})}));
    })()`, context, { timeout: 1000 })
    const total = expected.reduce((sum, p) => sum + p.score, 0)
    if (!equal(actual.total, total) || expected.some((p, i) => !equal(actual.pieces[i].score, p.score) || p.score > 70)) throw Error('Upstream score mismatch: ' + record.key + '/' + index)
    vectors.push({ key: record.key, context: index, ...(main ? { main } : {}), total, pieces: Object.fromEntries(expected.map(p => [String(p.slot), p.score])) })
  }
}
await fs.writeFile(fixturePath, JSON.stringify({ contexts: fixture.contexts, vectors }) + '\n')
console.log(JSON.stringify({ characters: catalog.characters.length, vectors: vectors.length, oracle: 'miao ArtisMark and ArtisMarkCfg' }))
