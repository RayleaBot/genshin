package app

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Atlas (Nwflower/Atlas apps/atlas.js) answers a message with a picture of
// its downloaded library when one of the library's modules takes the message:
// the module's rule decides whether it does and which name is left, and the
// name is looked up among the module's aliases.

// AtlasLibrary is an Atlas library: its path.json Index, the Othername
// directory of Atlas's own copies of the library's alias files, read for the
// modules the library has none for, and the Rules of its modules (Atlas's
// rule_default/<module>.yaml), config for the others.
type AtlasLibrary struct {
	Source    string               `json:"source"`
	Index     string               `json:"index"`
	Othername artworkFile          `json:"othername"`
	Rules     map[string]AtlasRule `json:"rules"`
}

// AtlasRule is which messages an Atlas module takes, as its PickRule reads
// them with # for the prefix: Condition 0 any message, 1 one with the
// prefix, 2 one with a Pick word, 3 either, 4 both, 5 one with a Pick word;
// the Pick words and, except for 2, the prefix are removed before the name
// is looked up. The Genshin rules all match aliases exactly (mode 0) and
// keep Atlas's # prefix, so neither setting is carried.
type AtlasRule struct {
	Condition int      `json:"condition"`
	Pick      []string `json:"pick"`
}

// atlasRule is an AtlasRule with its Pick words joined into one expression,
// as PickRule builds it, and the first of them, which index() leads a list's
// entries with.
type atlasRule struct {
	condition int
	pick      *regexp.Regexp
	first     string
}

var (
	atlasPrefixed = regexp.MustCompile(`^#.*$`)
	atlasPrefix   = regexp.MustCompile(`^#*`)
)

func (r AtlasRule) compile() (atlasRule, error) {
	words := r.Pick
	if len(words) == 0 {
		words = []string{"图鉴"}
	}
	pick, err := regexp.Compile("(" + strings.Join(words, "|") + ")")
	return atlasRule{r.Condition, pick, words[0]}, err
}

// take is PickRule: the name a message leaves, false when the rule does not
// take the message.
func (r atlasRule) take(msg string) (string, bool) {
	picked, prefixed := r.pick.MatchString(msg), atlasPrefixed.MatchString(msg)
	name := ""
	switch {
	case r.condition == 0, r.condition == 1 && prefixed:
		name = atlasPrefix.ReplaceAllString(msg, "")
	case r.condition == 2 && picked:
		name = r.pick.ReplaceAllString(msg, "")
	case r.condition == 3 && (picked || prefixed), r.condition == 4 && picked && prefixed, r.condition == 5 && picked:
		name = r.pick.ReplaceAllString(atlasPrefix.ReplaceAllString(msg, ""), "")
	default:
		return "", false
	}
	name = strings.TrimSpace(name)
	return name, name != ""
}

// atlasModule is a path.json module as Atlas searches it: its rule, the image
// of each key, the key each alias of its othername file stands for, and the
// numbered lists of the library's index/<module>.yaml by key.
type atlasModule struct {
	name    string
	rule    atlasRule
	paths   map[string]string
	aliases map[string]string
	index   map[string][]string
}

// atlasLibraries keeps each library's modules, read again after the library
// or Atlas's own files are downloaded anew.
type atlasLibraries struct {
	mu    sync.Mutex
	items map[string]atlasModules
}

type atlasModules struct {
	versions [2]time.Time
	modules  []atlasModule
}

// atlasModules reads a library's path.json in the file's order, which Atlas
// searches in.
func (a *App) atlasModules(library AtlasLibrary) []atlasModule {
	versions := [2]time.Time{a.Artwork.Version(library.Source), a.Artwork.Version(library.Othername.Source)}
	a.atlases.mu.Lock()
	defer a.atlases.mu.Unlock()
	if cached, ok := a.atlases.items[library.Source]; ok && cached.versions == versions {
		return cached.modules
	}
	raw, err := a.Artwork.Open(library.Source, library.Index)
	if err != nil {
		return nil
	}
	modules := []atlasModule{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil
		}
		module := atlasModule{name: token.(string), paths: map[string]string{}}
		if decoder.Decode(&module.paths) != nil {
			return nil
		}
		rule, ok := library.Rules[module.name]
		if !ok {
			rule = library.Rules["config"]
		}
		if module.rule, err = rule.compile(); err != nil {
			continue
		}
		// The library's othername file wins over Atlas's copy.
		aliases, err := a.Artwork.Open(library.Source, "othername/"+module.name+".yaml")
		if err != nil {
			aliases, err = a.Artwork.Open(library.Othername.Source, library.Othername.Path+"/"+module.name+".yaml")
		}
		if err == nil {
			module.aliases = atlasAliases(aliases)
		}
		if index, err := a.Artwork.Open(library.Source, "index/"+module.name+".yaml"); err == nil {
			module.index = map[string][]string{}
			for _, list := range yamlLists(index) {
				module.index[list.key] = list.items
			}
		}
		modules = append(modules, module)
	}
	if a.atlases.items == nil {
		a.atlases.items = map[string]atlasModules{}
	}
	a.atlases.items[library.Source] = atlasModules{versions, modules}
	return modules
}

// yamlList is a top-level key of Atlas's YAML files with the list under it.
type yamlList struct {
	key   string
	items []string
}

// yamlLists reads the YAML files Atlas keeps names in: top-level keys, each
// followed by a list, in the file's order.
func yamlLists(raw []byte) []yamlList {
	unquote := func(value string) string {
		if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') && value[len(value)-1] == value[0] {
			return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
		}
		return value
	}
	lists := []yamlList{}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "" || strings.HasPrefix(trimmed, "#"):
		case strings.HasPrefix(trimmed, "- "):
			if item := unquote(strings.TrimSpace(trimmed[2:])); len(lists) > 0 && item != "" {
				lists[len(lists)-1].items = append(lists[len(lists)-1].items, item)
			}
		case line[0] != ' ' && strings.HasSuffix(trimmed, ":"):
			lists = append(lists, yamlList{key: unquote(strings.TrimSuffix(trimmed, ":"))})
		}
	}
	return lists
}

// atlasAliases maps each name of an othername file to its key. A name listed
// under several keys belongs to the first, as Atlas finds it.
func atlasAliases(raw []byte) map[string]string {
	aliases := map[string]string{}
	for _, list := range yamlLists(raw) {
		for _, alias := range list.items {
			if _, taken := aliases[alias]; !taken {
				aliases[alias] = list.key
			}
		}
	}
	return aliases
}

// roleWords are what Yunzai's gsCfg.getRole removes from a name.
var roleWords = regexp.MustCompile(`#|老婆|老公|(18|[1-9])[0-9]{8}`)

// atlasName is getName: the key the module's aliases give the name, else,
// for modules of roles, the character the name is (Yunzai's gsCfg.getRole,
// miao's Character.get on Miao-Yunzai), else the name. Atlas also swaps in
// aliases from miao-plugin's config/roleName.yaml, a file miao does not
// have, so that step never applies.
func (a *App) atlasName(module atlasModule, name string, aliases func() map[string]string) string {
	if key, ok := module.aliases[name]; ok {
		return key
	}
	if strings.Contains(module.name, "role") {
		if entry, ok := a.miaoCharacter(strings.TrimSpace(roleWords.ReplaceAllString(name, "")), aliases()); ok {
			return entry.Name
		}
	}
	return name
}

// atlasAnswer is a reply Atlas sends: a picture, or a numbered list whose
// entries are messages as Atlas reads them.
type atlasAnswer struct {
	picture artworkFile
	list    []string
}

// atlasRun is Atlas's handling of one message from a sender, which collects
// the replies.
type atlasRun struct {
	app     *App
	owner   Subject
	aliases func() map[string]string
	answers []atlasAnswer
}

// atlas is Atlas's atlas() for a message as Atlas reads it. A sender's list
// takes the message first (select), and a pick that answers resets the
// sender's count. Then the libraries' modules are searched in order: the
// first whose rule takes the message answers with its numbered list when the
// name is a key of its index, else with the image of the name when it has
// one. It returns true when Atlas ends the message: after a picture, or
// after a pick that sent anything; after its own list Atlas passes the
// message on.
func (r *atlasRun) atlas(msg string) bool {
	msg = strings.TrimSpace(msg)
	if entry, ok := r.app.atlasMenus.pick(r.owner, msg); ok && r.atlas(entry) {
		r.app.atlasMenus.answered(r.owner)
	}
	for _, library := range r.app.Game.Pictures.Atlas {
		if !r.app.Artwork.Ready(library.Source) {
			continue
		}
		for _, module := range r.app.atlasModules(library) {
			name, ok := module.rule.take(msg)
			if !ok {
				continue
			}
			if entries, ok := module.index[name]; ok {
				list := module.rule.lead(entries)
				r.answers = append(r.answers, atlasAnswer{list: list})
				r.app.atlasMenus.open(r.owner, list)
				return false
			}
			file, ok := module.paths[r.app.atlasName(module, name, r.aliases)]
			file = strings.TrimPrefix(file, "/")
			if !ok || !slices.Contains(pictureExtensions, strings.ToLower(path.Ext(file))) {
				continue
			}
			if _, found := r.app.Artwork.File(library.Source, file); found {
				r.answers = append(r.answers, atlasAnswer{picture: artworkFile{library.Source, file}})
				return true
			}
		}
	}
	return len(r.answers) > 0
}

// lead is how index() writes a list's entries: led by what the rule takes a
// message with, the prefix for conditions 1 and 3, the first Pick word for 2,
// both for 4.
func (r atlasRule) lead(entries []string) []string {
	lead := map[int]string{1: "#", 2: r.first, 3: "#", 4: "#" + r.first}[r.condition]
	list := []string{}
	for _, entry := range entries {
		list = append(list, lead+entry)
	}
	return list
}

// atlasMenus are the numbered lists Atlas last sent each sender (its
// context) with the messages the sender sent since without a pick that
// answered (its num).
type atlasMenus struct {
	mu    sync.Mutex
	items map[Subject]*atlasMenu
}

type atlasMenu struct {
	entries []string
	count   int
}

// open keeps the list sent to a sender, keeping the sender's count. Lists
// stay until their sender moves on, so all are dropped once 4096 senders
// have one.
func (m *atlasMenus) open(owner Subject, entries []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if menu := m.items[owner]; menu != nil {
		menu.entries = entries
		return
	}
	if m.items == nil || len(m.items) >= 4096 {
		m.items = map[Subject]*atlasMenu{}
	}
	m.items[owner] = &atlasMenu{entries: entries}
}

// pick is select()'s bookkeeping for a sender's message: it counts the
// message, drops the list at the fourth, and returns the entry a number in
// the message names.
func (m *atlasMenus) pick(owner Subject, msg string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	menu := m.items[owner]
	if menu == nil {
		return "", false
	}
	if menu.count++; menu.count >= 4 {
		delete(m.items, owner)
		return "", false
	}
	// JavaScript's Number reads the message; a number that names no entry
	// makes select fail.
	number, err := strconv.ParseFloat(msg, 64)
	if err != nil || number != math.Trunc(number) || number < 1 || number > float64(len(menu.entries)) {
		return "", false
	}
	return menu.entries[int(number)-1], true
}

// answered resets a sender's count after a pick that answered.
func (m *atlasMenus) answered(owner Subject) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if menu := m.items[owner]; menu != nil {
		menu.count = 0
	}
}

// AtlasIndexImage is a numbered list Atlas draws, for the sender Name.
type AtlasIndexImage struct {
	Name    string
	Entries []string
}

// AtlasIndexImageBuilder draws an AtlasIndexImage.
type AtlasIndexImageBuilder func(ImageContext, AtlasIndexImage) (Image, bool)

// postAtlasList sends a numbered list as index() means to: the lines split
// into forwarded messages by Reply.replyMessageArray, drawn as the
// AtlasIndex page when they cannot be forwarded. Atlas only builds the
// forwarded messages (Miao-Yunzai's makeForwardMsg sends nothing) and so
// always falls back to the page; the forward it means is sent here.
func (a *App) postAtlasList(ctx context.Context, event *rayleabot.EventContext, list []string) error {
	lines := []string{"请直接发送数字序号或对应指令："}
	entries := []string{}
	for index, entry := range list {
		if rest, ok := strings.CutPrefix(entry, "#"); ok {
			entry = a.Game.Prefix + rest
		}
		entries = append(entries, entry)
		lines = append(lines, strconv.Itoa(index+1)+"、"+entry)
	}
	forwarded := true
	for _, page := range atlasPages(lines) {
		parts := [][]rayleabot.Segment{}
		for _, message := range page {
			parts = append(parts, []rayleabot.Segment{rayleabot.Text(message)})
		}
		if forwarded = a.forward(ctx, event, parts); !forwarded {
			break
		}
	}
	if forwarded {
		return nil
	}
	view := View{Title: lines[0], Rows: []Row{}}
	for index, entry := range entries {
		view.Rows = append(view.Rows, Row{Label: strconv.Itoa(index + 1), Value: entry})
	}
	if a.atlasIndexImage != nil {
		if drawn, ok := a.atlasIndexImage(a.imageContext(ctx), AtlasIndexImage{Name: event.Event.Actor.Nickname, Entries: entries}); ok {
			view.Image = &drawn
		}
	}
	reply := rayleabot.Text(strings.Join(lines, "\n"))
	if image := a.renderView(ctx, event, view); image != "" {
		reply = rayleabot.Image(image)
	}
	_, err := post(ctx, event, reply)
	return err
}

// atlasPages are replyMessageArray's forwarded messages: lines joined until
// a message reaches 100 characters, 95 messages a page, each page closed by
// its number.
func atlasPages(lines []string) [][]string {
	pages := [][]string{}
	page := []string{}
	message := ""
	for _, line := range lines {
		if len(page) >= 95 {
			pages = append(pages, append(page, "第"+strconv.Itoa(len(pages)+1)+"页...未完待续"))
			page = []string{}
		}
		switch {
		case message != "" && len(utf16.Encode([]rune(message))) < 100:
			message += "\n" + line
		case message == "":
			message = line
		default:
			page = append(page, message)
			message = line
		}
	}
	if message != "" {
		number := strconv.Itoa(len(pages) + 1)
		page = append(page, message, "第"+number+"页，共"+number+"页")
	}
	if len(page) > 0 {
		pages = append(pages, page)
	}
	return pages
}
