const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, rule, def }) {
    if (attr.mastery > 120) {
        return rule('行秋-蒸发', { atk: 75, cpct: 100, cdmg: 100, mastery: 75, dmg: 100, recharge: 75 });
    }
    return def({ atk: 75, cpct: 100, cdmg: 100, mastery: 0, dmg: 100, recharge: 75 });
}

return exports;})();
