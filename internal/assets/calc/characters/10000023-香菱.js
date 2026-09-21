const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = exports.mainAttr = exports.defDmgIdx = exports.details = void 0;
exports.details = [{
        title: '锅巴单口伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['喷火伤害'], 'e')
    }, {
        title: '锅巴单口蒸发',
        dmg: ({ talent }, dmg) => dmg(talent.e['喷火伤害'], 'e', 'vaporize')
    }, {
        title: '旋火轮单次伤害',
        params: { isQ: true },
        dmg: ({ talent }, dmg) => dmg(talent.q['旋火轮伤害'], 'q')
    }, {
        title: '旋火轮单次蒸发',
        params: { isQ: true },
        dmg: ({ talent }, dmg) => dmg(talent.q['旋火轮伤害'], 'q', 'vaporize')
    }];
exports.defDmgIdx = 3;
exports.mainAttr = 'atk,cpct,cdmg';
exports.buffs = [{
        cons: 1,
        title: '香菱1命：锅巴降低敌人火抗15',
        data: {
            kx: 15
        }
    }, {
        check: ({ params }) => !params.isQ,
        title: '香菱6命：旋火轮持续期间获得15%火伤加成',
        cons: 6,
        data: {
            dmg: 15
        }
    }, 'vaporize'];

return exports;})();
