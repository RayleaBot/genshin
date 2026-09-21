const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['绮良良'] };
    if (attr.cpct * 2 + attr.cdmg >= 240) {
        title.push('战斗');
        particularAttr.hp = 50;
        particularAttr.atk = 75;
        particularAttr.cpct = 100;
        particularAttr.cdmg = 100;
        particularAttr.dmg = 100;
        particularAttr.recharge = 30;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['绮良良']);
}

return exports;})();
