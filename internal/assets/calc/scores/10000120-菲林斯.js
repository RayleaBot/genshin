const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ weapon, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['菲林斯'] };
    let recharge = particularAttr.recharge;
    if (weapon.name === '血染荒城') {
        title.push('专武');
        recharge -= 5;
    }
    if (title.length > 0) {
        particularAttr.recharge = recharge;
        return def(particularAttr, title);
    }
    return def(usefulAttr['菲林斯']);
}

return exports;})();
