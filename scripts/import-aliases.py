"""Import miao-plugin's alias tables into internal/assets/catalog.json.

Usage: python scripts/import-aliases.py <参考项目/2026-09-15>

Takes character and weapon aliases from miao-plugin (character/alias.js,
weapon/alias.js `alias` and `abbr`, the abbreviation also kept as `abbr`), the
character names from miao's character index, which upstream resolves names
through, and the set abbreviations from artifact/alias.js as `set_abbrs` and
the set aliases as `set_aliases`; also miao's material abbreviations (material/abbr.js) as `material_abbrs`
and its 老婆 types (character/extra.js wifeCfg) as `wife_types`.
Only built-in aliases are replaced; custom aliases stay in plugin settings.
Needs node.
"""
import json
import pathlib
import subprocess
import sys

refs = pathlib.Path(sys.argv[1]).resolve()
root = pathlib.Path(__file__).resolve().parents[1]

EVAL = """
const fs = require('fs'), vm = require('vm')
const source = fs.readFileSync(process.argv[1], 'utf8').replace(/export const /g, 'exports.')
const exports = {}
vm.runInNewContext(source, { exports }, { timeout: 1000 })
process.stdout.write(JSON.stringify(exports))
"""


def module(path):
    out = subprocess.run(["node", "-e", EVAL, str(path)], check=True, capture_output=True, encoding="utf-8")
    return json.loads(out.stdout)


def words(value):
    if isinstance(value, list):
        return [str(item).strip() for item in value if str(item).strip()]
    return [item.strip() for item in str(value or "").split(",") if item.strip()]


def unique(values, exclude):
    seen, out = {exclude}, []
    for value in values:
        if value and value not in seen:
            seen.add(value)
            out.append(value)
    return out


def load():
    path = root / "internal/assets/catalog.json"
    raw = path.read_bytes().decode("utf-8")
    catalog = json.loads(raw)
    if json.dumps(catalog, ensure_ascii=False, indent=2) + "\n" != raw:
        raise SystemExit(f"{path} is not in the expected layout")
    return path, catalog


def save(path, catalog):
    path.write_bytes((json.dumps(catalog, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))


meta = refs / "miao-plugin/resources/meta-gs"
index = json.loads((meta / "character/data.json").read_text(encoding="utf-8"))
character_tables = module(meta / "character/alias.js")
characters = character_tables.get("alias", {})
weapons = module(meta / "weapon/alias.js")
path, catalog = load()
renamed, counted = [], 0
for entry in catalog["entries"]:
    if entry["kind"] == "character" and entry["id"] in index:
        record = index[entry["id"]]
        if record["name"] != entry["name"]:
            renamed.append(f"{entry['id']}:{entry['name']}->{record['name']}")
            entry["name"] = record["name"]
        abbr = record.get("abbr") or character_tables.get("abbr", {}).get(entry["name"], "")
        entry["aliases"] = unique([abbr] + words(characters.get(entry["name"])), entry["name"])
        # Upstream prints this short name where a long name does not fit.
        if abbr and abbr != entry["name"]:
            entry["abbr"] = abbr
        else:
            entry.pop("abbr", None)
    elif entry["kind"] == "weapon":
        abbr = weapons.get("abbr", {}).get(entry["name"], "")
        entry["aliases"] = unique([abbr] + words(weapons.get("alias", {}).get(entry["name"])), entry["name"])
        # Upstream prints this short name where a long name does not fit.
        if abbr and abbr != entry["name"]:
            entry["abbr"] = abbr
        else:
            entry.pop("abbr", None)
    counted += len(entry.get("aliases", []))
sets = module(meta / "artifact/alias.js")
catalog["set_abbrs"] = sets.get("setAbbr", {})
catalog["set_aliases"] = {name: words(value) for name, value in sets.get("setAlias", {}).items()}
catalog["material_abbrs"] = module(meta / "material/abbr.js").get("abbr", {})
# miao's 老婆 types (character/extra.js wifeCfg) by character ID, resolving
# names and aliases as its meta.getId does.
ids = {}
for entry in catalog["entries"]:
    if entry["kind"] == "character":
        for name in [entry["name"]] + entry.get("aliases", []):
            ids.setdefault(name, entry["id"])
wife = {}
for kind, names in module(meta / "character/extra.js").get("wifeCfg", {}).items():
    wife[kind] = [ids[name] for name in words(names) if name in ids]
catalog["wife_types"] = wife
save(path, catalog)
print(json.dumps({"aliases": counted, "set_abbrs": len(catalog["set_abbrs"]), "renamed": renamed}, ensure_ascii=False))
