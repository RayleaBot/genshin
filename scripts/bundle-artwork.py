"""Copies the upstream images, fonts and data files the templates read from
the pinned reference snapshots into assets/<source>/. The plugin build ships
the top-level assets/ directory, and the artwork store serves these files until
an administrator downloads a newer copy of the source. assets/<source>.json
records the upstream commit and the totals the status shows.

Usage: python scripts/bundle-artwork.py --references <参考项目/2026-09-15>

Run it before building a package. Official images are fetched file by file at
render time, and the Atlas and xiaoyao 图鉴 are downloaded on request, so none
of them are bundled.
"""
import argparse
import fnmatch
import json
import pathlib
import shutil

# Source id: the reference snapshot and the files the Go code reads from it.
# A pattern's * also matches "/".
BUNDLES = {
    "miao-plugin": ("miao-plugin", [
        "resources/common/font/tttgbnumber.woff",
        "resources/common/font/NZBZ.woff",
        "resources/common/font/HYWH-65W.woff",
        "resources/common/cont/card-bg.png",
        "resources/common/cont/logo.png",
        "resources/common/bg/bg-*.webp",
        "resources/common/bg/talent-*.webp",
        "resources/common/item/*",
        "resources/character/imgs/*",
        "resources/wiki/imgs/*",
        "resources/help/icon.png",
        "resources/help/theme/default/*",
        "resources/stat/imgs/bg1.png",
        "resources/stat/imgs/footer.png",
        "resources/gacha/imgs/*",
        # Panels, rankings, rosters, the calendar and the wiki draw every
        # character's portraits and icons; the Traveler's element icons sit
        # under 旅行者/<element>/.
        "resources/meta-gs/character/*/data.json",
        "resources/meta-gs/character/*/imgs/face.webp",
        "resources/meta-gs/character/*/imgs/face-q.webp",
        "resources/meta-gs/character/*/imgs/side.webp",
        "resources/meta-gs/character/*/imgs/gacha.webp",
        "resources/meta-gs/character/*/imgs/card.webp",
        "resources/meta-gs/character/*/imgs/banner.webp",
        "resources/meta-gs/character/*/imgs/splash.webp",
        "resources/meta-gs/character/*/icons/cons-*.webp",
        "resources/meta-gs/character/*/icons/passive-*.webp",
        "resources/meta-gs/character/*/icons/talent-[eq].webp",
        "resources/meta-gs/weapon/*/icon.webp",
        "resources/meta-gs/weapon/*/gacha.webp",
        "resources/meta-gs/artifact/imgs/*",
        "resources/meta-gs/material/data.json",
        "resources/meta-gs/material/talent/*",
        "resources/meta-gs/material/weapon/*",
        "resources/meta-gs/material/specialty/*",
        "resources/meta-gs/material/boss/*",
        "resources/meta-gs/material/weekly/*",
        "resources/meta-gs/material/normal/*",
        "resources/meta-gs/material/gem/*",
        # Character photos sent by 角色图片.
        "resources/character-img/*",
    ]),
    "yunzai-genshin": ("Yunzai-genshin", [
        "resources/font/tttgbnumber.ttf",
        "resources/font/HYWenHei-55W.ttf",
        "resources/img/abyss/*",
        "resources/img/other/*",
        "resources/img/combat/*",
        "resources/img/element/*",
        "resources/img/gacha/items/*",
        "resources/img/roleCard/bg1.jpg",
        "resources/img/deck/*",
        "resources/img/icon/check.webp",
        "resources/html/player/items/*",
        "resources/html/mysNews/iconfont.fb3712d.woff2",
        "resources/html/mysNews/mys.png",
        "resources/html/mysNews-list/蒙德.png",
        # Gacha link help pictures.
        "resources/logHelp/*",
    ]),
    "ark-plugin": ("ark-plugin", ["resources/graph/background.png", "resources/character/img/medal_*.png"]),
}

parser = argparse.ArgumentParser()
parser.add_argument("--references", type=pathlib.Path, required=True)
args = parser.parse_args()
refs = args.references.resolve()
root = pathlib.Path(__file__).resolve().parents[1]
game = json.loads((root / "internal/assets/game.json").read_text(encoding="utf-8"))
extensions = {source["id"]: tuple(source.get("extensions", [])) for source in game["artwork"]}

for source_id, (snapshot, patterns) in BUNDLES.items():
    base = refs / snapshot
    pinned = json.loads((refs / f"{snapshot}.source.json").read_text(encoding="utf-8"))
    names = [file.relative_to(base).as_posix() for file in base.rglob("*") if file.is_file() and ".git" not in file.relative_to(base).parts]
    # The download keeps the same extensions, so an update never drops a
    # bundled file type.
    names = [name for name in names if name.lower().endswith(extensions[source_id])]
    chosen = set()
    for pattern in patterns:
        matched = [name for name in names if fnmatch.fnmatchcase(name, pattern)]
        if not matched:
            raise SystemExit(f"{source_id}: {pattern} matches no file in {snapshot}")
        chosen.update(matched)
    target = root / "assets" / source_id
    shutil.rmtree(target, ignore_errors=True)
    size = 0
    for name in sorted(chosen):
        file = target / name
        file.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(base / name, file)
        size += file.stat().st_size
    record = {"commit": pinned["commit"], "archive": pinned["archive_url"], "files": len(chosen), "bytes": size}
    (root / "assets" / f"{source_id}.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    print(f"assets/{source_id}: {len(chosen)} files, {size / 1e6:.1f} MB at {pinned['commit'][:7]}")
