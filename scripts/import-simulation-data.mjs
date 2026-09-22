// Converts the Miao-Yunzai 模拟抽卡 item pools. Neither the odds nor the
// banners are advertised as the current official rules. Python and PyYAML
// are needed only to regenerate.
//
// Usage: node scripts/import-simulation-data.mjs <参考项目/2026-09-15>
// Reads internal/assets/data/resources.json for the banners and writes
// internal/assets/data/simulation.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const refs = path.resolve(process.argv[2])
const dataDir = path.join(root, 'internal/assets/data')

const parsed = spawnSync('python', ['-X', 'utf8', '-c',
  'import yaml,json,sys;print(json.dumps(yaml.safe_load(open(sys.argv[1],encoding="utf-8")),ensure_ascii=True))',
  path.join(refs, 'Miao-Yunzai/plugins/genshin/defSet/gacha/gacha.yaml')], { encoding: 'utf8' })
if (parsed.status !== 0) {
  throw Error('PyYAML conversion failed')
}
const gacha = JSON.parse(parsed.stdout)

// Only event banners with every rarity filled can be drawn, newest first.
const resources = JSON.parse(await fs.readFile(path.join(dataDir, 'resources.json'), 'utf8'))
const banners = resources.pools
  .filter(p => p.kind === 'event' && p.characters5.length && p.weapons5.length && p.characters4.length && p.weapons4.length)
  .map(p => ({ ...p, id: [p.version, p.half, p.from].join('|') }))
  .reverse()

const deck = {
  version: 'entertainment-reference-20260915-v1',
  five_characters: gacha.role5,
  five_weapons: gacha.weapon5,
  four_characters: gacha.role4,
  four_weapons: gacha.weapon4,
  three_weapons: gacha.weapon3,
  banners,
}
await fs.writeFile(path.join(dataDir, 'simulation.json'), JSON.stringify(deck) + '\n')
await fs.copyFile(path.join(refs, 'Miao-Yunzai/LICENSE'), path.join(root, 'LICENSES/Miao-Yunzai-GPL-3.0.txt'))
const standard = deck.five_characters.length + deck.five_weapons.length + deck.four_characters.length + deck.four_weapons.length + deck.three_weapons.length
console.log(JSON.stringify({ banners: banners.length, standard_items: standard }))
