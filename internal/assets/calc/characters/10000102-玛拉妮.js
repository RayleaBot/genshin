const characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createdBy = exports.buffs = exports.mainAttr = exports.defDmgKey = exports.defParams = exports.details = void 0;
exports.details = [{
        title: '巨浪鲨鲨撕咬伤害',
        dmg: ({ talent, calc, attr }, { basic }) => basic(calc(attr.hp) * (talent.e['鲨鲨撕咬基础伤害'] + talent.e['巨浪鲨鲨撕咬伤害额外提升']) / 100, 'a,nightsoul')
    }, {
        check: ({ cons }) => cons < 6,
        title: '巨浪鲨鲨撕咬蒸发',
        dmgKey: 'e',
        dmg: ({ talent, calc, attr }, { basic }) => basic(calc(attr.hp) * (talent.e['鲨鲨撕咬基础伤害'] + talent.e['巨浪鲨鲨撕咬伤害额外提升']) / 100, 'a,nightsoul', 'vaporize')
    }, {
        check: ({ cons }) => cons >= 1,
        title: '强化巨浪鲨鲨撕咬蒸发',
        dmgKey: 'e',
        params: { cons1: true },
        dmg: ({ talent, calc, attr }, { basic }) => basic(calc(attr.hp) * (talent.e['鲨鲨撕咬基础伤害'] + talent.e['巨浪鲨鲨撕咬伤害额外提升']) / 100, 'a,nightsoul', 'vaporize')
    }, {
        title: '爆瀑飞弹伤害',
        dmg: ({ talent, calc, attr }, { basic }) => basic(calc(attr.hp) * talent.q['技能伤害'] / 100, 'q,nightsoul')
    }, {
        title: '爆瀑飞弹蒸发',
        dmg: ({ talent, calc, attr }, { basic }) => basic(calc(attr.hp) * talent.q['技能伤害'] / 100, 'q,nightsoul', 'vaporize')
    }];
exports.defParams = { Nightsoul: true };
exports.defDmgKey = 'e';
exports.mainAttr = 'hp,cpct,cdmg,mastery';
exports.buffs = [{
        title: '踏鲨破浪：3层浪势充能使鲨鲨撕咬造成的伤害提升[aPlus]巨浪鲨鲨撕咬伤害额外提升[_aPlus]',
        sort: 9,
        data: {
            aPlus: ({ talent, calc, attr }) => calc(attr.hp) * talent.e['浪势充能伤害提升'] / 100 * 3,
            _aPlus: ({ talent, calc, attr }) => calc(attr.hp) * talent.e['巨浪鲨鲨撕咬伤害额外提升'] / 100
        }
    }, {
        title: '玛拉妮天赋：3层逐浪心得使爆瀑飞弹伤害提升[qPlus]',
        sort: 9,
        data: {
            qPlus: ({ calc, attr }) => calc(attr.hp) * 45 / 100
        }
    }, {
        check: ({ params }) => params.cons1 === true,
        title: '玛拉妮1命：第一次巨浪鲨鲨撕咬及它所触发的鲨鲨飞弹造成的伤害提升[aPlus]',
        cons: 1,
        sort: 9,
        data: {
            aPlus: ({ calc, attr }) => calc(attr.hp) * 66 / 100
        }
    }, {
        title: '玛拉妮4命：爆瀑飞弹造成的伤害提升[qDmg]%',
        cons: 4,
        data: {
            qDmg: 75
        }
    }
    /*
    , {
      title: '幻境祝福：角色编入队伍后，生命值、攻击力与防御力提升[hpPct]%,该效果不论是否在幻想真境剧诗中都将生效。',
      data: {
        hpPct: 20,
        atkPct: 20,
        defPct: 20
      }
    }
    */
];
exports.createdBy = 'liangshi';

return exports;})();
