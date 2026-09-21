const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['瓦雷莎'] };
    if (cons == 6) {
        title.push('满命');
        particularAttr.recharge = 0;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['瓦雷莎']);
}

return exports;})();
