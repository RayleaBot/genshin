const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ weapon, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['莫娜'] };
    if (weapon.name === '西风秘典') {
        title.push('西风');
        particularAttr.atk = 0;
        particularAttr.mastery = 0;
        particularAttr.cpct = 100;
        particularAttr.cdmg = 0;
        particularAttr.recharge = 100;
        particularAttr.dmg = 0;
    }
    if (weapon.name === '讨龙英杰谭') {
        title.push('讨龙');
        particularAttr.atk = 0;
        particularAttr.mastery = 0;
        particularAttr.cpct = 0;
        particularAttr.cdmg = 0;
        particularAttr.recharge = 100;
        particularAttr.dmg = 0;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['莫娜']);
}

return exports;})();
