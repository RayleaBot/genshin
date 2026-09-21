const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['玛拉妮'] };
    if (cons >= 4) {
        title.push('高命');
        particularAttr.recharge = 30;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['玛拉妮']);
}

return exports;})();
