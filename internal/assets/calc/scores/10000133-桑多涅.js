const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['桑多涅'] };
    if (cons >= 2) {
        title.push('高命');
        particularAttr.atk = 100;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['桑多涅']);
}

return exports;})();
