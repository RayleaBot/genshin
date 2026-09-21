const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.mainAttr = exports.defDmgIdx = exports.details = void 0;
exports.details = [{
        title: 'E伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['技能伤害'], 'e')
    },
    {
        title: 'Q释放伤害',
        dmg: ({ talent }, dmg) => dmg(talent.q['禁限区域生成伤害'], 'q')
    },
    {
        title: 'Q领域进入伤害',
        dmg: ({ talent }, dmg) => dmg(talent.q['禁限区域踏入伤害'], 'q')
    },
    {
        title: '离场引爆领域伤害',
        dmg: ({}, dmg) => dmg(200, 'q')
    }
];
exports.defDmgIdx = 1;
exports.mainAttr = 'atk,dmg,cpct,cdmg,mastery';

return exports;})();
