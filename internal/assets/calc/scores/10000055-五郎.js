const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, cons, artis, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['五郎'] };
    if (cons >= 4 && artis.is('heal', 5)) {
        title.push('治疗');
        particularAttr.heal = 100;
    }
    if (attr.cpct * 2 + attr.cdmg >= 240) {
        title.push('输出');
        particularAttr.atk = 75;
        particularAttr.def = 100;
        particularAttr.cpct = 100;
        particularAttr.cdmg = 100;
        particularAttr.dmg = 100;
        particularAttr.recharge = 55;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['五郎']);
}

return exports;})();
