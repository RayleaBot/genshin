const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.mainAttr = exports.details = void 0;
exports.details = [{
        title: 'E荒星伤害',
        dmg: ({ talent }, dmg) => dmg(talent.e['技能伤害'], 'e')
    }, {
        title: 'Q地震波单次伤害',
        dmg: ({ talent }, dmg) => dmg(talent.q['地震波单次伤害'], 'q')
    }];
exports.mainAttr = 'atk,cpct,cdmg,dmg';
exports.buffs = [{
        title: '岩主1命：处于Q岩造物范围内时，暴击率提高[cpct]%',
        cons: 1,
        data: {
            cpct: 10
        }
    }];
exports.createdBy = 'Aluxes';

return exports;})();
