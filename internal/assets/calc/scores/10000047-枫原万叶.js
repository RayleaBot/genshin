const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, rule, def }) {
    if (cons === 6) {
        return rule('万叶-满命', { atk: 75, cpct: 100, cdmg: 100, mastery: 100, dmg: 100, recharge: 55 });
    }
    return def(usefulAttr['枫原万叶']);
}

return exports;})();
