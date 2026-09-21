const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = exports.mainAttr = exports.details = void 0;
exports.details = [{
        title: 'Q施放治疗量',
        dmg: ({ talent, calc, attr }, { heal }) => heal(talent.q['施放治疗量2'][0] * calc(attr.hp) / 100 + talent.q['施放治疗量2'][1])
    }, {
        title: '鹰翎治疗量',
        dmg: ({ talent, calc, attr }, { heal }) => heal(talent.q['鹰翎治疗量2'][0] * calc(attr.hp) / 100 + talent.q['鹰翎治疗量2'][1])
    }];
exports.mainAttr = 'hp,cpct,cdmg';
exports.buffs = [];

return exports;})();
