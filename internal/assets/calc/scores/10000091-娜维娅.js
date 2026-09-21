const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['娜维娅'] };
    let recharge = particularAttr.recharge;
    if (cons >= 1) {
        title.push('高命');
        recharge -= 10;
        if (cons == 6) {
            particularAttr.atk = 100;
        }
    }
    if (title.length > 0) {
        particularAttr.recharge = recharge;
        return def(particularAttr, title);
    }
    return def(usefulAttr['娜维娅']);
}

return exports;})();
