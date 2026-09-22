# 资料来源

本插件的固定资料由 `game-plugin-kit/scripts/import-reference-data.py` 从已下载参考 JSON 转换，源提交记录在 `internal/assets/catalog.json`。转换保留角色、装备名称、属性、技能文字与材料，不包含素材图片；伤害与评分由打包的上游脚本计算，见下文。

原神与星铁资料来自 miao-plugin，固定提交 7f6f1c84c89102bc6b1c58c8e1b06c61f4642161；保留 LICENSES/miao-plugin-MIT.txt。

没有账号时的面板来自 [Enka.Network](https://github.com/EnkaNetwork/API-docs/blob/master/api.md)（miao-plugin 海外服的默认面板服务，国服默认的 MiniGG 为 HTTP 地址，不使用），按响应 TTL 缓存；解析同 miao 的 EnkaData，圣遗物部件名由 bundle-reference-calculation.mjs 从 miao `meta-gs/artifact/data.json` 写入资料目录的 `artifact_pieces`。`templates/panel-list/`、`templates/artifact-list/` 与 `templates/daily-material/` 按 miao-plugin 的 `character/profile-list`、`character/artis-list` 与 `wiki/today-material` 改写（MIT；今日素材的材料周期同 `meta-gs/material/daily.js`，材料各级图标运行时读取下载的 `meta-gs/material/data.json`，页尾提示中上游的“刷新天赋”换成本插件对应的“米游社更新面板”）。抽卡导入导出遵循 [UIGF](https://uigf.org/en/standards/uigf.html)，尚未申请兼容性认证。

本插件及编译期业务库沿用 RayleaBot SDK 的 AGPL-3.0 许可，安装包管理页提供对应源码下载。上游数据的原许可声明另行保留。

## 装备评分

评分运行 miao 提交 `7f6f1c84c89102bc6b1c58c8e1b06c61f4642161` 的 `ArtisMarkCfg`、`ArtisMark` 与角色专属 `artis.js`，规则按当前属性、命座、武器与套装自动选择，与上游一致。`internal/assets/calc/scores/<ID>-<角色名>.js` 由相邻库的 `scripts/bundle-reference-calculation.mjs` 从 `artis.js` 生成；默认权重（`artis-mark.js`）、主副词条表与套装简称随 `game.js` 打包。回归向量在 `internal/assets/testdata/score-vectors.json`。评分不等于伤害计算或队伍收益。

## 自动参考计算

`internal/assets/calc/` 保存本游戏的角色脚本（`characters/<ID>-<角色名>.js`）、武器与圣遗物效果（`game.js`）和计算资料（`catalog.json`），由相邻库的 `scripts/bundle-reference-calculation.mjs` 从 miao 固定快照生成；共用的 miao 运行时在相邻库 `reference/miao/`，打包修正见其中的 `PATCHES.md`。数值对照向量在 `internal/assets/testdata/calc-vectors.json`。

## 模拟与固定资料

材料、卡池与固定生日从同一 miao 固定快照的 material/info 数据转换，不联网补造日程。娱乐模拟的默认曲线来自 [Miao-Yunzai 固定提交](https://github.com/yoimiya-kokomi/Miao-Yunzai/tree/40cc2103efba1fbb279b768e3f5345d45372d357) 和 [StarRail-plugin 固定提交](https://github.com/TsukinaKasumi/StarRail-plugin/tree/090e411cf9be28b1644721fab54eab0ad283668f)，转换为 Go，不在运行时加载原插件。只向原神和星铁开放模拟入口；不声明为当前官方概率。

GPL-3.0 和 Apache-2.0 许可分别保留在 `LICENSES/Miao-Yunzai-GPL-3.0.txt` 与 `LICENSES/StarRail-plugin-Apache-2.0.txt`，对应的转换器、数据和原生实现随源码分发。转换模拟数据需要已有 Python/PyYAML，正常构建使用已生成数据，无需该再生成步骤。

云面板装备解释沿用同一固定 miao 快照的 artifact 元数据与 ArtisAttr 公式，包含原神 299、星铁 768 项装备映射。转换脚本为相邻业务库 `scripts/import-cloud-panel-data.mjs`，不加载上游机器人运行时；缺失档位明确显示未覆盖。

原魔属性使用 [Atlas 固定提交](https://github.com/Nwflower/atlas/tree/016e49357666e0823791abdf28fbb3b2efe68225) 的文字数值资料：556 项条目、200 级曲线、102 项修饰因子，仅在原神界面开放。保留 `LICENSES/Atlas-GPL-3.0.txt`，转换器为编译期库的 `scripts/import-enemy-data.py`；未引入其外部图鉴图片仓库。攻略合集编号来自上述固定 Yunzai/StarRail/ZZZ 参考，运行时匿名读取官方文字与原图链接，不打包攻略图。

## 图片模板

`templates/note/`、`templates/gacha/`、`templates/abyss/`、`templates/abyss-floor/`、`templates/combat/`、`templates/ledger/`、`templates/ledger-count/`、`templates/tcg-decks/`、`templates/role-card/` 与 `templates/role-explore/` 按 Miao-Yunzai 原神插件的 daily-note-gs、gacha-log、gacha-all-log（`templates/gacha-all/`；上游各卡池块共用第一个卡池的标签颜色、五星时间段与 UP 标记判断，这里照做）、html/abyss/abyss、html/abyss/abyss-floor、html/abyss/combat、html/ledger/ledger-gs、html/ledger/ledger-count-gs 与 html/deckList、html/deck（两页共用 deck 的样式，deckList 未用到其余部分）、html/player/role-card 与 html/player/role-explore 改写（探索页的各项总数取自同一快照的 `defSet/role/index.yaml`；角色卡片按接口的元素给角色底色，上游按 `defSet/element/role.yaml` 的角色名查找）（札记与原石统计的图表上游用 G2Plot 绘制，这里按相同的内边距、刻度算法、起点、方向、内外半径与标签规则输出 SVG，不随插件分发该库；原石统计页头的闲云横幅取自 miao-plugin）（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），抽卡分析的统计、卡池选择与 UP 判定同上游 gachaLog；`templates/calendar/` 与 `templates/calendar-list/` 按 miao-plugin 的 `wiki/calendar` 改写（MIT；列表模式在上游缩窄页面，这里以同一页面的第二份模板取其宽度），公告时间、深渊与剧诗周期、合并排布与今日天赋书的算法同上游 `apps/wiki/Calendar.js`，天赋书的城市与周期取自同一快照的 `meta-gs/material/daily.js`；上游另从 miao 自有的 HTTP 服务读取修正后的活动时间，本插件的公开资料只访问官方 HTTPS 接口，因此不含这部分修正。`templates/character-talent/` 按 miao-plugin 的 `wiki/character-talent` 与 `common/tpl/talent-detail` 改写（MIT），天赋、被动与命之座资料运行时读取素材更新下载的 miao `meta-gs/character/<角色>/data.json`（为此 miao 素材来源也保留 `.json` 文件），上游原样输出的描述 HTML 只保留标题、换行、斜体、加粗与颜色。`templates/character-wiki/` 按 miao-plugin 的 `wiki/character-wiki` 改写（MIT），资料读取同一 `data.json` 与 `meta-gs/material/data.json`，材料简称由 import-aliases.py 从 `meta-gs/material/abbr.js` 写入资料目录；平均等级、命座分布与武器、圣遗物使用率与上游一样运行时读取 api.lelaer.com，持有率读取 api.yshelper.com（均为第三方只读公开统计，请求不带账号信息，结果缓存一小时），七圣召唤卡牌页 `templates/tcg-cards/` 按 html/deckCard 改写。`templates/panel/` 的样式由 miao-plugin 固定提交的 `common/common.css` 与 `character/profile-detail.css` 转换而来，`templates/hard-challenge/`、`templates/characters/`、`templates/training/` 与 `templates/help/` 按 miao 的 `stat/hard-summary`、`character/avatar-list`、`character/profile-stat`（练度统计与天赋统计两种模式，天赋书的城市与周期取自 `meta-gs/material/daily.js`）、`help/index`（默认主题）与通用角色卡片改写（均为 MIT，角色列表的宝箱上限取自同一固定快照的 `meta-gs/info` chestInfo，帮助图标按 `config/help_default.js` 中列出该命令的条目选取，上游未列出的命令不显示图标），图片地址改为宿主渲染资源。幽境危战的角色卡片取自实时的角色详情查询，因此去掉了上游“先更新面板”的提示。模板用到的图片与字体不随插件分发，由管理员通过“素材更新”在运行时从上游仓库下载。
