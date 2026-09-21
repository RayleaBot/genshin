const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['希诺宁'] };
    if (attr.cpct * 2 + attr.cdmg >= 240) {
        title.push('战斗');
        particularAttr.cpct = 100;
        particularAttr.cdmg = 100;
        particularAttr.recharge = 55;
        particularAttr.heal = 70;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['希诺宁']);
}

return exports;})();
