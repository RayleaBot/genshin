const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ artis, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['甘雨'] };
    if (artis.is('冰套4')) {
        title.push('永冻');
        particularAttr.mastery = 0;
        particularAttr.recharge = 55;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['甘雨']);
}

return exports;})();
