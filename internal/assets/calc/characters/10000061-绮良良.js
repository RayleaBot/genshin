const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = exports.mainAttr = exports.details = void 0;
exports.details = [{
        title: '秘法·惊喜特派伤害',
        talent: 'q',
        dmg: ({ talent }, dmg) => dmg(talent.q['技能伤害'], 'q')
    }, {
        title: '护盾最大吸收量',
        talent: 'e',
        dmg: ({ talent, calc, attr }, { shield }) => shield(talent.e['护盾吸收量上限2'][0] * calc(attr.hp) / 100 + talent.e['护盾吸收量上限2'][1] * 1)
    }];
exports.mainAttr = 'atk,hp,cpct,cdmg';
exports.buffs = [{
        title: '绮良良被动：基于绮良良的生命上限，秘法·惊喜特派伤害提升[qDmg]%',
        sort: 9,
        data: {
            qDmg: ({ attr, calc }) => calc(attr.hp) / 1000 * 0.3
        }
    }];

return exports;})();
