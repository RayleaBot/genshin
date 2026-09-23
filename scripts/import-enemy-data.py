"""Converts the pinned Atlas enemy data; the plugin reads only the JSON.

Usage: python scripts/import-enemy-data.py <Atlas snapshot>
Writes internal/assets/data/enemies.json.
"""
import json
import shutil
import sys
from pathlib import Path

import yaml

source = Path(sys.argv[1]).resolve()
plugin = Path(__file__).resolve().parents[1]
# The snapshot is an extracted archive, not a Git checkout, so its commit is
# recorded here.
version = "016e49357666e0823791abdf28fbb3b2efe68225"
base = source / "resource/enemy"


def read_yaml(name):
    return yaml.safe_load((base / name).read_text(encoding="utf-8"))


enemies = read_yaml("Enemy.yaml")
aliases = read_yaml("OtherName.yaml")
curves = [{key: float(value) for key, value in row.items()} for row in read_yaml("Common.yaml")]
# OtherName keeps the file's order, which decides between names of equal
# length, and the keys that name no enemy, which the chat still matches.
# Four keys miss their Enemy.yaml rows by a slip (察 for 查, a dropped
# hyphen, reversed words, 型 once for twice), so upstream finds no enemy for
# them; here they name the rows. Each key is also among its own aliases, so
# a message still matches it.
enemy_names = {"遗迹侦察者": "遗迹侦查者", "帽子水母大": "帽子水母-大", "狂暴公义": "公义狂暴", "始基动能型场力发生装置": "始基动能型型场力发生装置"}
out = {"version": "Atlas-" + version, "enemies": [], "names": [{"name": enemy_names.get(name, name), "aliases": names} for name, names in aliases.items()], "curves": curves, "modifiers": []}
for group, rows in enemies.items():
    for row in rows:
        out["enemies"].append({
            "name": row[group],
            "group": group,
            "hp_curve": row.get("HPScale", ""),
            "hp_base": row.get("HPValue"),
            "atk_curve": row.get("ATKScale", ""),
            "atk_base": row.get("ATKValue"),
        })
factors = json.loads((base / "EnemyOtherAttributionData.json").read_text(encoding="utf-8"))
for group, rows in factors.items():
    for i, row in enumerate(rows):
        names = row["Factors"]
        out["modifiers"].append({
            "id": f"{group}:{i}",
            "group": group,
            "names": [names] if isinstance(names, str) else names,
            "level": row.get("Level", 0),
            "value": row.get("HPRatioValue", row.get("ATKRatioValue")),
        })
text = json.dumps(out, ensure_ascii=False, separators=(",", ":")) + "\n"
(plugin / "internal/assets/data/enemies.json").write_bytes(text.encode("utf-8"))
shutil.copyfile(source / "LICENSE", plugin / "LICENSES/Atlas-GPL-3.0.txt")
print(json.dumps({"version": version, "enemies": len(out["enemies"]), "names": len(out["names"]), "curves": len(curves), "modifiers": len(out["modifiers"])}))
