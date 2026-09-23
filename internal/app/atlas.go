package app

import (
	"bytes"
	"encoding/json"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
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
// as PickRule builds it.
type atlasRule struct {
	condition int
	pick      *regexp.Regexp
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
	return atlasRule{r.Condition, pick}, err
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
// of each key, and the key each alias of its othername file stands for.
type atlasModule struct {
	name    string
	rule    atlasRule
	paths   map[string]string
	aliases map[string]string
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

// atlasAnswer is a reply Atlas sends.
type atlasAnswer struct {
	picture artworkFile
}

// atlasRun is Atlas's handling of one message, which collects the replies.
type atlasRun struct {
	app     *App
	aliases func() map[string]string
	answers []atlasAnswer
}

// atlas is Atlas's atlas() for a message as Atlas reads it: the libraries'
// modules in order, the first whose rule takes the message and has the
// image of the name answering. It returns true when Atlas ends the message.
func (r *atlasRun) atlas(msg string) bool {
	msg = strings.TrimSpace(msg)
	for _, library := range r.app.Game.Pictures.Atlas {
		if !r.app.Artwork.Ready(library.Source) {
			continue
		}
		for _, module := range r.app.atlasModules(library) {
			name, ok := module.rule.take(msg)
			if !ok {
				continue
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
