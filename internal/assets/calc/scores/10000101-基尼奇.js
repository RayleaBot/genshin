const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['基尼奇'] };
    if (cons >= 1) {
        title.push('高命');
        particularAttr.atk = 100;
        if (cons >= 4) {
            particularAttr.recharge = 35;
        }
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['基尼奇']);
}

return exports;})();
