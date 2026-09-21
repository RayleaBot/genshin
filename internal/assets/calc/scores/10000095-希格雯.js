const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['希格雯'] };
    if (cons === 6) {
        title.push('满命');
        particularAttr.dmg = 100;
        particularAttr.recharge = 100;
        particularAttr.heal = 90;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['希格雯']);
}

return exports;})();
