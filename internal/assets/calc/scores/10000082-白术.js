const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['白术'] };
    if (cons === 6) {
        title.push('满命');
        particularAttr.atk = 75;
        particularAttr.cpct = 100;
        particularAttr.cdmg = 100;
        particularAttr.dmg = 100;
        particularAttr.recharge = 35;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['白术']);
}

return exports;})();
