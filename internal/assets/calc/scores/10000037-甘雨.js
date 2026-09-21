const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ artis, rule, def }) {
    if (artis.is('冰套4')) {
        return rule('甘雨-永冻', { atk: 75, cpct: 100, cdmg: 100, dmg: 100, recharge: 55 });
    }
    return def({ atk: 75, cpct: 100, cdmg: 100, mastery: 75, dmg: 100 });
}

return exports;})();
