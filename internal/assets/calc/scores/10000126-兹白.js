const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ def, weapon }) {
    let title = [];
    let particularAttr = { ...usefulAttr['兹白'] };
    if (weapon.name === '息燧之笛') {
        title.push('息燧');
        particularAttr.def = 75;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['兹白']);
}

return exports;})();
