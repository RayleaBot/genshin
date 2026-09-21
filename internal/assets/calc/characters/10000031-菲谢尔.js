const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = exports.mainAttr = exports.defParams = exports.details = void 0;
exports.details = [{
        title: '奥兹攻击激化伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['奥兹攻击伤害'], 'e', 'aggravate')
    }, {
        title: '奥兹攻击伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['奥兹攻击伤害'], 'e')
    }, {
        title: '奥兹召唤伤害',
        params: { e: true },
        dmg: ({ talent, cons }, dmg) => dmg(talent.e['召唤伤害'], 'e')
    }, {
        title: '至夜幻现伤害',
        dmg: ({ talent, cons }, dmg) => dmg(talent.q['落雷伤害'], 'q')
    }];
exports.defParams = { Hexenzirkel: true }; // 魔女会成员
exports.mainAttr = 'atk,cpct,cdmg,mastery';
exports.buffs = [{
        title: '皇女2命：施放夜巡影翼时，能额外造成200%攻击力的伤害',
        data: {
            ePct: ({ params }) => params.e ? 200 : 0
        }
    }, {
        title: '皇女4命：施放夜巡影翼时，能额外造成200%攻击力的伤害',
        data: {
            qPct: ({ params }) => params.q ? 222 : 0
        }
    }, 'aggravate'];

return exports;})();
