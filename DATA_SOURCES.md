# 资料来源

本插件的固定资料由 `game-plugin-kit/scripts/import-reference-data.py` 从已下载参考 JSON 转换，源提交记录在 `internal/assets/catalog.json`。转换保留角色、装备名称、属性、技能文字与材料；不执行上游脚本，不包含完整伤害计算或全部素材。

原神与星铁资料来自 miao-plugin，固定提交 7f6f1c84c89102bc6b1c58c8e1b06c61f4642161；保留 LICENSES/miao-plugin-MIT.txt。

公开展柜来自 [Enka.Network](https://github.com/EnkaNetwork/API-docs/blob/master/api.md)，按响应 TTL 缓存。抽卡导入导出遵循 [UIGF](https://uigf.org/en/standards/uigf.html)，尚未申请兼容性认证。

本插件及编译期业务库沿用 RayleaBot SDK 的 AGPL-3.0 许可，安装包管理页提供对应源码下载。上游数据的原许可声明另行保留。

## 装备评分

评分核心按固定参考原生实现：miao `ArtisMark`/`ArtisMarkCfg` 与 ZZZ `Score`。转换脚本 `game-plugin-kit/scripts/import-score-data.py` 只读取静态数字对象与 JSON，不运行原插件。对应版权许可保留在 `LICENSES/`，评分结果带规则版本。

原神、星铁采用 miao 提交 `7f6f1c84c89102bc6b1c58c8e1b06c61f4642161` 的基础权重；角色专属 `artis.js` 已由受限语法转换器编译为 Go，并与原始纯函数进行固定输入对照。绝区零采用 dev 提交 `fb66219cec0294e1834bacdf0033b2d43a9ccaf4` 的十个显式预设；动态模式已原生适配三份角色覆盖函数和条件预设选择，不在插件运行期执行 JS。评分不等于伤害计算或队伍收益。

## 自动参考计算

`internal/assets/calc/` 保存本游戏的角色脚本（`characters/<ID>-<角色名>.js`）、武器与圣遗物效果（`game.js`）和计算资料（`catalog.json`），由相邻库的 `scripts/bundle-reference-calculation.mjs` 从 miao 固定快照生成；共用的 miao 运行时在相邻库 `reference/miao/`，打包修正见其中的 `PATCHES.md`。数值对照向量在 `internal/assets/testdata/calc-vectors.json`。

## 模拟与固定资料

材料、卡池与固定生日从同一 miao 固定快照的 material/info 数据转换，不联网补造日程。娱乐模拟的默认曲线来自 [Miao-Yunzai 固定提交](https://github.com/yoimiya-kokomi/Miao-Yunzai/tree/40cc2103efba1fbb279b768e3f5345d45372d357) 和 [StarRail-plugin 固定提交](https://github.com/TsukinaKasumi/StarRail-plugin/tree/090e411cf9be28b1644721fab54eab0ad283668f)，转换为 Go，不在运行时加载原插件。只向原神和星铁开放模拟入口；不声明为当前官方概率。

GPL-3.0 和 Apache-2.0 许可分别保留在 `LICENSES/Miao-Yunzai-GPL-3.0.txt` 与 `LICENSES/StarRail-plugin-Apache-2.0.txt`，对应的转换器、数据和原生实现随源码分发。转换模拟数据需要已有 Python/PyYAML，正常构建使用已生成数据，无需该再生成步骤。

云面板装备解释沿用同一固定 miao 快照的 artifact 元数据与 ArtisAttr 公式，包含原神 299、星铁 768 项装备映射。转换脚本为相邻业务库 `scripts/import-cloud-panel-data.mjs`，不加载上游机器人运行时；缺失档位明确显示未覆盖。


绝区零卡池历史使用 [GachaClock 固定提交](https://github.com/iaoongin/GachaClock/tree/99d16c10bfeeb5f885e9cb42861c50f993f3a746) 的 JSON 数据，保留 `LICENSES/GachaClock-MIT.txt`。110 条收录中 54 个起点按参考规则推算并标记；当前快照末期结束日为 2026-05-05，不能代表最新官方排期。仅转换文字和日期，没有下载或分发关联图片。原神/星铁仍使用 miao 同一固定快照的数据。


原魔属性使用 [Atlas 固定提交](https://github.com/Nwflower/atlas/tree/016e49357666e0823791abdf28fbb3b2efe68225) 的文字数值资料：556 项条目、200 级曲线、102 项修饰因子，仅在原神界面开放。保留 `LICENSES/Atlas-GPL-3.0.txt`，转换器为编译期库的 `scripts/import-enemy-data.py`；未引入其外部图鉴图片仓库。攻略合集编号来自上述固定 Yunzai/StarRail/ZZZ 参考，运行时匿名读取官方文字与原图链接，不打包攻略图。
