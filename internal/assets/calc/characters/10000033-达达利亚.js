const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = exports.mainAttr = exports.details = void 0;
exports.details = [{
        title: '开E后重击',
        dmg: ({ talent }, dmg) => dmg(talent.e['重击伤害'], 'a2')
    }, {
        title: '断流·斩 伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['断流·斩 伤害'], 'e')
    }, {
        title: '开E后Q伤害',
        dmg: ({ talent }, dmg) => dmg(talent.q['技能伤害·近战'], 'q')
    }, {
        title: '开E后Q蒸发',
        dmg: ({ talent }, dmg) => dmg(talent.q['技能伤害·近战'], 'q', 'vaporize')
    }];
exports.mainAttr = 'atk,cpct,cdmg,mastery';
exports.buffs = ['vaporize'];

return exports;})();
