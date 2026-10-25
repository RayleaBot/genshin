const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['行秋'] };
    if (attr.mastery > 120) {
        title.push('蒸发');
        particularAttr.mastery = 75;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['行秋']);
}

return exports;})();
