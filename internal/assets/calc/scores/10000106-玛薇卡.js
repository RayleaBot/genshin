const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ attr, def }) {
    let title = [];
    let particularAttr = { ...usefulAttr['玛薇卡'] };
    if (attr.mastery < 50) {
        title.push('纯火/超载');
        particularAttr.atk = 85;
        particularAttr.mastery = 0;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['玛薇卡']);
}

return exports;})();
