const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = exports.mainAttr = exports.defDmgKey = exports.details = void 0;
exports.details = [{
        title: '重击伤害',
        dmg: ({ talent }, dmg) => dmg(talent.a['重击伤害'], 'a2')
    }, {
        title: '重击蒸发',
        dmg: ({ talent }, dmg) => dmg(talent.a['重击伤害'], 'a2', 'vaporize')
    }, {
        title: 'E每跳治疗',
        dmg: ({ talent, attr, calc }, { heal }) => heal(talent.e['持续治疗量2'][0] * calc(attr.hp) / 100 + talent.e['持续治疗量2'][1] * 1)
    }, {
        title: 'Q治疗量',
        dmgKey: 'qHeal',
        dmg: ({ talent, attr, calc }, { heal }) => heal(talent.q['治疗量2'][0] * calc(attr.hp) / 100 + talent.q['治疗量2'][1] * 1)
    }];
exports.defDmgKey = 'qHeal';
exports.mainAttr = 'atk,hp,cpct,cdmg,mastery';
exports.buffs = [{
        title: '芭芭拉2命：开E水环持续期间获得15%水伤加成',
        cons: 2,
        data: {
            dmg: 15
        }
    }, 'vaporize'];

return exports;})();
