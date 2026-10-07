package app

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/RayleaBot/genshin/internal/pluginmeta"
)

// commandSet maps the command word the host delivered back to this plugin's
// declaration. The host matched the word against the same triggers and only
// passes the word on, so the plugin repeats the lookup in the host's order:
// declarations in manifest order, exact names compared as is, patterns with
// MatchString. Handlers dispatch on the command ID, so display names and
// trigger words can follow upstream without touching them.
type commandSet struct {
	commands []declaredCommand
}

type declaredCommand struct {
	id       string
	names    []string
	pattern  *regexp.Regexp
	fallback bool
}

func newCommandSet(manifest pluginmeta.Manifest) (commandSet, error) {
	set := commandSet{}
	for _, command := range manifest.Commands {
		declared := declaredCommand{id: command.ID, fallback: command.Trigger.Fallback}
		switch command.Trigger.Type {
		case "exact":
			declared.names = command.Trigger.Names
		case "pattern":
			expression, err := regexp.Compile(command.Trigger.Pattern)
			if err != nil {
				return set, fmt.Errorf("command %s has an invalid pattern: %w", command.ID, err)
			}
			declared.pattern = expression
		default:
			continue
		}
		set.commands = append(set.commands, declared)
	}
	return set, nil
}

// resolve returns the command ID and arguments for a delivered command word.
// The named groups of a pattern trigger become the leading arguments, so
// "雷神面板 1000" reaches the panel handler with ["雷神", "1000"]. Words sent
// after a command its upstream rule reads to the end are read as that rule
// reads them (upstreamArgs), and a command whose rule does not take them is
// passed over, as the message goes on to other rules upstream. As the host
// does, fallback commands are tried only after every ordinary one.
func (s commandSet) resolve(word string, args []string) (string, []string, bool) {
	if id, leading, ok := s.match(strings.TrimSpace(word), args, false); ok {
		return id, leading, ok
	}
	return s.match(strings.TrimSpace(word), args, true)
}

func (s commandSet) match(word string, args []string, fallback bool) (string, []string, bool) {
	if word == "" {
		return "", args, false
	}
	for _, command := range s.commands {
		if command.fallback != fallback {
			continue
		}
		leading := []string{}
		if command.pattern == nil {
			if !slices.ContainsFunc(command.names, func(name string) bool { return strings.TrimSpace(name) == word }) {
				continue
			}
		} else {
			match := command.pattern.FindStringSubmatch(word)
			if match == nil {
				continue
			}
			for i, group := range command.pattern.SubexpNames() {
				if group != "" && match[i] != "" {
					leading = append(leading, match[i])
				}
			}
		}
		if out, taken := upstreamArgs(command.id, word, leading, args); taken {
			return command.id, out, true
		}
	}
	return "", args, false
}

// upstreamTails are the upstream rules of the commands whose rules read
// words after the command: Yunzai's that end in [ |0-9]* and miao's that
// take the same words over, as Miao-Yunzai runs miao's first. The words
// after the command keep their intended writing: [ |0-9]* is spaces and
// digits.
var upstreamTails = map[string][]*regexp.Regexp{
	"abyss-summary": {regexp.MustCompile(`^#*(喵喵|上传|本期)*(深渊|深境|深境螺旋)[ 0-9]*(数据)?$`)},
	"abyss":         {regexp.MustCompile(`^#[上期|往期|本期]*(深渊|深境|深境螺旋)[上期|往期|本期]*[ 0-9]*$`)},
	"abyss-floor":   {regexp.MustCompile(`^#*[上期|往期|本期]*(深渊|深境|深境螺旋)[上期|往期|本期]*[第]*(9|10|11|12|九|十|十一|十二)层[ 0-9]*$`)},
	"weapons":       {regexp.MustCompile(`^#[五星|四星|5星|4星]*武器[ 0-9]*$`)},
	"profile":       {regexp.MustCompile(`^#(宝箱|成就|尘歌壶|家园|探索|探险|声望|探险度|探索度)[ 0-9]*$`), regexp.MustCompile(`^(#*角色3|#*角色卡片|角色)$`)},
	"characters": {regexp.MustCompile(`^#喵喵(角色|查询)[ 0-9]*$`),
		regexp.MustCompile(`(^#(五|四|5|4|星)*(角色|查询|查询角色|角色查询|人物)[ 0-9]*$)|(^(#*uid|#*UID)\+*(18|[1-9])[0-9]{8}$)|(^#[+＋]*(18|[1-9])[0-9]{8})`)},
	"training": {regexp.MustCompile(`^#(星铁|原神)?(面板|喵喵)?练度统计$`),
		regexp.MustCompile(`^#*(我的)*(风|岩|雷|草|水|火|冰)*(武器|角色|练度|五|四|5|4|星)+(汇总|统计|列表)(force|五|四|5|4|星)*[ 0-9]*$`)},
	"theater-training": {regexp.MustCompile(`^#202\d{3}(幻想|真境|剧诗|幻想真境剧诗)练度统计$`),
		regexp.MustCompile(`^#202\d{3}(幻想|真境|剧诗|幻想真境剧诗)(角色|练度)(汇总|统计|列表)[ 0-9]*$`)},
	"talent-stat":    {regexp.MustCompile(`^#*(我的)?(今日|今天|明日|明天|周.*)?([五四54]星)?(技能|天赋)+(汇总|统计|列表)?[ 0-9]*$`)},
	"daily-material": {regexp.MustCompile(`^#(今日|今天|每日|我的|明天|明日|周([1-7]|一|二|三|四|五|六|日))*(素材|材料|天赋)[ 0-9]*$`)},
	"theater":        {regexp.MustCompile(`^#*(喵喵)*(本期|上期)?(幻想|幻境|剧诗|幻想真境剧诗)[ 0-9]*(数据)?$`)},
	"role-cards":     {regexp.MustCompile(`^#*(喵喵)*(月谕|越狱|幻想|幻境|剧诗|幻想真境剧诗)(圣牌|卡片|卡牌|塔罗牌|card|tarot)(收藏|收集)?[ 0-9]*(数据)?$`)},
	"hard_challenge": {regexp.MustCompile(`^#*(喵喵)*(本期|上期)?(幽境|危战|幽境危战)(单人|单挑|组队|多人|合作|最佳)?[ 0-9]*(数据)?$`)},
}

// upstreamArgs reads the words sent after a command as its upstream rules
// read the message: words no rule takes leave the message to other rules
// (taken is false), and of the words a rule takes only a UID counts, the
// first Yunzai's getUid finds, which a UID written with the command comes
// before; other digits are ignored and the current UID read. A command word
// no rule takes is the plugin's own and keeps its words.
func upstreamArgs(id, word string, leading, args []string) (out []string, taken bool) {
	rules := upstreamTails[id]
	matches := func(message string) bool {
		return slices.ContainsFunc(rules, func(rule *regexp.Regexp) bool { return rule.MatchString(message) })
	}
	if len(args) == 0 || !matches("#"+word) {
		return append(leading, args...), true
	}
	if !matches("#" + word + " " + strings.Join(args, " ")) {
		return nil, false
	}
	if slices.ContainsFunc(leading, cardUID.MatchString) {
		return leading, true
	}
	for _, arg := range args {
		if uid := cardUID.FindString(arg); uid != "" {
			return append(leading, uid), true
		}
	}
	return leading, true
}

// usage writes a declared usage with the prefix replies use. Manifests start
// usages with the placeholder "#", which the host help menu also replaces.
func (a *App) usage(text string) string {
	if strings.HasPrefix(text, "#") {
		return a.Game.Prefix + text[1:]
	}
	return text
}
