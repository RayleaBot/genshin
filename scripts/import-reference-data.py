"""Convert pinned local miao-plugin JSON into internal/assets/catalog.json.

Usage: python scripts/import-reference-data.py --references <参考项目/2026-09-15>

This reads data only. It never evaluates reference JavaScript or installs its
dependencies. Generated catalogs carry their source version and attribution.
"""
import argparse
import html
import json
import pathlib
import re
import shutil

parser = argparse.ArgumentParser()
parser.add_argument("--references", type=pathlib.Path, required=True)
args = parser.parse_args()
refs = args.references.resolve()
root = pathlib.Path(__file__).resolve().parents[1]

def clean(value):
    if not isinstance(value, str):
        return ""
    value = re.sub(r"<br\s*/?>", "\n", value, flags=re.I)
    value = html.unescape(re.sub(r"<[^>]*>", "", value))
    return value.encode("utf-8", errors="replace").decode("utf-8").strip()

def load(file):
    return json.loads(file.read_text(encoding="utf-8"))

def dump(file, value):
    file.parent.mkdir(parents=True, exist_ok=True)
    file.write_bytes((json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))

elements = {"anemo": "风", "geo": "岩", "electro": "雷", "dendro": "草", "hydro": "水", "pyro": "火", "cryo": "冰", "multi": "多属性"}
weapons = {"sword": "单手剑", "claymore": "双手剑", "polearm": "长柄武器", "bow": "弓", "catalyst": "法器"}
material_names = {"gem": "突破宝石", "boss": "首领材料", "specialty": "地区特产", "normal": "通用材料", "talent": "天赋材料", "weekly": "周本材料", "weapon": "武器材料"}

entries = {}
base = refs / "miao-plugin/resources/meta-gs"
for kind in ("character", "weapon"):
    for file in sorted((base / kind).rglob("data.json")):
        # The traveler's element folders hold its talents per element under
        # the shared id 7; the traveler is the folder's own data.
        if kind == "character" and file.parent.parent != base / kind:
            continue
        raw = load(file)
        if not isinstance(raw, dict) or not raw.get("name") or not raw.get("id"):
            continue
        item_id = str(raw["id"])
        talents = []
        for talent in (raw.get("talent") or {}).values():
            if not isinstance(talent, dict):
                continue
            description = talent.get("desc", [])
            if isinstance(description, str):
                description = [description]
            talents.append({"name": clean(talent.get("name")) or "技能", "description": [clean(line) for line in description if clean(line)]})
        if item_id in entries:
            continue
        entries[item_id] = {
            "id": item_id,
            "name": clean(raw["name"]),
            "aliases": [clean(raw["abbr"])] if raw.get("abbr") and raw["abbr"] != raw["name"] else [],
            "kind": kind,
            "rarity": raw.get("star", 0),
            "element": elements.get(raw.get("elem"), raw.get("elem", "")),
            "weapon": weapons.get(raw.get("weapon"), raw.get("weapon", "")),
            "description": clean(raw.get("desc")),
            "materials": {material_names.get(k, k): clean(v) for k, v in (raw.get("materials") or {}).items() if isinstance(v, str)},
            "stats": raw.get("baseAttr", {}),
            "talents": talents,
        }
source = "https://github.com/yoimiya-kokomi/miao-plugin/tree/7f6f1c84c89102bc6b1c58c8e1b06c61f4642161/resources/meta-gs"
dump(root / "internal/assets/catalog.json", {"version": "miao-7f6f1c84c891", "source": source, "entries": sorted(entries.values(), key=lambda item: item["id"])})
shutil.copyfile(refs / "miao-plugin/LICENSE", root / "LICENSES/miao-plugin-MIT.txt")
print(json.dumps({"entries": len(entries)}, ensure_ascii=True))
