package images

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/gacha"
)

// gachaWords are stripped from the command word before picking the pool, as
// upstream's getPool does.
var gachaWords = regexp.MustCompile(`#|抽卡|记录|祈愿|分析|池|原神|星铁|崩坏星穹铁道|铁道`)

// standardFiveStars never count as the rate-up in upstream's checkIsUp.
var standardFiveStars = []string{"莫娜", "七七", "迪卢克", "琴", "姬子", "杰帕德", "彦卿", "白露", "瓦尔特", "克拉拉", "布洛妮娅"}

// rateUpPeriods are the standard five-stars that were once a rate-up; upstream
// counts them as rate-up only inside these periods (UTC+8).
var rateUpPeriods = map[string][][2]string{
	"刻晴":    {{"2021-02-17 18:00:00", "2021-03-02 15:59:59"}},
	"提纳里":   {{"2022-08-24 06:00:00", "2022-09-09 17:59:59"}},
	"迪希雅":   {{"2023-03-01 06:00:00", "2023-03-21 17:59:59"}},
	"梦见月瑞希": {{"2025-02-12 06:00:00", "2025-03-04 17:59:59"}},
	"希儿":    {{"2023-04-26 06:00:00", "2023-05-17 17:59:59"}, {"2023-10-27 12:00:00", "2023-11-14 14:59:59"}},
	"刃":     {{"2023-07-19 06:00:00", "2023-08-09 11:59:59"}, {"2023-12-27 06:00:00", "2024-01-17 11:59:59"}},
	"符玄":    {{"2023-09-20 12:00:00", "2023-10-10 14:59:59"}, {"2024-05-29 12:00:00", "2024-06-18 14:59:59"}},
	"银狼": {{"2023-06-07 06:00:00", "2023-06-28 11:59:59"}, {"2023-12-06 12:00:00", "2023-12-26 14:59:59"},
		{"2025-02-05 12:00:00", "2025-02-25 14:59:59"}, {"2025-09-02 12:00:00", "2025-09-23 14:59:59"}},
	"银枝": {{"2023-12-06 12:00:00", "2023-12-26 14:59:59"}, {"2024-07-10 12:00:00", "2024-07-31 14:59:59"}},
	"云璃": {{"2024-07-31 06:00:00", "2024-08-21 11:59:59"}, {"2025-02-26 06:00:00", "2025-03-19 11:59:59"}},
}

func rateUp(record gacha.Record) bool {
	for _, name := range standardFiveStars {
		if record.Name == name {
			return false
		}
	}
	periods, known := rateUpPeriods[record.Name]
	if !known {
		return true
	}
	at, err := time.ParseInLocation("2006-01-02 15:04:05", record.Time, chinaTime)
	if err != nil {
		return false
	}
	for _, period := range periods {
		start, _ := time.ParseInLocation("2006-01-02 15:04:05", period[0], chinaTime)
		end, _ := time.ParseInLocation("2006-01-02 15:04:05", period[1], chinaTime)
		if !at.Before(start) && !at.After(end) {
			return true
		}
	}
	return false
}

// gachaPool picks the pool from the command word as upstream's getPool does.
func gachaPool(word string) (string, string) {
	switch gachaWords.ReplaceAllString(word, "") {
	case "常驻":
		return "200", "常驻"
	case "武器":
		return "302", "武器"
	case "集录":
		return "500", "集录"
	case "新手":
		return "100", "新手"
	}
	return "301", "角色"
}

// Gacha draws one pool's record the way Yunzai's gacha-log does: the summary
// lines for the pool type and every five-star with its pull count.
func Gacha(context gamekit.ImageContext, image gamekit.GachaImage) (gamekit.Image, bool) {
	pool, poolName := gachaPool(image.Word)
	records := []gacha.Record{}
	for _, record := range image.Archive.Records {
		if gacha.Pool("genshin", record.GachaType) == pool {
			records = append(records, record)
		}
	}
	if len(records) == 0 {
		return gamekit.Image{}, false
	}
	// Upstream reads the log newest first; record IDs grow with time.
	sort.Slice(records, func(i, j int) bool {
		if len(records[i].ID) != len(records[j].ID) {
			return len(records[i].ID) > len(records[j].ID)
		}
		return records[i].ID > records[j].ID
	})

	type five struct {
		record gacha.Record
		num    int
		up     bool
	}
	fives := []five{}
	fourCounts, fourOrder := map[string]int{}, []string{}
	fiveNum, fourNum, fiveSince, fourSince, noFive, noFour := 0, 0, 0, 0, 0, 0
	wai, weaponFive, weaponFour, all := 0, 0, 0, len(records)
	for _, record := range records {
		if record.Rank == "4" {
			fourNum++
			if noFour == 0 {
				noFour = fourSince
			}
			fourSince = 0
			if fourCounts[record.Name] == 0 {
				fourOrder = append(fourOrder, record.Name)
			}
			fourCounts[record.Name]++
			if record.ItemType == "武器" {
				weaponFour++
			}
		}
		fourSince++
		if record.Rank == "5" {
			fiveNum++
			if len(fives) > 0 {
				fives[len(fives)-1].num = fiveSince
			} else {
				noFive = fiveSince
			}
			fiveSince = 0
			up := false
			if record.ItemType == "角色" {
				if up = rateUp(record); !up {
					wai++
				}
			} else {
				weaponFive++
			}
			fives = append(fives, five{record: record, up: up})
		}
		fiveSince++
	}
	big := 0
	if len(fives) > 0 {
		fives[len(fives)-1].num = fiveSince
		for index := range fives {
			// A five-star right after an off-banner one came from the guarantee.
			if next := index + 1; next < len(fives) && !fives[next].up {
				big++
			}
		}
	} else {
		noFive = all
	}
	sort.SliceStable(fourOrder, func(i, j int) bool { return fourCounts[fourOrder[i]] > fourCounts[fourOrder[j]] })
	maxFourName, maxFour := "无", 0
	if len(fourOrder) > 0 {
		maxFourName, maxFour = fourOrder[0], fourCounts[fourOrder[0]]
	}
	average := func(total, count int) int {
		if count == 0 {
			return 0
		}
		return int(math.Round(float64(total) / float64(count)))
	}
	fiveAvg, fourAvg := average(all-noFive, fiveNum), average(all-noFour, fourNum)
	valid := 0
	if fiveNum > 0 && fiveNum > wai {
		if len(fives) > 0 && !fives[0].up {
			valid = average(all-noFive-fives[0].num, fiveNum-wai)
		} else {
			valid = average(all-noFive, fiveNum-wai)
		}
	}
	upCost := strconv.Itoa(valid * 160)
	if valid*160 >= 10000 {
		upCost = fmt.Sprintf("%.2fw", float64(valid*160)/10000)
	}
	noWaiRate := "0"
	if fiveNum > 0 {
		noWaiRate = fmt.Sprintf("%.1f", float64(fiveNum-big-wai)/float64(fiveNum-big)*100)
	}
	worst, best := 0, 0
	for _, item := range fives {
		if item.num == 0 {
			continue
		}
		if worst == 0 || item.num > worst {
			worst = item.num
		}
		if best == 0 || item.num < best {
			best = item.num
		}
	}
	cell := func(label string, num any, unit string) map[string]any {
		return map[string]any{"label": label, "num": fmt.Sprint(num), "unit": unit}
	}
	fourUnit := []rune(maxFourName)
	fourUnit = fourUnit[:min(len(fourUnit), 4)]
	var lines [][]any
	switch pool {
	case "301":
		lines = [][]any{
			{cell("未出五星", noFive, "抽"), cell("五星", fiveNum, "个"), cell("五星平均", fiveAvg, "抽"), cell("小保底不歪", noWaiRate+"%", ""), cell("最非", worst, "抽")},
			{cell("未出四星", noFour, "抽"), cell("五星常驻", wai, "个"), cell("UP平均", valid, "抽"), cell("UP花费原石", upCost, ""), cell("最欧", best, "抽")},
		}
	default:
		third := cell("四星武器", weaponFour, "个")
		if pool == "200" || pool == "100" {
			third = cell("五星武器", weaponFive, "个")
		}
		lines = [][]any{
			{cell("未出五星", noFive, "抽"), cell("五星", fiveNum, "个"), cell("五星平均", fiveAvg, "抽"), third, cell("最非", worst, "抽")},
			{cell("未出四星", noFour, "抽"), cell("四星", fourNum, "个"), cell("四星平均", fourAvg, "抽"), cell("四星最多", maxFour, string(fourUnit)), cell("最欧", best, "抽")},
		}
	}

	resources := []rayleabot.RenderImageResource{}
	add := func(id, source, name string) bool {
		resource, ok := context.ArtworkResource(id, source, name)
		if ok {
			resources = append(resources, resource)
		}
		return ok
	}
	add("tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf")
	add("HYWenHei-55W", "yunzai-genshin", "resources/font/HYWenHei-55W.ttf")
	add("img-other-bg4", "yunzai-genshin", "resources/img/other/bg4.png")
	add("img-other-bg5", "yunzai-genshin", "resources/img/other/bg5.png")
	add("logo", "yunzai-genshin", "resources/img/other/原神.png")
	add("head-img", "miao-plugin", "resources/meta-gs/character/闲云/imgs/banner.webp")
	weaponTypes := map[string]string{}
	if context.Game.Calc != nil {
		for _, weapon := range context.Game.Calc.Metadata().Weapons {
			weaponTypes[weapon.Name] = weapon.Type
		}
	}
	// Upstream colours a count by the pool's pity: 90, or 80 for weapons.
	pity := 90
	if pool == "302" {
		pity = 80
	}
	icons := map[string]string{}
	cards := []any{}
	for _, item := range fives {
		class := "bad"
		switch {
		case item.num <= 10:
			class = "gold"
		case float64(item.num) < float64(pity)*0.5:
			class = "good"
		case float64(item.num) < float64(pity)*0.83:
			class = "normal"
		}
		icon, seen := icons[item.record.Name]
		if !seen {
			path := "resources/meta-gs/character/" + item.record.Name + "/imgs/face.webp"
			if item.record.ItemType == "武器" {
				path = "resources/meta-gs/weapon/" + weaponTypes[item.record.Name] + "/" + item.record.Name + "/icon.webp"
			}
			icon = "icon-" + strconv.Itoa(len(icons))
			if !add(icon, "miao-plugin", path) {
				icon = ""
			}
			icons[item.record.Name] = icon
		}
		cards = append(cards, map[string]any{"class": class, "up": item.up && poolName == "角色", "icon": icon, "num": item.num})
	}
	first, last := records[len(records)-1].Time, records[0].Time
	return gamekit.Image{Template: "gacha", Data: map[string]any{
		"uid": image.UID, "all": all, "pool": pool, "pool_name": poolName, "lines": lines, "cards": cards,
		"first_time": first[:min(len(first), 16)], "last_time": last[:min(len(last), 16)],
	}, Resources: resources}, true
}
