const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, artis, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['云堇'] };
    if (artis.is('cpct,cdmg,def', 5) && (attr.cpct * 2 + attr.cdmg >= 240)) {
        title.push('输出');
        particularAttr.atk = 75;
        particularAttr.def = 100;
        particularAttr.cpct = 100;
        particularAttr.cdmg = 100;
        particularAttr.dmg = 100;
        particularAttr.recharge = 75;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['云堇']);
}

return exports;})();
