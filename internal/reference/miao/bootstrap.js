// The parts of miao-plugin's runtime the pinned calculation modules use:
// numbers and formatting only. Logging is silenced so combat values never
// reach the host log.
const lodash = _;
let Format;
const Data = { eachStr: (text, fn) => String(text).split(',').forEach(fn) };
const Cfg = { get: (_key, defaultValue) => defaultValue };

// Base answers unknown properties from _get or meta, as upstream's Base does.
class Base {
  constructor() {
    this.game = 'gs';
    return new Proxy(this, {
      get(target, key, receiver) {
        if (key in target) return Reflect.get(target, key, receiver);
        if (target._get) return target._get.call(receiver, key);
        return target.meta?.[key];
      },
    });
  }
  get isGs() { return true; }
  get isSr() { return false; }
}

// Upstream getMeta(game, type, key) returns one entry when key is given.
const Meta = {
  getMeta(_game, kind, key) {
    const meta = kind === 'weapon' ? { weaponBuffs } : artiMeta;
    return key ? meta[key] : meta;
  },
};
const console = { log() {} };
const logger = { info() {}, warn() {}, error() {}, debug() {} };

// currentGame is the game upstream's modules are called with; weaponCatalog
// holds the weapons of the running calculation.
let currentGame, weaponCatalog;

const ArtifactSet = {
  getArtisSetBuff(name, count) {
    const data = artifactBuffs[name]?.[count] || artifactBuffs[name + count];
    if (!data) return [];
    return Array.isArray(data) ? data : [data];
  },
};

// loadWeaponDefinitions calls a weapon family's calc.js with upstream's step
// and attr helpers; plain definition objects are returned as they are.
function loadWeaponDefinitions(definitions) {
  if (typeof definitions !== 'function') return definitions;
  const step = (start, increase = 0) => Array.from({ length: 6 }, (_, i) => start + (increase || start / 4) * i);
  const attr = (key, start, increase) => ({ title: key + '提高[key]', isStatic: true, refine: { [key]: step(start, increase) } });
  return definitions(step, attr);
}

// Weapon ascension levels; attr values are listed at each level, and at
// level+ for the ascended value at the same level.
const weaponSteps = [1, 20, 40, 50, 60, 70, 80, 90];

class ReferenceWeapon {
  constructor(record) {
    this.name = record.name;
    this.type = record.type;
    this.id = record.id;
    this.detail = record.data;
    this.game = currentGame;
  }

  // calcAttr interpolates base attack and the bonus stat, the bonus in steps
  // of five levels, as upstream's Weapon.calcAttr.
  calcAttr(level, promote) {
    const attr = this.detail.attr;
    const left = weaponSteps[promote];
    const right = weaponSteps[promote + 1];
    if (left === undefined || right === undefined || level < left || level > right) throw Error('weapon.promote');
    const a = attr.atk[left + '+'] ?? attr.atk[left];
    const b = attr.atk[right];
    const x = attr.bonusData[left + '+'] ?? attr.bonusData[left];
    const y = attr.bonusData[right];
    const n = Math.ceil((right - left) / 5);
    return {
      atkBase: a + (b - a) * (level - left) / (right - left),
      attr: { key: attr.bonusKey, value: x + (n - Math.ceil((right - level) / 5)) * (y - x) / n },
    };
  }

  // getWeaponAffixBuffs returns the permanent or the combat effects at a
  // refinement, as upstream's Weapon.getWeaponAffixBuffs.
  getWeaponAffixBuffs(affix, isStatic = true) {
    let definitions = lodash.cloneDeep(weaponBuffs[this.id] || weaponBuffs[this.name] || []);
    if (!Array.isArray(definitions)) definitions = [definitions];
    const tables = Object.fromEntries(Object.entries(this.detail.skill?.tables || {}).map(([k, v]) => [k, v[affix - 1]]));
    const result = [];
    for (let data of definitions) {
      if (typeof data === 'function') data = data(tables);
      if (!data || !!data.isStatic !== !!isStatic) continue;
      if (isStatic) {
        const values = {};
        if (data.idx && data.key) values[data.key] = tables[data.idx];
        for (const [k, v] of Object.entries(data.refine || {})) values[k] = v[affix - 1] * (data.buffCount || 1);
        if (Object.keys(values).length) result.push({ isStatic: true, data: values });
        continue;
      }
      data.title = /：/.test(data.title) ? data.title : this.name + '：' + (data.title || '额外效果');
      data.data = data.data || {};
      if (data.idx && data.key) {
        data.data[data.key] = tables[data.idx];
      } else {
        for (const [k, v] of Object.entries(data.refine || {})) data.data[k] = ({ refine }) => v[refine] * (data.buffCount || 1);
      }
      result.push(data);
    }
    return result;
  }
}

const Weapon = {
  get(name) {
    const record = weaponCatalog.find(w => w.name === name || w.id === String(name));
    return record ? new ReferenceWeapon(record) : false;
  },
};
