const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.mainAttr = exports.defDmgIdx = exports.details = void 0;
const getHunterAimStacks = ({ cons }) => cons >= 6 ? 2 : 1;
exports.details = [{
        title: '点按E伤害',
        params: { HunterAim: true },
        dmg: ({ talent }, dmg) => dmg(talent.e['点按伤害'], 'e')
    }, {
        title: 'Q每跳治疗量',
        params: { HunterAim: true },
        dmg: ({ attr, calc }, { heal }) => heal(calc(attr.atk) * 120 / 100)
    }, {
        title: 'E攻击力提升',
        params: { HunterAim: true },
        dmg: ({ talent, cons }) => ({
            avg: Format.percent(talent.e['猎者之准攻击力提升'] * getHunterAimStacks({ cons }) / 100),
            type: 'text'
        })
    }, {
        cons: 4,
        title: '4命追加每跳治疗',
        params: { HunterAim: true },
        dmg: ({ attr, calc }, { heal }) => heal(calc(attr.atk) * 60 / 100)
    }];
exports.defDmgIdx = 0;
exports.mainAttr = 'atk,recharge,cpct,cdmg,heal';
exports.buffs = [{
        check: ({ params }) => params.HunterAim === true,
        title: '猎者之准：当前场上角色攻击力提升[atkPct]%',
        data: {
            atkPct: ({ talent, cons }) => talent.e['猎者之准攻击力提升'] * getHunterAimStacks({ cons })
        }
    }, {
        title: '阿罗夏天赋：基于元素充能效率提升元素战技与元素爆发伤害[eDmg]%',
        sort: 4,
        data: {
            eDmg: ({ attr, calc }) => Math.min(calc(attr.recharge) * 0.35, 70),
            qDmg: ({ attr, calc }) => Math.min(calc(attr.recharge) * 0.35, 70)
        }
    }];
exports.createdBy = '墨痕';

return exports;})();
