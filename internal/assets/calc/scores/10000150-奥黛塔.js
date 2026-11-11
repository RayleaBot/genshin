const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['奥黛塔'] };
    if (cons >= 4) {
        title.push('高命');
        particularAttr.recharge = 50;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['奥黛塔']);
}

return exports;})();
