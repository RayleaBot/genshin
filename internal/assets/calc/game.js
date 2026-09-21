const artifactBuffs=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const attr = function (key, val, elem = '', unit = '%') {
    const keyMap = {
        hp: '生命值',
        hpPlus: '生命值',
        atk: '攻击力',
        def: '防御力',
        cpct: '暴击率',
        dmg: '元素伤害',
        phy: '物理伤害',
        shield: '护盾强效',
        heal: '治疗',
        mastery: '元素精通'
    };
    let ret = {
        title: `${keyMap[key]}提高${val}${unit}`,
        isStatic: true,
        data: {}
    };
    ret.data[key] = val;
    if (elem) {
        ret.elem = elem;
    }
    return ret;
};
const buffs = {
    行者之心: {
        2: attr('atkPct', 18),
        4: {
            title: '重击的暴击率提高30%',
            data: {
                a2Cpct: 30
            }
        }
    },
    勇士之心: {
        2: attr('atkPct', 18),
        4: {
            title: '对生命值高于50%的敌人，造成的伤害增加30%',
            data: {
                dmg: 30
            }
        }
    },
    守护之心: {
        2: attr('defPct', 30)
    },
    奇迹: {},
    战狂: {
        2: attr('cpct', 12),
        4: {
            title: '生命值低于70%时，暴击率提升24%',
            data: {
                cpct: 24
            }
        }
    },
    武人: {
        2: {
            title: '普攻与重击造成的伤害提高15%',
            data: {
                aDmg: 15,
                a2Dmg: 15
            }
        },
        4: {
            title: '施放元素战技后的8秒内，普攻和重击伤害提升25%',
            data: {
                aDmg: 25,
                a2Dmg: 25
            }
        }
    },
    教官: {
        2: attr('mastery', 80),
        4: {
            title: '触发元素反应后，队伍中所有角色的元素精通提高120点',
            data: {
                mastery: 120,
                masteryInc: 120
            }
        }
    },
    赌徒: {
        2: {
            title: '元素战技造成的伤害提升20%',
            data: {
                eDmg: 20
            }
        }
    },
    流放者: {
        2: attr('recharge', 20)
    },
    冒险家: {
        2: attr('hpPlus', 1000, '', '点')
    },
    幸运儿: {
        2: attr('defPlus', 100, '', '点')
    },
    学士: {
        2: attr('recharge', 20)
    },
    // TODO 此处是受治疗
    游医: {
        2: attr('heal', 20)
    },
    冰风迷途的勇士: {
        2: attr('dmg', 15, '冰'),
        4: {
            check: ({ element, mastery }) => element === '冰' && mastery != 'melt',
            title: '攻击处于冰元素影响下的敌人时，暴击率提高20%',
            data: {
                cpct: 20
            }
        }
    },
    平息鸣雷的尊者: {
        4: {
            check: ({ element }) => element === '雷',
            title: '对处于雷元素影响下的敌人造成的伤害提升35%',
            data: {
                dmg: 35
            }
        }
    },
    渡过烈火的贤人: {
        4: {
            check: ({ element }) => element === '火',
            title: '对处于火元素影响下的敌人造成的伤害提升35%',
            data: {
                dmg: 35
            }
        }
    },
    被怜爱的少女: {
        2: attr('heal', 15),
        4: {
            title: '施放元素战技或元素爆发后，受治疗效果加成提高20%',
            data: {
                healInc: 20
            }
        }
    },
    角斗士的终幕礼: {
        2: attr('atkPct', 18),
        4: {
            check: ({ weaponTypeName }) => ['单手剑', '双手剑', '长柄武器'].includes(weaponTypeName),
            title: '角色普通攻击造成的伤害提高35%',
            data: {
                aDmg: 35
            }
        }
    },
    翠绿之影: {
        2: attr('dmg', 15, '风'),
        4: {
            title: '扩散反应造成的伤害提升60%，降低对应元素抗性40%；星扩散反应造成的伤害提升20%',
            data: {
                swirl: 60,
                fykx: 40,
                stellarSwirl: 20,
                stellarVortex: 20
            }
        }
    },
    流浪大地的乐团: {
        2: attr('mastery', 80),
        4: {
            check: ({ weaponTypeName }) => ['法器', '弓'].includes(weaponTypeName),
            title: '角色重击造成的伤害提高35%',
            data: {
                a2Dmg: 35
            }
        }
    },
    如雷的盛怒: {
        2: attr('dmg', 15, '雷'),
        4: {
            title: '超载、感电、超导、超绽放反应造成的伤害提升40%，超激化反应带来的伤害提升提高20%，月感电、星超导反应造成的伤害提升20%。',
            data: {
                overloaded: 40,
                electroCharged: 40,
                superConduct: 40,
                hyperBloom: 40,
                stellarConduct: 20,
                aggravate: 20,
                lunarCharged: 20
            }
        }
    },
    炽烈的炎之魔女: {
        2: attr('dmg', 15, '火'),
        4: {
            check: ({ element }) => element === '火',
            title: '蒸发、融化伤害提高15%，[buffCount]层额外提高[dmg]%火元素伤害加成，超载、燃烧、烈绽放反应造成的伤害提升40%',
            data: {
                vaporize: 15,
                melt: 15,
                overloaded: 40,
                burning: 40,
                burgeon: 40,
                dmg: ({ params }) => (params.monv || 1) * 7.5,
                buffCount: ({ params }) => params.monv || 1
            }
        }
    },
    昔日宗室之仪: {
        2: {
            title: '元素爆发造成的伤害提升20%',
            data: {
                qDmg: 20
            }
        },
        4: {
            title: '施放元素爆发后，攻击力提升20%',
            check: ({ currentTalent }) => !currentTalent || currentTalent === 'q',
            data: {
                atkPct: 20
            }
        }
    },
    染血的骑士道: {
        2: attr('phy', 25),
        4: {
            title: '击败敌人后的10秒内，重击造成的伤害提升50%',
            data: {
                a2Dmg: 50
            }
        }
    },
    悠古的磐岩: {
        2: attr('dmg', 15, '岩'),
        4: {
            title: '获得元素反应晶片，对应元素伤害提高35%',
            check: ({ element }) => ['水', '火', '冰', '雷'].includes(element),
            //拾取幼岩龙蜥产生的岩元素晶片盾不能获得35%岩元素伤害加成
            data: {
                dmg: 35
            }
        }
    },
    逆飞的流星: {
        2: attr('shield', 35),
        4: {
            title: '处于护盾庇护下时，获得40%普攻和重击伤害加成',
            data: {
                aDmg: 40,
                a2Dmg: 40
            }
        }
    },
    沉沦之心: {
        2: attr('dmg', 15, '水'),
        4: {
            title: '施放元素战技后，普攻与重击伤害提高30%',
            data: {
                aDmg: 30,
                a2Dmg: 30
            }
        }
    },
    千岩牢固: {
        2: attr('hpPct', 20),
        4: {
            title: '元素战技命中敌人后，攻击力提升[atkPct]%，护盾强效提升[shield]%',
            data: {
                atkPct: 20,
                shield: 30
            }
        }
    },
    苍白之火: {
        2: attr('phy', 25),
        4: {
            title: '2层提高18%攻击力，物理伤害额外提高25%',
            data: {
                atkPct: 18,
                phy: 25
            }
        }
    },
    追忆之注连: {
        2: attr('atkPct', 18),
        4: {
            title: '施放元素战技后，普通攻击、重击、下落攻击造成的伤害提高50%',
            data: {
                aDmg: 50,
                a2Dmg: 50,
                a3Dmg: 50
            }
        }
    },
    绝缘之旗印: {
        2: attr('recharge', 20),
        4: {
            title: '基于元素充能效率提高元素爆发[qDmg]%伤害',
            sort: 4,
            data: {
                qDmg: ({ attr }) => Math.min(75, (attr.recharge.base + attr.recharge.plus) * 0.25)
            }
        }
    },
    华馆梦醒形骸记: {
        2: attr('defPct', 30),
        4: {
            title: '满层获得[defPct]%防御及[dmg]%岩伤加成',
            data: {
                defPct: 24,
                dmg: ({ element }) => ['岩'].includes(element) ? 24 : 0
            }
        }
    },
    海染砗磲: {
        2: attr('heal', 15)
    },
    辰砂往生录: {
        2: attr('atkPct', 18),
        4: {
            title: '满层提高48%攻击力',
            data: {
                atkPct: 48
            }
        }
    },
    来歆余响: {
        2: attr('atkPct', 18),
        4: {
            title: '触发提高普攻[aPlus]伤害',
            sort: 9,
            data: {
                aPlus: ({ attr }) => attr.atk * 0.35
            }
        }
    },
    深林的记忆: {
        2: attr('dmg', 15, '草'),
        4: {
            title: '元素战技或元素爆发命中敌人后，使命中目标的草元素抗性降低30%',
            check: ({ element }) => element === '草',
            data: {
                kx: 30
            }
        }
    },
    饰金之梦: {
        2: attr('mastery', 80),
        4: {
            title: '队伍存在[mArtisDiffCount]个不同元素类型角色，[sameCount]个相同类型角色，精通提高[mastery]，攻击力提高[atkPct]%',
            data: {
                mArtisDiffCount: ({ params }) => params.mArtisDiffCount || 3,
                sameCount: ({ params }) => 3 - (params.mArtisDiffCount || 3),
                mastery: ({ params }) => (params.mArtisDiffCount || 3) * 50,
                atkPct: ({ params }) => (3 - (params.mArtisDiffCount || 3)) * 14
            }
        }
    },
    沙上楼阁史话: {
        2: attr('dmg', 15, '风'),
        4: {
            title: '重击命中敌人后，普攻重击与下落攻击伤害提升40',
            data: {
                aDmg: 40,
                a2Dmg: 40,
                a3Dmg: 40
            }
        }
    },
    乐园遗落之花: {
        2: attr('mastery', 80),
        4: {
            title: '满层提高绽放、超绽放、烈绽放反应造成的伤害提升80%,月绽放反应伤害提升[lunarBloom]%',
            data: {
                bloom: 80,
                burgeon: 80,
                hyperBloom: 80,
                lunarBloom: 20
            }
        }
    },
    水仙之梦: {
        2: attr('dmg', 15, '水'),
        4: {
            title: '3层Buff下提高攻击力[atkPct]%，水伤[dmg]%',
            data: {
                atkPct: 25,
                dmg: ({ element }) => ['水'].includes(element) ? 15 : 0
            }
        }
    },
    花海甘露之光: {
        2: attr('hpPct', 20),
        4: {
            title: '5层Buff下提高元素战技与元素爆发伤害50%',
            data: {
                eDmg: 50,
                qDmg: 50
            }
        }
    },
    逐影猎人: {
        2: {
            title: '普通攻击与重击造成的伤害提高15%',
            data: {
                aDmg: 15,
                a2Dmg: 15
            }
        },
        4: {
            title: '3层Buff下提高暴击率36%',
            data: {
                cpct: 36
            }
        }
    },
    黄金剧团: {
        2: {
            title: '元素战技造成的伤害提升20%',
            data: {
                eDmg: 20
            }
        },
        4: {
            title: '元素战技造成的伤害额外提升[eDmg]%',
            data: {
                eDmg: ({ params }) => params.off_field === false ? 25 : 50
            }
        }
    },
    昔时之歌: {
        2: attr('heal', 15),
        4: {
            title: '触发后，普通攻击、重击、下落攻击、元素战技与元素爆发伤害提高1200',
            sort: 9,
            data: {
                aPlus: 1200,
                a2Plus: 1200,
                a3Plus: 1200,
                ePlus: 1200,
                qPlus: 1200
            }
        }
    },
    回声之林夜话: {
        2: attr('atkPct', 18),
        4: {
            check: ({ element }) => element === '岩',
            title: '施放元素战技后，岩元素伤害加成提升50%',
            data: {
                dmg: 50
            }
        }
    },
    谐律异想断章: {
        2: attr('atkPct', 18),
        4: {
            title: '生命之契的数值提升或降低[buff]次，角色造成的伤害提升[dmg]%',
            data: {
                buff: ({ params, weapon }) => (params.BondOfLifeGet || 0) + (params.DecreasedBondOfLife || 0) + (['纯水流华', '海渊终曲'].includes(weapon.name) ? ((params.SkillsUse || 1) >= 1 ? ((params.HealNumber || 0) + 1) : 0) : 0) + (weapon.name === "赤月之形" ? ((params.HealNumber || 0) + 1) : 0),
                dmg: ({ params, weapon }) => Math.min(((params.BondOfLifeGet || 0) + (params.DecreasedBondOfLife || 0) + (['纯水流华', '海渊终曲'].includes(weapon.name) ? ((params.SkillsUse || 1) >= 1 ? ((params.HealNumber || 0) + 1) : 0) : 0) + (weapon.name === "赤月之形" ? ((params.HealNumber || 0) + 1) : 0)), 3) * 18
            }
        }
    },
    未竟的遐思: {
        2: attr('atkPct', 18),
        4: {
            title: '存在处于燃烧状态下的敌人时，伤害提升[dmg]%',
            data: {
                dmg: 10 * 5
            }
        }
    },
    烬城勇者绘卷: {
        4: {
            title: '触发元素反应后，元素伤害加成提升[dmg]%',
            data: {
                dmg: ({ params }) => params.Nightsoul === true ? 40 : 12
            }
        }
    },
    黑曜秘典: {
        2: {
            check: ({ params }) => params.Nightsoul === true,
            title: '在场上处于夜魂加持状态时，造成的伤害提高[dmg]%',
            data: {
                dmg: 15
            }
        },
        4: {
            check: ({ params }) => params.Nightsoul === true,
            title: '在场上消耗夜魂值后，暴击率提高[cpct]%',
            data: {
                cpct: 40
            }
        }
    },
    长夜之誓: {
        2: {
            title: '下落攻击造成的伤害提升[a3Dmg]%',
            data: {
                a3Dmg: 25
            }
        },
        4: {
            title: '5层buff,下落攻击伤害提升[a3Dmg]%',
            data: {
                a3Dmg: 15 * 5
            }
        }
    },
    深廊终曲: {
        2: attr('dmg', 15, '冰'),
        4: {
            title: '元素能量为0时普攻与元素爆发伤害提高[aDmg]%',
            data: {
                aDmg: 60,
                qDmg: 60
            }
        }
    },
    穹境示现之夜: {
        2: attr('mastery', 80),
        4: {
            title: '依据队伍的月兆与月辉明光效果，暴击率提升[cpct]%,月曜反应造成的伤害提升[lunarBloom]%',
            data: {
                cpct: ({ params }) => Math.min(((params.Moonsign || 0) * 15), 30),
                lunarCharged: ({ params }) => (params["月辉明光"] || 1) * 10,
                lunarBloom: ({ params }) => (params["月辉明光"] || 1) * 10,
                lunarCrystallize: ({ params }) => (params["月辉明光"] || 1) * 10
            }
        }
    },
    纺月的夜歌: {
        2: attr('recharge', 20),
        4: {
            title: '依据队伍的月兆与月辉明光效果，元素精通提升[mastery],月曜反应造成的伤害提升[lunarBloom]%',
            data: {
                mastery: ({ params }) => Math.min(((params.Moonsign || 0) * 60), 120),
                lunarCharged: ({ params }) => (params["月辉明光"] || 1) * 10,
                lunarBloom: ({ params }) => (params["月辉明光"] || 1) * 10,
                lunarCrystallize: ({ params }) => (params["月辉明光"] || 1) * 10
            }
        }
    },
    晨星与月的晓歌: {
        2: attr('mastery', 80),
        4: {
            title: '装备者处于队伍后台时，造成的月曜反应伤害提升20%；队伍的月兆等级至少为满辉时，造成的月曜反应伤害进一步提升40%',
            data: {
                lunarCharged: ({ params }) => (params.Moonsign || 0) > 1 ? 60 : 20,
                lunarBloom: ({ params }) => (params.Moonsign || 0) > 1 ? 60 : 20,
                lunarCrystallize: ({ params }) => (params.Moonsign || 0) > 1 ? 60 : 20
            }
        }
    },
    风起之日: {
        2: attr('atkPct', 18),
        4: {
            title: '普通攻击、重击、元素战技或元素爆发命中敌人后，攻击力提高[atkPct]%。若装备者已经完成了「魔女的课业」，则额外使装备者的暴击率提升[cpct]%',
            data: {
                atkPct: 25,
                cpct: ({ params }) => params.Hexenzirkel ? 20 : 0
            }
        }
    },
    天之美赐: {
        2: attr('recharge', 20),
        4: {
            title: '依据魔导效果，施放元素战技后附近的所有角色获得[dmg]%元素伤害加成',
            data: {
                dmg: ({ params }) => params.Hexenzirkel ? 40 : 0
            }
        }
    },
    影中沉凝的幻灭: {
        2: attr('atkPct', 18),
        4: {
            check: ({ element }) => ['雷', '冰'].includes(element),
            title: '超导反应造成的伤害提升[superConduct]%；星超导反应造成的伤害提升[stellarConduct]%；攻击受到超导或星超导反应影响的敌人暴击率提高[cpct]%',
            data: {
                superConduct: 80,
                stellarConduct: 40,
                cpct: 16
            }
        }
    },
    炉火融炼之心: {
        2: attr('atkPct', 18),
        4: {
            check: ({ element }) => ['雷', '风', '冰'].includes(element),
            title: '装备者触发星烁反应或造成星烁反应伤害后的12秒内，攻击力提升[atkPct]%，队伍中附近的所有角色造成的星烁反应伤害提升[stellarConduct]%。',
            data: {
                atkPct: 12,
                stellarConduct: 50
            }
        }
    },
    血红之证: {
        2: attr('atkPct', 18),
        4: {
            check: ({ element }) => ['风', '冰'].includes(element),
            title: '装备者触发星扩散反应后的10秒内，暴击率提升[cpct]%，星扩散反应伤害提升[stellarSwirl]%。',
            data: {
                cpct: 16,
                stellarSwirl: 40,
                stellarVortex: 40
            }
        }
    }
};
exports.default = buffs;

return exports;})().default;
const weaponBuffs={};
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(step, staticStep) {
    return {
        猎弓: false,
        历练的猎弓: false,
        鸦羽弓: {
            check: ({ element }) => ['水', '火'].includes(element),
            title: '对处于水或火元素影响下的敌人，造成的伤害提高[dmg]%',
            refine: {
                dmg: step(12)
            }
        },
        神射手之誓: {
            title: '针对要害造成的伤害提升[a2Dmg]%',
            refine: {
                a2Dmg: step(24)
            }
        },
        反曲弓: false,
        弹弓: {
            title: '普攻与重击的箭矢0.3秒内击中敌人，伤害增加[a2Dmg]%',
            refine: {
                aDmg: step(36, 6),
                a2Dmg: step(36, 6)
            }
        },
        信使: false,
        黑檀弓: false,
        西风猎弓: false,
        绝弦: {
            title: '元素战技与元素爆发的伤害提高[eDmg]%',
            refine: {
                eDmg: step(24),
                qDmg: step(24)
            }
        },
        祭礼弓: false,
        宗室长弓: {
            title: '3层提高暴击率[cpct]%',
            buffCount: 3,
            refine: {
                cpct: step(8)
            }
        },
        弓藏: {
            title: '普攻造成的伤害提升[aDmg]%，重击造成的伤害下降10%',
            refine: {
                aDmg: step(40),
                a2Dmg: -10
            }
        },
        试作澹月: {
            title: '重击命中要害提高[atkPct]%攻击力',
            refine: {
                atkPct: step(36)
            }
        },
        钢轮弓: {
            title: '普通攻击与重击命中时，满层提升[atkPct]%',
            buffCount: 4,
            refine: {
                atkPct: step(4)
            }
        },
        黑岩战弓: {
            title: '击败敌人后，攻击力满层提升[atkPct]%',
            buffCount: 3,
            refine: {
                atkPct: step(12)
            }
        },
        苍翠猎弓: false,
        暗巷猎手: {
            title: '满层提高[dmg]%伤害',
            refine: {
                dmg: step(20)
            }
        },
        落霞: {
            title: '二层状态下提高伤害[dmg]%',
            refine: {
                dmg: step(10, 2.5)
            }
        },
        幽夜华尔兹: {
            title: 'Buff下普攻及元素战技造成的伤害提升[aDmg]%',
            refine: {
                aDmg: step(20),
                eDmg: step(20)
            }
        },
        风花之颂: {
            title: '施放元素战技时攻击力提升[atkPct]%',
            refine: {
                atkPct: step(16)
            }
        },
        破魔之弓: {
            title: '满能量下普攻伤害提高[aDmg]%，重击伤害提高[a2Dmg]%',
            buffCount: 2,
            refine: {
                aDmg: step(16),
                a2Dmg: step(12)
            }
        },
        掠食者: [{
                check: ({ element }) => element === '冰',
                title: '满Buff普攻与重击伤害提高[aDmg]%',
                refine: {
                    aDmg: [20]
                }
            }, {
                check: ({ characterName }) => characterName === '埃洛伊',
                title: '攻击力提升66点',
                refine: {
                    atkPlus: 66
                }
            }],
        曚云之月: {
            title: '满层元素爆发伤害提高[qDmg]%',
            refine: {
                qDmg: step(40)
            }
        },
        王下近侍: {
            title: '施放元素战技或元素爆发时精通提高[mastery]',
            refine: {
                mastery: step(60, 20)
            }
        },
        竭泽: false,
        天空之翼: staticStep('cdmg', 20),
        阿莫斯之弓: [{
                title: '普攻与重击伤害提高[a2Dmg]%',
                refine: {
                    aDmg: step(12),
                    a2Dmg: step(12)
                }
            }, {
                title: '5段Buff重击伤害提高[a2Dmg]%',
                buffCount: 5,
                refine: {
                    aDmg: step(8),
                    a2Dmg: step(8)
                }
            }],
        终末嗟叹之诗: [staticStep('mastery', 60), {
                title: 'Buff下提高元素精通[mastery],攻击力[atkPct]%',
                sort: 0,
                refine: {
                    mastery: step(100),
                    atkPct: step(20)
                }
            }],
        冬极白星: [{
                title: '元素战技与元素爆发伤害提高[eDmg]%',
                refine: {
                    eDmg: step(12),
                    qDmg: step(12)
                }
            }, {
                title: '满Buff下提高攻击力[atkPct]%',
                refine: {
                    atkPct: step(48, 12)
                }
            }],
        飞雷之弦振: [staticStep('atkPct', 20), {
                title: '满Buff下提高普攻伤害[aDmg]%',
                refine: {
                    aDmg: step(40)
                }
            }],
        若水: [staticStep('hpPct', 16), {
                title: '生命值提高[_hpPct]%，伤害提高[dmg]%',
                refine: {
                    _hpPct: step(16),
                    dmg: step(20)
                }
            }],
        陨龙之梦: {
            title: '护盾+满Buff提高攻击力[atkPct]%',
            buffCount: 10,
            refine: {
                atkPct: step(4)
            }
        },
        猎人之径: [staticStep('dmg', 12), {
                title: '重击造成的伤害值提高[a2Plus]',
                sort: 9,
                data: {
                    a2Plus: ({ attr, calc, refine }) => calc(attr.mastery) * step(160)[refine] / 100
                }
            }],
        鹮穿之喙: {
            title: '重击命中敌人2层提高元素精通[mastery]点',
            refine: {
                mastery: step(80)
            }
        },
        最初的大魔术: [{
                title: '重击造成的伤害提升[a2Dmg]%',
                refine: {
                    a2Dmg: step(16)
                }
            }, {
                title: '满Buff下提高攻击力[atkPct]%',
                refine: {
                    atkPct: step(48)
                }
            }],
        烈阳之嗣: {
            title: '对于灼心状态下的敌人造成的伤害提升[dmg]%',
            refine: {
                dmg: step(28)
            }
        },
        测距规: {
            title: '满层下，提高[atkPct]%攻击力与[dmg]%所有元素伤害加成',
            refine: {
                atkPct: [3 * 3, 4 * 3, 5 * 3, 6 * 3, 7 * 3],
                dmg: [7 * 3, 8.5 * 3, 10 * 3, 11.5 * 3, 13 * 3]
            }
        },
        静谧之曲: {
            title: '受到治疗后，造成的伤害提升[dmg]%',
            refine: {
                dmg: step(16)
            }
        },
        筑云: {
            title: '元素能量减少后，装备者的元素精通提升[mastery]%',
            refine: {
                mastery: step(40, 10)
            }
        },
        白雨心弦: {
            title: '满层下，生命值上限提升[hpPct]%元素爆发的暴击率提[qCpct]%',
            refine: {
                hpPct: step(40, 10),
                qCpct: step(28)
            }
        },
        碎链: {
            title: '三名与装备者元素类型不同的角色，攻击力提升[atkPct]%，元素精通提升[mastery]点',
            refine: {
                atkPct: [4.8 * 3, 6 * 3, 7.2 * 3, 8.4 * 3, 9.6 * 3],
                mastery: step(24)
            }
        },
        星鹫赤羽: [{
                check: ({ element }) => !['草', '岩'].includes(element),
                title: '触发扩散反应后，攻击力提高[atkPct]%',
                refine: {
                    atkPct: step(24)
                }
            }, {
                title: '存在至少2名元素类型不同的角色，重击造成的伤害提高[a2Dmg]%,元素爆发伤害提高[qDmg]%',
                refine: {
                    a2Dmg: step(48),
                    qDmg: step(24)
                }
            }],
        缀花之翎: {
            title: '重击造成的伤害提升[a2Dmg]%',
            refine: {
                a2Dmg: step(6 * 6)
            }
        },
        罗网勾针: {
            title: '触发元素反应后元素精通提升[mastery]',
            data: {
                mastery: ({ params, refine }) => step(60)[refine] * ((params.Moonsign || 0) >= 2 ? 2 : 1)
            }
        },
        虹蛇的雨弦: {
            title: '装备者处于队伍后台时，装备者的攻击命中敌人后的8秒内，攻击力提升[atkPct]%',
            refine: {
                atkPct: step(28)
            }
        },
        黎明破晓之史: {
            title: '普通攻击、元素战技和元素爆发造成的伤害提升[aDmg]%',
            refine: {
                aDmg: step(60),
                eDmg: step(60),
                qDmg: step(60)
            }
        },
        霜结的誓金枝: [staticStep('defPct', 16), {
                check: ({ element }) => element === '岩',
                title: '装备者的元素战技或月结晶攻击命中敌人时，造成的岩元素伤害提升[dmg]%，月结晶反应伤害提升[lunarCrystallize]%',
                refine: {
                    dmg: step(40),
                    lunarCrystallize: step(40)
                }
            }],
        悬黎千钧: {
            title: '队伍中有一名角色元素类型与装备者相同，元素精通提升[mastery]点；有2名角色与装备者元素类型不同，攻击力提升[atkPct]%',
            refine: {
                mastery: step(64),
                atkPct: step(12 * 2)
            }
        },
        霜雪誓约: {
            title: '施放元素战技后的12秒内，元素精通提升[mastery]点',
            refine: {
                mastery: step(120)
            }
        }
    };
}

return exports;})().default,"gs"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(step, staticStep) {
    return {
        翡玉法球: {
            check: ({ element }) => element === '水',
            title: '触发蒸发、感电、冰冻或水元素扩散反应后的12秒内，攻击力提高[atkPct]%',
            refine: {
                atkPct: step(20)
            }
        },
        魔导绪论: {
            check: ({ element }) => ['水', '雷'].includes(element),
            title: '对处于水元素或雷元素影响下的敌人，造成的伤害提高[dmg]%',
            refine: {
                dmg: step(12)
            }
        },
        甲级宝珏: {
            title: '击败敌人后攻击力提升[atkPct]%',
            refine: {
                atkPct: step(12, 2)
            }
        },
        黑岩绯玉: {
            title: '击败敌人后，满层攻击力提升[atkPct]%',
            buffCount: 3,
            refine: {
                atkPct: step(12)
            }
        },
        万国诸海图谱: {
            title: '触发元素反应后，满层提高[dmg]%的元素伤害',
            buffCount: 2,
            refine: {
                dmg: step(8)
            }
        },
        宗室秘法录: {
            title: '3层状态下提高暴击率[cpct]%',
            buffCount: 3,
            refine: {
                cpct: step(8)
            }
        },
        匣里日月: {
            title: '普攻提高元素战技与爆发伤害[eDmg]%，元素战技与爆发提高普攻伤害[aDmg]%',
            refine: {
                aDmg: step(20),
                eDmg: step(20),
                qDmg: step(20)
            }
        },
        流浪乐章: {
            title: '咏叹调下全元素伤害提升[dmg]%',
            refine: {
                dmg: step(48, 12)
            }
        },
        暗巷的酒与诗: {
            title: '冲刺后攻击力提升[atkPct]%',
            refine: {
                atkPct: step(20)
            }
        },
        嘟嘟可故事集: {
            title: '普攻提高重击伤害[a2Dmg]%，重击提高攻击力[atkPct]%',
            refine: {
                a2Dmg: step(16),
                atkPct: step(8)
            }
        },
        白辰之环: {
            title: '与雷元素反应后提高元素反应加成[dmg]%',
            refine: {
                dmg: step(10, 2.5)
            }
        },
        证誓之明瞳: {
            title: '施放元素战技后，元素充能效率提升[recharge]%',
            refine: {
                recharge: step(24)
            }
        },
        四风原典: {
            title: '满层获得[dmg]%的元素伤害加成',
            buffCount: 4,
            refine: {
                dmg: step(8)
            }
        },
        天空之卷: staticStep('dmg', 12),
        尘世之锁: [staticStep('shield', 20), {
                title: '护盾+满层情况下攻击力提高[atkPct]%',
                buffCount: 10,
                refine: {
                    atkPct: step(4)
                }
            }],
        不灭月华: [staticStep('heal', 10, 2.5), {
                title: '普攻伤害增加[aPlus]',
                sort: 9,
                data: {
                    aPlus: ({ attr, calc, refine }) => calc(attr.hp) * step(1, 0.5)[refine] / 100
                }
            }],
        神乐之真意: {
            title: '满层提高元素战技伤害[eDmg]%，星超导反应伤害提升[stellarConduct]%，元素伤害提高[dmg]%',
            refine: {
                eDmg: [12 * 3, 15 * 3, 18 * 3, 21 * 3, 24 * 3],
                dmg: step(12),
                stellarConduct: step(12 * 3)
            }
        },
        盈满之实: {
            title: '满层提高元素精通[mastery]，攻击力降低25%',
            refine: {
                mastery: step(24 * 5, 3 * 5),
                atkPct: -25
            }
        },
        流浪的晚星: {
            title: '基于元素精通提升攻击力[atkPlus]',
            sort: 6,
            data: {
                atkPlus: ({ attr, calc, refine }) => step(24)[refine] * calc(attr.mastery) / 100
            }
        },
        图莱杜拉的回忆: {
            title: '满Buff下提高普攻伤害[aDmg]%',
            refine: {
                aDmg: step(48)
            }
        },
        千夜浮梦: {
            title: '3个不同元素队友满层，元素伤害提高[dmg]%',
            buffCount: 3,
            refine: {
                dmg: step(10, 4)
            }
        },
        碧落之珑: {
            title: '释放元素爆发后基于生命值提高元素伤害[dmg]%',
            sort: 9,
            data: {
                dmg: ({ attr, calc, refine }) => Math.min(Math.floor(calc(attr.hp) / 1000) * step(0.3, 0.2)[refine], step(12, 8)[refine])
            }
        },
        纯水流华: [{
                title: '生命之契提升[dmg]%全部元素伤害加成',
                sort: 9,
                data: {
                    dmg: ({ attr, calc, refine }) => Math.min(Math.floor(calc(attr.hp) / 1000) * 2 * step(0.24)[refine], step(12)[refine])
                }
            }, {
                title: '释放元素战技全部元素伤害加成提升[dmg]%',
                refine: {
                    dmg: step(8)
                }
            }],
        金流监督: [staticStep('atkPct', 16), {
                title: '满层下，普通攻击造成的伤害提升[aDmg]%，星超导反应伤害提升[stellarConduct]%，重击造成的伤害提升[a2Dmg]%',
                refine: {
                    aDmg: step(16 * 3),
                    a2Dmg: step(14 * 3),
                    stellarConduct: step(14 * 3)
                }
            }],
        遗祀玉珑: [{
                title: '处于队伍后台超过5秒后，生命值上限提升[hpPct]%，元素精通提升[mastery]点',
                refine: {
                    hpPct: step(32),
                    mastery: step(40)
                }
            }],
        万世流涌大典: [staticStep('hpPct', 16), {
                title: '满层下，重击造成的伤害提升[a2Dmg]%',
                refine: {
                    a2Dmg: [14 * 3, 18 * 3, 22 * 3, 26 * 3, 30 * 3]
                }
            }],
        无垠蔚蓝之歌: [{
                title: '满层下，普通攻击造成的伤害提升[aDmg]%，重击造成的伤害提升[a2Dmg]%',
                refine: {
                    aDmg: step(8 * 3),
                    a2Dmg: step(6 * 3)
                }
            }],
        鹤鸣余音: [{
                title: '下落攻击命中敌人后，下落攻击造成的伤害提高[a3Dmg]%',
                refine: {
                    a3Dmg: step(28, 13)
                }
            }],
        冲浪时光: [staticStep('hpPct', 20), {
                title: '施放元素战技后，普通攻击造成的伤害提升[aDmg]%',
                refine: {
                    aDmg: step(12 * 4)
                }
            }],
        木棉之环: {
            title: '施放元素战技时，基于生命值提升普攻造成的伤害[aDmg]%',
            sort: 9,
            data: {
                aDmg: ({ attr, calc, refine }) => Math.min(Math.floor(calc(attr.hp) / 1000) * step(0.6, 0.1)[refine], step(16)[refine])
            }
        },
        乘浪的回旋: {
            title: '施放元素战技后，生命值上限提升[hpPct]%',
            refine: {
                hpPct: step(44)
            }
        },
        祭星者之望: [staticStep('mastery', 100), {
                title: '创造护盾后造成的伤害提升[dmg]%',
                refine: {
                    dmg: step(28)
                }
            }],
        寝正月初晴: {
            title: '触发[_buff]种方式，元素精通提升[mastery]',
            //扩散作为无序元素反应，冰雷火水后手也可触发扩散反应
            data: {
                _buff: ({ element }) => !['草', '岩'].includes(element) ? 3 : 2,
                mastery: ({ element, refine }) => !['草', '岩'].includes(element) ? step(120 + 96 + 32)[refine] : step(96 + 32)[refine]
            }
        },
        溢彩心念: [staticStep('atkPct', 28), {
                title: '施放元素战技或元素爆发后进行下落攻击造成的暴击伤害提升[a3Cdmg]',
                refine: {
                    a3Cdmg: step(28 + 40)
                }
            }],
        天光的纺琴: {
            title: '施放元素战技后元素精通提升[mastery]',
            refine: {
                mastery: step(100)
            }
        },
        乌髓孑灯: {
            title: '绽放反应造成的伤害提升[bloom]%，月绽放反应造成的伤害提升[lunarBloom]%',
            data: {
                bloom: ({ refine }) => step(48)[refine],
                lunarBloom: ({ params, refine }) => step(12)[refine] * ((params.Moonsign || 0) >= 2 ? 2 : 1)
            }
        },
        纺夜天镜: {
            title: '元素精通提升[mastery],绽放反应伤害提升[bloom]%,超绽放、烈绽放伤害提升[burgeon]%,月绽放伤害提升[lunarBloom]%',
            data: {
                mastery: ({ params, refine, element }) => ((params.Moonsign || 0) >= 1 ? step(60)[refine] : 0) + (['水', '草'].includes(element) ? step(60)[refine] : 0),
                bloom: ({ params, refine, element }) => (((params.Moonsign || 0) >= 1 ? 1 : 0) + (['水', '草'].includes(element) ? 1 : 0)) === 2 ? step(120)[refine] : 0,
                burgeon: ({ params, refine, element }) => (((params.Moonsign || 0) >= 1 ? 1 : 0) + (['水', '草'].includes(element) ? 1 : 0)) === 2 ? step(80)[refine] : 0,
                hyperBloom: ({ params, refine, element }) => (((params.Moonsign || 0) >= 1 ? 1 : 0) + (['水', '草'].includes(element) ? 1 : 0)) === 2 ? step(80)[refine] : 0,
                lunarBloom: ({ params, refine, element }) => (((params.Moonsign || 0) >= 1 ? 1 : 0) + (['水', '草'].includes(element) ? 1 : 0)) === 2 ? step(40)[refine] : 0
            }
        },
        霜辰: {
            title: '元素精通提升[mastery]',
            refine: {
                mastery: step(120)
            }
        },
        真语秘匣: [staticStep('cpct', 8), {
                title: '施放元素战技时元素精通提升[mastery]点, 暴击伤害提升[cdmg]%',
                data: {
                    mastery: ({ params, refine }) => ((params.Moonsign || 0) >= 1 ? 1.5 : 1) * step(80)[refine],
                    cdmg: ({ params, refine }) => ((params.Moonsign || 0) >= 1 ? 1.5 : 1) * step(24)[refine]
                }
            }],
        帷间夜曲: [staticStep('hpPct', 10, 2), {
                title: '装备者触发月曜反应或对敌人造成月曜反应伤害时，生命值上限进一步提高[hpPct]%，月曜反应伤害的暴击伤害提升[cdmg]%',
                data: {
                    hpPct: ({ params, refine }) => ((params.Moonsign || 0) > 0 ? 1 : 0) * step(14, 2)[refine],
                    cdmg: ({ params, refine }) => ((params.Moonsign || 0) > 0 ? 1 : 0) * step(60, 20)[refine]
                }
            }],
        尘光七谕: [staticStep('atkPct', 12, 3), {
                title: '装备者创造护盾后，使当前场上角色造成的伤害提升[dmg]%',
                data: {
                    dmg: ({ attr, calc, refine }) => Math.min(Math.floor(calc(attr.atk) / 1000) * step(10, 3)[refine], step(26, 8)[refine])
                }
            }],
        群王局戏: {
            title: '施放元素战技后，攻击力提升[atkPct]%，元素精通提升[mastery]点',
            refine: {
                atkPct: step(20),
                mastery: step(100)
            }
        },
        寸心余响: [{
                title: '触发元素反应后，元素精通提高[mastery]点',
                refine: {
                    mastery: step(60)
                }
            }, {
                check: ({ element }) => ['雷', '风', '冰'].includes(element),
                title: '触发星烁反应后，星烁反应伤害提升[stellarConduct]%',
                refine: {
                    stellarConduct: step(16),
                    stellarSwirl: step(16),
                    stellarVortex: step(16)
                }
            }]
    };
}

return exports;})().default,"gs"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(step, staticStep) {
    return {
        沐浴龙血的剑: {
            check: ({ element }) => ['火', '雷'].includes(element),
            title: '对处于火元素或雷元素影响下的敌人，造成的伤害提高[dmg]%',
            refine: {
                dmg: step(12),
                phy: step(12)
            }
        },
        铁影阔剑: {
            title: '生命值低于70%时，提高[a2Dmg]%重击伤害',
            refine: {
                a2Dmg: step(30, 5)
            }
        },
        飞天大御剑: {
            title: '满层提高攻击力[atkPct]%',
            buffCount: 4,
            refine: {
                atkPct: step(6, 1)
            }
        },
        黑岩斩刀: {
            title: '击败敌人满Buff下攻击力提升[atkPct]%',
            buffCount: 3,
            refine: {
                atkPct: step(12)
            }
        },
        千岩古剑: {
            title: '四璃月角色提升攻击力[atkPct]%及暴击率[cpct]%',
            buffCount: 4,
            refine: {
                atkPct: step(3, 1),
                cpct: step(3, 1)
            }
        },
        雨裁: {
            check: ({ element }) => ['水', '雷'].includes(element),
            title: '对处于水元素或雷元素影响下的敌人，造成的伤害提高[dmg]%',
            refine: {
                dmg: step(20, 4),
                phy: step(20, 4)
            }
        },
        宗室大剑: {
            title: '3层Buff提高暴击率[cpct]%',
            buffCount: 3,
            refine: {
                cpct: step(8)
            }
        },
        螭骨剑: {
            title: '满Buff提高伤害[dmg]%',
            buffCount: 5,
            refine: {
                dmg: step(6, 1),
                phy: step(6, 1)
            }
        },
        钟剑: {
            title: '角色处于护盾庇护下时，造成的伤害提升[dmg]%',
            refine: {
                dmg: step(12),
                phy: step(12)
            }
        },
        白影剑: {
            title: '满Buff提升攻击力及防御力[atkPct]%',
            buffCount: 4,
            refine: {
                atkPct: step(6),
                defPct: step(6)
            }
        },
        桂木斩长正: {
            title: '元素战技造成的伤害提升[eDmg]%',
            refine: {
                eDmg: step(6)
            }
        },
        衔珠海皇: {
            title: '元素爆发造成的伤害提升[qDmg]%',
            refine: {
                qDmg: step(12)
            }
        },
        恶王丸: {
            title: '满层元素爆发造成的伤害提升[qDmg]%',
            refine: {
                qDmg: step(40)
            }
        },
        天空之傲: {
            title: '造成伤害提高[dmg]%',
            refine: {
                dmg: step(8),
                phy: step(8)
            }
        },
        狼的末路: [staticStep('atkPct', 20)],
        无工之剑: [staticStep('shield', 20), {
                title: '满Buff护盾下攻击力提高[atkPct]%',
                buffCount: 10,
                refine: {
                    atkPct: step(4)
                }
            }],
        松籁响起之时: [staticStep('atkPct', 16), {
                title: 'Buff状态下提高攻击力[atkPct]%',
                refine: {
                    atkPct: step(20)
                }
            }],
        赤角石溃杵: [staticStep('defPct', 28), {
                title: '普通攻击与重击造成的伤害值提高[aPlus]',
                sort: 9,
                data: {
                    aPlus: ({ attr, calc, refine }) => calc(attr.def) * step(40)[refine] / 100,
                    a2Plus: ({ attr, calc, refine }) => calc(attr.def) * step(40)[refine] / 100
                }
            }],
        森林王器: {
            title: '拾取种识之叶的角色元素精通提升[mastery]',
            refine: {
                mastery: step(60)
            }
        },
        玛海菈的水色: {
            title: '基于元素精通提升攻击力[atkPlus]',
            sort: 6,
            data: {
                atkPlus: ({ attr, calc, refine }) => step(24)[refine] * calc(attr.mastery) / 100
            }
        },
        饰铁之花: {
            title: '元素战技或触发元素反应提高[atkPct]%攻击力，[mastery]点元素精通',
            refine: {
                atkPct: step(12),
                mastery: step(48)
            }
        },
        苇海信标: [{
                title: '元素战技命中敌人并受伤害后提升攻击力[atkPct]%',
                refine: {
                    atkPct: step(40)
                }
            }, {
                title: '不处于护盾情况下提升生命值[hpPct]%',
                refine: {
                    hpPct: step(32)
                }
            }],
        浪影阔剑: {
            title: '受到治疗时，攻击力提升[atkPct]%',
            refine: {
                atkPct: step(24)
            }
        },
        聊聊棒: [{
                title: '承受火元素附着，攻击力提升[atkPct]%',
                refine: {
                    atkPct: step(16)
                }
            }, {
                title: '承受水元素、冰元素或雷元素，元素伤害加成提升[dmg]%',
                refine: {
                    dmg: step(12)
                }
            }],
        便携动力锯: [{
                title: '满层时，元素精通提升[mastery]点',
                refine: {
                    mastery: step(40 * 3)
                }
            }],
        裁断: [staticStep('atkPct', 20), {
                title: '满层时，元素战技造成的伤害提升[eDmg]%',
                refine: {
                    eDmg: step(18 * 2)
                }
            }],
        '「究极霸王超级魔剑」': {
            title: '攻击力提升[atkPct]%',
            refine: {
                atkPct: step(12 * 2)
            }
        },
        山王长牙: [{
                title: '满层时，元素战技与元素爆发伤害提升[eDmg]%',
                refine: {
                    eDmg: step(10 * 6),
                    qDmg: step(10 * 6)
                }
            }],
        撼地者: {
            check: ({ element }) => ['火'].includes(element),
            title: '触发火元素相关反应，元素战技造成的伤害提升[eDmg]%',
            refine: {
                eDmg: step(16)
            }
        },
        硕果钩: {
            title: '下落攻击的暴击率提升[a3Cpct]%普通攻击、重击、下落攻击造成的伤害提升[aDmg]%',
            refine: {
                a3Cpct: step(16),
                aDmg: step(16),
                a2Dmg: step(16),
                a3Dmg: step(16)
            }
        },
        焚曜千阳: {
            title: '施放元素战技或元素爆发时，暴击伤害提高[cdmg]%，攻击力提升[atkPct]%',
            data: {
                cdmg: ({ params, refine }) => params.Nightsoul === true ? (step(20)[refine] * 1.75) : step(20)[refine],
                atkPct: ({ params, refine }) => params.Nightsoul === true ? (step(28)[refine] * 1.75) : step(28)[refine]
            }
        },
        拾慧铸熔: {
            check: ({ element }) => ['风', '水', '雷', '草'].includes(element),
            title: '触发感电、月感电或绽放反应时，元素精通提升[mastery]',
            refine: {
                mastery: step(60)
            }
        },
        万能钥匙: {
            title: '触发元素反应后元素精通提升[mastery]',
            data: {
                mastery: ({ params, refine }) => step(60)[refine] * ((params.Moonsign || 0) >= 2 ? 2 : 1)
            }
        },
        狼的武功歌: {
            title: '满层「四风诗系」造成的伤害提升[dmg]%。队伍拥有「魔导·秘仪」效果时，暴击伤害提高[cdmg]%',
            data: {
                dmg: ({ refine }) => step(7.5, 2)[refine] * 4,
                cdmg: ({ params, refine }) => (params.Hexenzirkel ? step(7.5, 2)[refine] * 4 : 0)
            }
        },
        超越之匙: [staticStep('atkPct', 28), {
                title: '装备者的重击每次命中敌人后，都会短暂达成「超越」。该效果至多叠加3层满层时，使装备者的星超导反应伤害提升[stellarConduct]%',
                refine: {
                    stellarConduct: step(16 * 3)
                }
            }],
        金律铸影: [{
                title: '「谐律乐章」之一————元素精通提高[mastery]点',
                refine: {
                    mastery: step(120)
                }
            }, {
                check: ({ element }) => ['雷', '风', '冰'].includes(element),
                title: '触发星烁反应后，元素精通提高[mastery]点',
                refine: {
                    mastery: step(120)
                }
            }],
        救赎之斩: [{
                title: '触发元素反应后，元素精通提高[mastery]点',
                refine: {
                    mastery: step(64)
                }
            }, {
                check: ({ element }) => ['雷', '风', '冰'].includes(element),
                title: '触发星烁反应后，攻击力提高[atkPct]%',
                refine: {
                    atkPct: step(16)
                }
            }]
    };
}

return exports;})().default,"gs"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(step, staticStep) {
    return {
        白缨枪: {
            title: '白缨枪普通攻击伤害提高[aDmg]%',
            refine: { aDmg: step(24) }
        },
        黑岩刺枪: {
            title: '黑岩刺枪满层攻击力加成[atkPct]%',
            data: {
                atkPct: ({ refine }) => step(12)[refine] * 3
            }
        },
        决斗之枪: {
            title: '身边敌人少于2个时，获得[atkPct]%的攻击力提升',
            refine: {
                atkPct: step(24)
            }
        },
        匣里灭辰: {
            check: ({ element }) => ['水', '火'].includes(element),
            title: '对于水或火元素影响的敌人造成伤害提高[dmg]%',
            refine: {
                dmg: step(20, 4),
                phy: step(20, 4)
            }
        },
        千岩长枪: {
            title: '四璃月队伍提高[atkPct]%攻击力及[cpct]%的暴击率提高',
            buffCount: 4,
            refine: {
                atkPct: [7, 8, 9, 10, 11],
                cpct: [3, 4, 5, 6, 7]
            }
        },
        试作星镰: {
            title: '释放元素战技后，2层Buff普攻与重击造成伤害提高[aDmg]%',
            buffCount: 2,
            refine: {
                aDmg: step(8),
                a2Dmg: step(8)
            }
        },
        宗室猎枪: {
            title: '3层Buff暴击提高[cpct]%',
            buffCount: 3,
            refine: {
                cpct: step(8)
            }
        },
        喜多院十文字: {
            title: '元素战技伤害提升[eDmg]%',
            refine: {
                eDmg: step(6)
            }
        },
        '「渔获」': {
            title: '元素爆发造成伤害提高[qDmg]%，元素爆发的暴击率提高[qCpct]%',
            refine: {
                qDmg: step(16),
                qCpct: step(6)
            }
        },
        断浪长鳍: {
            title: '满层元素爆发伤害提高[qDmg]%',
            refine: { qDmg: step(40) }
        },
        贯虹之槊: [staticStep('shield', 20), {
                title: '护盾满层状态提高攻击力[atkPct]%',
                buffCount: 10,
                refine: {
                    atkPct: step(4)
                }
            }],
        和璞鸢: {
            title: '满层攻击力提高[atkPct]%，伤害提升[dmg]%',
            refine: {
                atkPct: [3.2 * 7, 3.9 * 7, 4.6 * 7, 5.3 * 7, 6 * 7],
                dmg: step(12),
                phy: step(12)
            }
        },
        护摩之杖: [staticStep('hpPct', 20), {
                title: '基于生命值上限，获得[atkPlus]的攻击力提升',
                sort: 9,
                data: {
                    atkPlus: ({ attr, refine, calc }) => {
                        let totalHp = calc(attr.hp);
                        return totalHp * ([0.8, 1, 1.2, 1.4, 1.6][refine]) / 100;
                    }
                }
            }, {
                check: ({ attr }) => attr.characterName === '胡桃',
                title: '角色生命低于50%时额外获得[atkPlus]攻击力（默认仅胡桃触发）',
                sort: 9,
                data: {
                    atkPlus: ({ attr, refine, calc }) => {
                        let totalHp = calc(attr.hp);
                        return totalHp * ([1, 1.2, 1.4, 1.6, 1.8][refine]) / 100;
                    }
                }
            }],
        天空之脊: staticStep('cpct', 8),
        薙草之稻光: [{
                title: '元素爆发12秒内元素充能提高[rechargePlus]%',
                refine: {
                    rechargePlus: [30, 35, 40, 45, 50]
                }
            }, {
                title: '基于元素充能提升攻击力[atkPct]%',
                sort: 4,
                data: {
                    atkPct: ({ attr, refine }) => {
                        let recharge = attr.recharge.base + attr.recharge.plus - 100;
                        return Math.min(recharge * step(28)[refine] / 100, [80, 90, 100, 110, 120][refine]);
                    }
                }
            }],
        息灾: [staticStep('dmg', 12), {
                title: '满Buff提供[atkPct]%攻击力加成',
                data: {
                    atkPct: ({ refine, params }) => step(3.2 * 6)[refine] * (params.off_field === true ? 2 : 1)
                }
            }],
        贯月矢: {
            title: '拾取苏生之叶的角色攻击力提升[atkPct]%',
            refine: {
                atkPct: step(16)
            }
        },
        赤沙之杖: {
            title: '赤沙之杖被动：基于元素精通获得攻击力[_atk1]，3层Buff提高攻击力[_atk2]',
            sort: 6,
            data: {
                _atk1: ({ attr, calc, refine }) => step(52)[refine] * calc(attr.mastery) / 100,
                _atk2: ({ attr, calc, refine }) => step(28 * 3)[refine] * calc(attr.mastery) / 100,
                atkPlus: ({ attr, calc, refine }) => step(52 + 28 * 3)[refine] * calc(attr.mastery) / 100
            }
        },
        赤月之形: {
            title: '生命之契大于等于生命上限30%，造成的伤害提升[dmg]%',
            refine: {
                dmg: step(36, 12)
            }
        },
        风信之锋: {
            title: '触发元素反应提升攻击力[atkPct]%, 精通[mastery]',
            refine: {
                atkPct: step(12),
                mastery: step(48)
            }
        },
        峡湾长歌: {
            title: '队伍中存在至少三种不同元素类型的角色时，元素精通提升[mastery]点',
            refine: {
                mastery: step(120)
            }
        },
        勘探钻机: {
            title: '满层下，提高[atkPct]%攻击力与[dmg]%所有元素伤害加成',
            refine: {
                atkPct: [3 * 3, 4 * 3, 5 * 3, 6 * 3, 7 * 3],
                dmg: [7 * 3, 8.5 * 3, 10 * 3, 11.5 * 3, 13 * 3]
            }
        },
        柔灯挽歌: [staticStep('atkPct', 15), {
                title: '对处于燃烧状态的敌人造成伤害提升[dmg]%',
                data: {
                    dmg: ({ refine }) => step(18, 5)[refine] * 2
                }
            }],
        公义的酬报: false,
        虹的行迹: {
            title: '施放元素战技时，防御力提升[defPct]%',
            refine: {
                defPct: step(16)
            }
        },
        镇山之钉: {
            title: '元素战技造成的伤害提升[eDmg]%',
            refine: {
                eDmg: step(12 * 2)
            }
        },
        且住亭御咄: {
            title: '释放元素战技时，攻击力提升[atkPct]%',
            refine: {
                atkPct: step(20)
            }
        },
        香韵奏者: [staticStep('atkPct', 12), {
                title: '处于后台时进行治疗，攻击力提升[atkPct]%',
                refine: {
                    atkPct: step(12 + 32)
                }
            }],
        支离轮光: {
            title: '施放元素战技或元素爆发后攻击力提升[atkPct]%，创造护盾后月感电伤害提升[lunarCharged]%',
            refine: {
                atkPct: step(24),
                lunarCharged: step(40)
            }
        },
        掘金之锹: {
            title: '感电反应造成的伤害提升[electroCharged]%，月感电反应造成的伤害提升[lunarCharged]%',
            data: {
                electroCharged: ({ refine }) => step(48)[refine],
                lunarCharged: ({ params, refine }) => step(12)[refine] * ((params.Moonsign || 0) >= 2 ? 2 : 1)
            }
        },
        血染荒城: {
            title: '施放元素爆发后月感电反应伤害提高[lunarCharged]%,触发月感电反应后暴击伤害提高[cdmg]%',
            data: {
                lunarCharged: ({ refine }) => step(36)[refine],
                cdmg: ({ element, refine }) => ['水', '雷', '风'].includes(element) ? step(28)[refine] : 0
            }
        },
        圣祭者的辉杖: {
            title: '攻击力提升[atkPct]%，元素充能效率提升[recharge]%',
            refine: {
                atkPct: step(8 * 3),
                recharge: step(12 * 3)
            }
        },
        灾悔: {
            title: '装备者施放元素战技后普攻,重击,元素战技与元素爆发造成的伤害提升[aDmg]%',
            data: {
                aDmg: ({ params, refine }) => (params.Hexenzirkel ? 1.75 : 1) * step(40)[refine],
                a2Dmg: ({ params, refine }) => (params.Hexenzirkel ? 1.75 : 1) * step(40)[refine],
                eDmg: ({ params, refine }) => (params.Hexenzirkel ? 1.75 : 1) * step(40)[refine],
                qDmg: ({ params, refine }) => (params.Hexenzirkel ? 1.75 : 1) * step(40)[refine]
            }
        },
        寒息: {
            title: '触发冰元素或水元素相关反应后，攻击力提升[atkPct]%',
            refine: {
                atkPct: step(20)
            }
        },
        戍望谣歌: {
            check: ({ element }) => ['雷', '风', '冰'].includes(element),
            title: '触发星烁反应后，攻击力提升[atkPct]%',
            refine: {
                atkPct: step(20)
            }
        }
    };
}

return exports;})().default,"gs"));
Object.assign(weaponBuffs,loadWeaponDefinitions((()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
function default_1(step, staticStep) {
    return {
        辰砂之纺锤: {
            title: '元素战技造成的伤害值提高[ePlus]',
            sort: 9,
            data: {
                ePlus: ({ attr, refine }) => attr.def * step(40)[refine] / 100
            }
        },
        腐殖之剑: {
            title: '元素战技的伤害增加[eDmg]%，暴击率提高[eCpct]%',
            refine: {
                eDmg: step(16),
                eCpct: step(6)
            }
        },
        降临之剑: {
            check: ({ characterName }) => ['空', '荧', '旅行者'].includes(characterName),
            title: '攻击力提高[atkPlus]点',
            data: {
                atkPlus: 66
            }
        },
        黑剑: {
            title: '普攻与重击的造成的伤害提升[aDmg]%',
            refine: {
                aDmg: step(20),
                a2Dmg: step(20)
            }
        },
        暗巷闪光: {
            title: '角色造成的伤害提升[dmg]%',
            refine: {
                dmg: step(12),
                phy: step(12)
            }
        },
        宗室长剑: {
            title: '3层Buff下，暴击率提高[cpct]%',
            buffCount: 3,
            refine: {
                cpct: step(8)
            }
        },
        试作斩岩: {
            title: '满Buff提高攻击力及防御力[atkPct]%',
            buffCount: 4,
            refine: {
                atkPct: step(4),
                defPct: step(4)
            }
        },
        匣里龙吟: {
            check: ({ element }) => ['火', '雷'].includes(element),
            title: '对处于火元素或雷元素影响下的敌人，造成的伤害提高[dmg]%',
            refine: {
                dmg: step(20, 4),
                phy: step(20, 4)
            }
        },
        铁蜂刺: {
            title: '满Buff伤害提高[dmg]%',
            buffCount: 2,
            refine: {
                dmg: step(6),
                phy: step(6)
            }
        },
        黑岩长剑: {
            title: '满Buff攻击力提高[atkPct]%',
            buffCount: 3,
            refine: {
                atkPct: step(12)
            }
        },
        飞天御剑: {
            title: '施放元素爆发后，提高[atkPct]%的攻击力',
            refine: {
                atkPct: step(12)
            }
        },
        黎明神剑: {
            title: '生命值高于90%时，暴击率提升[cpct]%',
            refine: {
                cpct: step(14)
            }
        },
        暗铁剑: {
            check: ({ element }) => element === '雷',
            title: '触发雷元素相关反应后攻击力提高[atkPct]%',
            refine: {
                atkPct: step(20)
            }
        },
        冷刃: {
            check: ({ element }) => ['水', '冰'].includes(element),
            title: '对处于水或冰元素影响的敌人伤害提高[dmg]%',
            refine: {
                dmg: step(12),
                phy: step(12)
            }
        },
        笼钓瓶一心: {
            title: '触发效果时攻击力提升[atkPct]%',
            refine: {
                atkPct: step(15)
            }
        },
        西福斯的月光: {
            title: '基于元素精通，提升[recharge]%元素充能效率',
            sort: 6,
            data: {
                recharge: ({ attr, calc, refine }) => calc(attr.mastery) * step(0.036)[refine]
            }
        },
        波乱月白经津: [staticStep('dmg', 12), {
                title: '满层提高普攻[aDmg]%',
                buffCount: 2,
                refine: {
                    aDmg: step(20)
                }
            }],
        雾切之回光: [staticStep('dmg', 12), {
                title: '满层获得伤害加成[dmg]%',
                refine: {
                    dmg: step(28)
                }
            }],
        苍古自由之誓: [{
                title: '造成的伤害提高[dmg]%',
                refine: {
                    dmg: step(10)
                }
            }, {
                title: '触发Buff后提高普攻重击与下落攻击[aDmg]%，攻击力提升[atkPct]%',
                refine: {
                    aDmg: step(16),
                    a2Dmg: step(16),
                    a3Dmg: step(16),
                    atkPct: step(20)
                }
            }],
        磐岩结绿: [staticStep('hpPct', 20), {
                title: '基于生命值上限提高攻击力[atkPlus]',
                sort: 9,
                data: {
                    atkPlus: ({ attr, calc, refine }) => calc(attr.hp) * step(1.2)[refine] / 100
                }
            }],
        裁叶萃光: [staticStep('cpct', 4), {
                title: '普攻与元素战技造成的伤害值提高[aPlus]',
                sort: 9,
                data: {
                    aPlus: ({ attr, calc, refine }) => calc(attr.mastery) * step(120)[refine] / 100,
                    ePlus: ({ attr, calc, refine }) => calc(attr.mastery) * step(120)[refine] / 100
                }
            }],
        斫峰之刃: [staticStep('shield', 20), {
                title: '满Buff提高攻击力[atkPct]%',
                buffCount: 10,
                refine: {
                    atkPct: step(4)
                }
            }],
        天空之刃: [staticStep('cpct', 4), {
                title: '暴击提高[_cpct]%',
                refine: {
                    _cpct: step(4)
                }
            }],
        风鹰剑: [staticStep('atkPct', 20), {
                title: '攻击力提高[_atkPct]%',
                refine: {
                    _atkPct: step(20)
                }
            }],
        原木刀: {
            title: '拾取种识之叶的角色元素精通提升[mastery]',
            refine: {
                mastery: step(60)
            }
        },
        圣显之钥: [staticStep('hpPct', 20), {
                title: '基于生命提升元素精通，满层提升[mastery]',
                sort: 5,
                data: {
                    mastery: ({ attr, calc, refine }) => step(0.36 + 0.2)[refine] * calc(attr.hp) / 100
                }
            }],
        灰河渡手: {
            title: '元素战技暴击率提升[eCpct]%；此外，施放元素战技后的5秒内，元素充能效率提升[rechargePlus]%',
            refine: {
                eCpct: step(8),
                rechargePlus: [16, 20, 24, 28, 32]
            }
        },
        海渊终曲: {
            title: '释放元素战技攻击力提升[atkPct]%，生命之契提升[atkPlus]点攻击力',
            sort: 9,
            data: {
                atkPlus: ({ attr, calc, refine }) => Math.min(Math.floor(calc(attr.hp) * 0.25 * step(0.24)[refine] / 10, step(150)))
            },
            refine: {
                atkPct: step(12)
            }
        },
        船坞长剑: {
            title: '满层提高[mastery]点元素精通',
            refine: {
                mastery: step(40 * 3)
            }
        },
        狼牙: [{
                title: '元素战技与元素爆发造成的伤害提升[eDmg]%',
                refine: {
                    eDmg: step(16),
                    qDmg: step(16)
                }
            }, {
                title: '满层下，元素战技与元素爆发命中敌人，其暴击率提升[eCpct]%',
                refine: {
                    eCpct: step(8),
                    qCpct: step(8)
                }
            }],
        静水流涌之辉: [{
                title: '生命值变化时，3层Buff战技伤害提高[eDmg]%',
                refine: {
                    eDmg: step(8 * 3)
                }
            }, {
                title: '其他角色生命值变化时，2层Buff提高生命上限[hpPct]%',
                refine: {
                    hpPct: step(14 * 2)
                }
            }],
        有乐御簾切: [staticStep('defPct', 20), {
                title: '附近的角色在场上造成岩元素伤害后，普通攻击伤害提升[aDmg]%，元素战伤害提升[eDmg]%；',
                refine: {
                    aDmg: step(16 * 2),
                    eDmg: step(24 * 2)
                }
            }],
        赦罪: [staticStep('cdmg', 20), {
                title: '生命之契的数值增加时，装备者造成的伤害提升[dmg]%',
                refine: {
                    dmg: step(16 * 3)
                }
            }],
        息燧之笛: {
            title: "施放元素战技时，防御力提升[defPct]%",
            refine: {
                defPct: step(16)
            }
        },
        弥坚骨: {
            title: "冲刺或替代冲刺的能力后，普通攻击造成的伤害提高[aPlus]%",
            data: {
                aPlus: ({ attr, calc, refine }) => calc(attr.atk) * step(16)[refine] / 100
            }
        },
        岩峰巡歌: [{
                title: '2层buff使防御力提高[defPct]%所有元素伤害加成提高[dmg]%',
                refine: {
                    defPct: step(8 * 2),
                    dmg: step(10 * 2)
                }
            }, {
                title: '基于防御力,使队伍中附近所有角色的所有元素伤害加成提高[dmg]%',
                sort: 9,
                data: {
                    dmg: ({ attr, calc, refine }) => Math.min(calc(attr.def) / 1000 * step(8)[refine], step(25.6)[refine])
                }
            }],
        厄水之祸: {
            title: '处于护盾庇护下,普攻和重击造成伤害提升[aDmg]%暴击率提升[aCpct]%',
            refine: {
                aDmg: step(20),
                a2Dmg: step(20),
                aCpct: step(8),
                a2Cpct: step(8)
            }
        },
        苍耀: {
            title: '元素能量为0时攻击力提升[atkPct]%,暴击伤害提升[cdmg]%',
            refine: {
                atkPct: step(48),
                cdmg: step(40)
            }
        },
        谧音吹哨: {
            title: '触发反应后生命值上限提高[hpPct]%',
            data: {
                hpPct: ({ params, refine }) => step(16)[refine] * ((params.Moonsign || 0) >= 2 ? 2 : 1)
            }
        },
        织月者的曙色: {
            title: '元素爆发造成的伤害提高[qDmg]%',
            data: {
                qDmg: ({ talent, refine }) => step(20)[refine] + (talent.q['元素能量'] > 60 ? 0 : step(16)[refine]) + (talent.q['元素能量'] > 40 ? 0 : step(12)[refine])
            }
        },
        黑蚀: {
            title: '元素爆发造成的暴击伤害提升[qCdmg]%，攻击力提升[atkPct]%',
            data: {
                qCdmg: ({ refine }) => step(16)[refine],
                atkPct: ({ params, refine }) => step(20)[refine] * (params.Hexenzirkel ? 1.75 : 1)
            }
        },
        朏魄含光: [staticStep('defPct', 20), {
                title: '装备者施放元素战技后的5秒内，月结晶反应伤害提升[lunarCrystallize]%',
                refine: {
                    lunarCrystallize: step(64)
                }
            }],
        熔猎异端之刃: {
            title: '施放元素战技后，每秒都将基于上一秒记录的移动距离，获得最高[atkPct]%的攻击力加成',
            data: {
                atkPct: step(36)
            }
        },
        引火之源: [{
                title: '触发元素反应后，装备者的攻击力提升[atkPct]%',
                refine: {
                    atkPct: step(16)
                }
            }, {
                check: ({ element }) => ['冰', '雷', '风'].includes(element),
                title: '触发星烁反应后，星烁反应伤害提升[stellarConduct]%',
                refine: {
                    stellarConduct: step(16),
                    stellarSwirl: step(16),
                    stellarVortex: step(16)
                }
            }],
        白湖冬羽: [{
                title: '元素战技命中敌人时，攻击力提升[atkPct]%',
                refine: {
                    atkPct: step(8)
                }
            }, {
                check: ({ element }) => ['冰', '雷', '风'].includes(element),
                title: '满层时，星烁反应造成的暴击伤害提升[stellarConduct]%',
                refine: {
                    cdmg: step(50, 15)
                }
            }],
        星锋剑: {
            check: ({ attr }) => ['空', '荧'].includes(attr.characterName),
            title: '旅行者装备时，命中敌人后，攻击力提升[atkPct]%；与7种元素共鸣过，旅行者的暴击伤害就会总共提升[cdmg]%',
            refine: {
                atkPct: step(16),
                cdmg: 42
            }
        }
    };
}

return exports;})().default,"gs"));
