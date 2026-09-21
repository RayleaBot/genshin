const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ artis, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['八重神子'] };
    if (artis.is('影中沉凝的幻灭4')) {
        title.push('星超导');
        particularAttr.atk = 100;
        particularAttr.mastery = 100;
        particularAttr.dmg = 0;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['八重神子']);
}

return exports;})();
