const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.mainAttr = exports.defDmgIdx = exports.defParams = exports.details = void 0;
exports.details = [{
        title: '雷霆飞缒伤害',
        dmg: ({ talent, attr }, dmg) => dmg(talent.a['雷霆飞缒伤害'], 'a2,nightsoul')
    }, {
        title: 'E技能伤害',
        dmg: ({ talent, attr }, dmg) => dmg(talent.e['技能伤害'], 'e,nightsoul')
    }, {
        title: 'E后Q技能伤害',
        params: { Precise_Movement: true },
        dmg: ({ talent, attr }, dmg) => dmg(talent.q['技能伤害'], 'q,nightsoul')
    }, {
        title: '动能标示攻击力最多提升',
        params: { Precise_Movement: true },
        dmg: ({ talent, calc, attr }) => {
            return {
                avg: Math.min(calc(attr.atk) * 27 / 100, talent.q['最大攻击力加成'])
            };
        }
    }, {
        title: '「热身效应」每跳治疗',
        dmg: ({ calc, attr }) => {
            return {
                avg: calc(attr.atk) * 60 / 100
            };
        }
    }];
exports.defParams = { Nightsoul: true };
exports.defDmgIdx = 2;
exports.mainAttr = 'atk,cpct,cdmg';
exports.buffs = [{
        check: ({ params }) => params.Precise_Movement === true,
        title: '伊安珊天赋：「雷霆飞缒」命中敌人后，伊安珊的攻击力提升[atkPct]%',
        data: {
            atkPct: 20
        }
    }, {
        title: '伊安珊6命：若夜魂值恢复量溢出，使队伍中自己的当前场上角色造成的伤害提升[dmg]%',
        cons: 6,
        data: {
            dmg: 25
        }
    }];
exports.createdBy = '冰翼';

return exports;})();
