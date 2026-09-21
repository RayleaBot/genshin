const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['伊法'] };
    if (attr.cpct * 2 + attr.cdmg >= 240) {
        title.push('直伤');
        particularAttr.cpct = 100;
        particularAttr.cdmg = 100;
        particularAttr.dmg = 100;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['伊法']);
}

return exports;})();
