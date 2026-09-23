"""Converts the alias lists xiaoyao-cvs-plugin's 七圣召唤 card 图鉴 reads
(resources/Atlas_alias Basic_Event, wuqi_tujian and yuanmo_tujian); the plugin
reads only the JSON.

Usage: python scripts/import-xiaoyao-aliases.py <参考项目/2026-09-15>
Writes internal/assets/data/xiaoyao-aliases.json.
"""
import json
import shutil
import sys
from pathlib import Path

import yaml

refs = Path(sys.argv[1]).resolve()
plugin = Path(__file__).resolve().parents[1]
source = refs / "xiaoyao-cvs-plugin"
pinned = json.loads((refs / "xiaoyao-cvs-plugin.source.json").read_text(encoding="utf-8"))
base = source / "resources/Atlas_alias"


def read_yaml(name):
    return yaml.safe_load((base / (name + ".yaml")).read_text(encoding="utf-8"))


def names(table):
    # info_img compares the names and aliases with the word as written, so
    # each must be text.
    out = []
    for name, aliases in table.items():
        if not isinstance(name, str) or not isinstance(aliases, list) or not all(isinstance(alias, str) for alias in aliases):
            raise SystemExit(f"{name}: expected a name with a list of aliases")
        out.append({"name": name, "aliases": aliases})
    return out


cards = []
for category, entries in read_yaml("Basic_Event").items():
    # Each entry of a category is one card with its aliases.
    if not all(isinstance(entry, dict) and len(entry) == 1 for entry in entries):
        raise SystemExit(f"{category}: expected one card per entry")
    cards.append({"category": category, "cards": [card for entry in entries for card in names(entry)]})
out = {"version": pinned["commit"], "cards": cards, "weapons": names(read_yaml("wuqi_tujian")), "enemies": names(read_yaml("yuanmo_tujian"))}
text = json.dumps(out, ensure_ascii=False, separators=(",", ":")) + "\n"
(plugin / "internal/assets/data/xiaoyao-aliases.json").write_bytes(text.encode("utf-8"))
shutil.copyfile(source / "LICENSE", plugin / "LICENSES/xiaoyao-cvs-plugin-GPL-3.0.txt")
print(json.dumps({"version": pinned["commit"], "cards": sum(len(group["cards"]) for group in cards), "weapons": len(out["weapons"]), "enemies": len(out["enemies"])}))
