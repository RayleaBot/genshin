const scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1({ cons, def, weapon }) {
    let title = [];
    let particularAttr = { ...usefulAttr['闲云'] };
    let recharge = particularAttr.recharge;
    if (weapon.name === '鹤鸣余音') {
        recharge -= 30;
    }
    if (cons > 0) {
        recharge -= 30;
    }
    if (recharge != particularAttr.recharge) {
        title.push('高配');
        particularAttr.recharge = recharge;
    }
    if (cons === 6) {
        title.push('战斗');
        particularAttr.cpct = 100;
        particularAttr.cdmg = 100;
        particularAttr.dmg = 100;
        particularAttr.heal = 95;
    }
    if (title.length > 0) {
        return def(particularAttr, title);
    }
    return def(usefulAttr['闲云']);
}

return exports;})();
