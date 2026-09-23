# 资料来源

本插件的固定资料由 `scripts/import-reference-data.py` 从已下载参考 JSON 转换，源提交记录在 `internal/assets/catalog.json`。转换保留角色、装备名称、属性、技能文字与材料，不包含素材图片；伤害与评分由打包的上游脚本计算，见下文。

原神资料来自 miao-plugin，固定提交 7f6f1c84c89102bc6b1c58c8e1b06c61f4642161；保留 LICENSES/miao-plugin-MIT.txt。

没有账号时的面板来自 [Enka.Network](https://github.com/EnkaNetwork/API-docs/blob/master/api.md)（miao-plugin 海外服的默认面板服务，国服默认的 MiniGG 为 HTTP 地址，不使用），按响应 TTL 缓存；解析同 miao 的 EnkaData，圣遗物部件名由 bundle-reference-calculation.mjs 从 miao `meta-gs/artifact/data.json` 写入资料目录的 `artifact_pieces`。`templates/panel-list/`、`templates/artifact-list/` 与 `templates/daily-material/` 按 miao-plugin 的 `character/profile-list`、`character/artis-list` 与 `wiki/today-material` 改写（MIT；今日素材的材料周期同 `meta-gs/material/daily.js`，材料各级图标按 miao 素材来源的 `meta-gs/material/data.json` 读取，页尾提示中上游的“刷新天赋”换成本插件对应的“米游社更新面板”）。抽卡导入导出遵循 [UIGF](https://uigf.org/en/standards/uigf.html)，尚未申请兼容性认证。

本插件沿用 RayleaBot SDK 的 AGPL-3.0 许可，安装包管理页提供对应源码下载。上游数据的原许可声明另行保留。

## 装备评分

评分运行 miao 提交 `7f6f1c84c89102bc6b1c58c8e1b06c61f4642161` 的 `ArtisMarkCfg`、`ArtisMark` 与角色专属 `artis.js`，规则按当前属性、命座、武器与套装自动选择，与上游一致。`internal/assets/calc/scores/<ID>-<角色名>.js` 由 `scripts/bundle-reference-calculation.mjs` 从 `artis.js` 生成；默认权重（`artis-mark.js`）、主副词条表与套装简称随 `game.js` 打包。回归向量在 `internal/assets/testdata/score-vectors.json`。评分不等于伤害计算或队伍收益。

## 自动参考计算

`internal/assets/calc/` 保存本游戏的角色脚本（`characters/<ID>-<角色名>.js`）、武器与圣遗物效果（`game.js`）和计算资料（`catalog.json`），由 `scripts/bundle-reference-calculation.mjs` 从 miao 固定快照生成；miao 运行时的适配在 `internal/reference/miao/`，打包修正见其中的 `PATCHES.md`。数值对照向量在 `internal/assets/testdata/calc-vectors.json`。

## 模拟与固定资料

材料、卡池与固定生日从同一 miao 固定快照的 material/info 数据转换，不联网补造日程。娱乐模拟的默认曲线来自 [Miao-Yunzai 固定提交](https://github.com/yoimiya-kokomi/Miao-Yunzai/tree/40cc2103efba1fbb279b768e3f5345d45372d357)，转换为 Go，不在运行时加载原插件；不声明为当前官方概率。

GPL-3.0 许可保留在 `LICENSES/Miao-Yunzai-GPL-3.0.txt`，对应的转换器、数据和原生实现随源码分发。转换模拟数据需要已有 Python/PyYAML，正常构建使用已生成数据，无需该再生成步骤。

云面板装备解释沿用同一固定 miao 快照的 artifact 元数据与 ArtisAttr 公式，包含 299 项圣遗物映射。转换脚本为 `scripts/import-cloud-panel-data.mjs`，不加载上游机器人运行时；缺失档位明确显示未覆盖。

原魔属性使用 [Atlas 固定提交](https://github.com/Nwflower/atlas/tree/016e49357666e0823791abdf28fbb3b2efe68225) 的文字数值资料：556 项条目、200 级曲线、102 项修饰因子。保留 `LICENSES/Atlas-GPL-3.0.txt`，转换器为 `scripts/import-enemy-data.py`；未引入其外部图鉴图片仓库。攻略合集编号来自上述固定 Yunzai 原神插件参考，运行时匿名读取官方文字与原图链接，不打包攻略图。

## 图片模板

`templates/note/`、`templates/gacha/`、`templates/abyss/`、`templates/abyss-floor/`、`templates/combat/`、`templates/ledger/`、`templates/ledger-count/`、`templates/tcg-decks/`、`templates/role-card/` 与 `templates/role-explore/` 按 Miao-Yunzai 原神插件的 daily-note-gs、gacha-log、gacha-all-log（`templates/gacha-all/`；上游各卡池块共用第一个卡池的标签颜色、五星时间段与 UP 标记判断，这里照做）、html/abyss/abyss、html/abyss/abyss-floor、html/abyss/combat、html/ledger/ledger-gs、html/ledger/ledger-count-gs 与 html/deckList、html/deck（两页共用 deck 的样式，deckList 未用到其余部分）、html/player/role-card 与 html/player/role-explore 改写（探索页的各项总数取自同一快照的 `defSet/role/index.yaml`；角色卡片按接口的元素给角色底色，上游按 `defSet/element/role.yaml` 的角色名查找）（札记与原石统计的图表上游用 G2Plot 绘制，这里按相同的内边距、刻度算法、起点、方向、内外半径与标签规则输出 SVG，不随插件分发该库；原石统计页头的闲云横幅取自 miao-plugin）（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），抽卡分析的统计、卡池选择与 UP 判定同上游 gachaLog；剧诗练度统计与 miao 默认数据源一样运行时读取 static.nanoka.cc 的 manifest 与 rolecombat 资料（只读，不带账号信息），月份换算、开幕角色补入、筛选与排序同上游 `apps/profile/ProfileStat.js`。`templates/character-card/` 按 miao-plugin 的 character/character-card 改写（MIT），横竖版的页面宽度与缩放同上游 `apps/character/AvatarCard.js`，触发写法同其 check。`templates/stat-character/`、`templates/stat-abyss-pct/` 与 `templates/stat-abyss-team/` 按 miao-plugin 的 stat/character、stat/abyss-pct 与 stat/abyss-team 改写（MIT），数据与上游 `apps/stat/HutaoApi.js` 一样读取 api.lelaer.com 与 api.yshelper.com 的公开统计（只读，不带账号信息，缓存一小时），排序、持有率换算与配队算法同 `apps/stat/AbyssStat.js`、`AbyssTeam.js`；页面沿用上游“数据来自提瓦特小助手API”的说明。`templates/pool-info/` 按 miao-plugin 的 gacha/gacha-info 改写（MIT），卡池的筛选、每行八个与精简显示同上游 `apps/gacha/GachaPool.js`；`templates/gacha-count/` 按同一上游的 html/gacha/log-count 改写（GPL-3.0），分池、计数与排序同 `model/logCount.js`；上游固定提交的 `defSet/pool` 只到 2025 年，活动祈愿的期次改用固定快照中的卡池资料，常驻、集录与新手沿用上游的单一卡池；`templates/gacha-detail/` 与 `templates/gacha-stat/` 按 miao-plugin 的 gacha/gacha-detail 与 gacha/gacha-stat 改写（MIT），出金、UP 判定与分期统计同上游 `apps/gacha/GachaData.js`，期次取自同一卡池资料，最后一期之后的“新版本”不设结束日期（上游写死的结束日期已过）；上游抽卡记录去掉的前缀不含“喵喵”，带“喵喵”的写法总是落到角色池，这里一并去掉；`templates/pay-log/` 按同一 Yunzai 插件的 html/payLog 改写（GPL-3.0），读取与档位统计同 `model/payLogData.js`（上游第一个月的计数会被该月后续记录清零，这里每条记录都计入），ECharts 的柱状图与饼图按其 5.x 默认样式输出为 SVG；五星简称取自 `defSet/role|weapon/other.yaml` 的 sortName（由 import-aliases.py 写入资料目录）。`templates/weapons/` 与 `templates/weapons-narrow/` 按同一上游的 html/avatar/weapon 改写（GPL-3.0；武器不超过 8 把时上游缩窄页面，这里以第二份模板取其宽度），排序规则与不计精炼的活动武器同上游 `model/weapon.js` 与 `defSet/weapon/other.yaml`；`templates/calendar/` 与 `templates/calendar-list/` 按 miao-plugin 的 `wiki/calendar` 改写（MIT；列表模式在上游缩窄页面，这里以同一页面的第二份模板取其宽度），公告时间、深渊与剧诗周期、合并排布与今日天赋书的算法同上游 `apps/wiki/Calendar.js`，天赋书的城市与周期取自同一快照的 `meta-gs/material/daily.js`；上游另从 miao 自有的 HTTP 服务读取修正后的活动时间，本插件的公开资料只访问官方 HTTPS 接口，因此不含这部分修正。`templates/character-talent/` 按 miao-plugin 的 `wiki/character-talent` 与 `common/tpl/talent-detail` 改写（MIT），天赋、被动与命之座资料读取 miao 素材来源的 `meta-gs/character/<角色>/data.json`（为此 miao 素材来源也保留 `.json` 文件），上游原样输出的描述 HTML 只保留标题、换行、斜体、加粗与颜色。`templates/character-wiki/` 按 miao-plugin 的 `wiki/character-wiki` 改写（MIT），资料读取同一 `data.json` 与 `meta-gs/material/data.json`，材料简称由 import-aliases.py 从 `meta-gs/material/abbr.js` 写入资料目录；平均等级、命座分布与武器、圣遗物使用率与上游一样运行时读取 api.lelaer.com，持有率读取 api.yshelper.com（均为第三方只读公开统计，请求不带账号信息，结果缓存一小时），七圣召唤卡牌页 `templates/tcg-cards/` 按 html/deckCard 改写。`templates/panel/` 的样式由 miao-plugin 固定提交的 `common/common.css` 与 `character/profile-detail.css` 转换而来，`templates/hard-challenge/`、`templates/characters/`、`templates/training/` 与 `templates/help/` 按 miao 的 `stat/hard-summary`、`character/avatar-list`、`character/profile-stat`（练度统计与天赋统计两种模式，天赋书的城市与周期取自 `meta-gs/material/daily.js`）、`help/index`（默认主题）与通用角色卡片改写（均为 MIT，角色列表的宝箱上限取自同一固定快照的 `meta-gs/info` chestInfo，帮助图标按 `config/help_default.js` 中列出该命令的条目选取，上游未列出的命令不显示图标），图片地址改为宿主渲染资源。幽境危战的角色卡片取自实时的角色详情查询，因此去掉了上游“先更新面板”的提示。模板用到的图片、字体与资料文件取自各上游的固定提交，由 `scripts/bundle-artwork.py` 复制到插件包的 `assets/` 随插件分发（许可同各上游，见 `LICENSES/`）；管理员可通过“素材更新”从上游仓库下载新版本，下载的文件优先使用。

`templates/uid-list/` 按 Miao-Yunzai 原神插件的 html/user/uid-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），页面框架与样式沿用 miao-plugin 的 common/layout/elem 与 common/common.css（MIT，见 `LICENSES/miao-plugin-MIT.txt`），同 miao 的 1.4 倍缩放；图片地址改为宿主渲染资源。玩家的头像与名片按 miao 的 faceImgs 取该 UID 已保存面板中编号最小的角色，没有时用 miao 的通用头像与名片。

`templates/rank/` 按 miao-plugin 的 character/rank-profile-list 改写（MIT），容器宽度同上游在页面内设置；ark-plugin 的自定义排行沿用这一页面（MIT，见 `LICENSES/ark-plugin-MIT.txt`），同 `apps/customRank.js` 写入“全服数据”、排序与筛选说明及请求行数，页面宽 850，评级按其平均单件分阈值（56 分以上为 MAX）。

`templates/stygian-rank/` 按 ark-plugin 的 character/stygian-rank-list 改写（MIT，见 `LICENSES/ark-plugin-MIT.txt`），样式为其 common/common.css（与 miao 相同）与 stygian-rank-list.css 的转换，页面宽度同 `apps/user.js` 按显示的排名栏计算；难度奖章取自“ark 插件图片”素材来源的 `resources/character/img/`，字体与背景取自 miao 的相同文件。上游样式引用的物品底图与名次徽标不在 ark 的资源中，上游页面上为空白，这里同样留空。akasha.cv 的排行按上游的查询读取（上游在查询中写死了一个开发者的 UID 参数，对结果没有影响，这里留空）。

`templates/rank-stats/` 按 ark-plugin 的 graph/stats 改写（MIT，见 `LICENSES/ark-plugin-MIT.txt`），页面框架沿用 miao-plugin 的 common/layout/default 与 common/common.css，同 ark 的 1.4 倍缩放。上游在浏览器中用 ECharts 绘制折线，这里按 ECharts 的刻度与平滑算法输出相同的 SVG，不随插件分发该库；背景图取自“ark 插件图片”素材来源的 `resources/graph/`（该来源另含幽境危战排名的奖章 `resources/character/img/`）。

`templates/panel/` 另按 ark-plugin 备份的 `miao-plugin-rank/resources/character/profile-detail.html`（MIT）加入面板排名：名次行接在伤害表后，排名统计块照搬其布局与内联样式，两张图按其 ECharts 选项（默认网格、smooth、面积渐变与标注）输出为 SVG，渐变的色标按上游的原顺序输出（上游把红色色标写在前一色标之前，浏览器因此从本面板的位置起画 2.5 个百分点的红带，这里保持一致）。

面板帮助同 miao-plugin 的 character/profile-detail 帮助，直接发送该仓库的 `resources/character/imgs/help.jpg`（MIT），取自 miao 素材来源。

`templates/news/` 与 `templates/news-list/` 按 Yunzai 原神插件的 html/mysNews 与 html/mysNews-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），公告、资讯、活动、米游社搜索、帖子与预估按其 mysNews 的规则出图；详情页的样式去掉了米游社编辑器、加载、提示与回复控件等页面用不到的规则。米游社正文是任意 HTML，而渲染器可以联网，因此正文按页面样式用到的标签、类名与颜色字号等样式重建，图片只经渲染资源引用（官方图片缓存），链接与脚本不保留。与上游不同之处：正文中没有地址的超链图片按帖子的 structured_content 补上（上游显示为空白）；纯图片帖显示全部图片（上游只显示最后一张）；上游按 4000 像素分段截图，这里出一整张长图；二维码由 go-qrcode 生成（MIT，见 `LICENSES/go-qrcode-MIT.txt`），指向帖子所在游戏的米游社地址（上游固定为原神路径）。页面的米游社标志、图标字体、列表页背景与数字字体取自“原神插件图片”素材来源。
