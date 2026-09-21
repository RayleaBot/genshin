const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ artis, cons, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['杜林'] };
    if (cons > 0 && artis.is('atk', 4)) {
        title.push('辅助');
        particularAttr.atk = 100;
        particularAttr.mastery = 30;
        particularAttr.dmg = 80;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['杜林']);
}

return exports;})();
