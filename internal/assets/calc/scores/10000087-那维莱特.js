const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ weapon, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['那维莱特'] };
    if (weapon.name === '万世流涌大典') {
        title.push('专武');
        particularAttr.recharge = 40;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['那维莱特']);
}

return exports;})();
