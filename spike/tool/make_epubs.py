"""Generate small test EPUBs: Persian RTL reflowable, fixed-layout RTL, plain English."""
import sys
import zipfile
from pathlib import Path

out = Path(sys.argv[1])
out.mkdir(parents=True, exist_ok=True)

CONTAINER = """<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>"""


def write_epub(name, opf, files):
    with zipfile.ZipFile(out / name, "w") as z:
        z.writestr("mimetype", "application/epub+zip", compress_type=zipfile.ZIP_STORED)
        z.writestr("META-INF/container.xml", CONTAINER)
        z.writestr("OEBPS/content.opf", opf)
        for path, data in files.items():
            z.writestr("OEBPS/" + path, data)


def nav(title, items, lang):
    lis = "".join(f'<li><a href="{h}">{t}</a></li>' for h, t in items)
    return f"""<?xml version="1.0" encoding="utf-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="{lang}">
<head><title>{title}</title></head>
<body><nav epub:type="toc"><ol>{lis}</ol></nav></body></html>"""


def chapter(title, paras, lang, direction):
    ps = "".join(f"<p>{p}</p>" for p in paras)
    return f"""<?xml version="1.0" encoding="utf-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" lang="{lang}" dir="{direction}">
<head><title>{title}</title><meta charset="utf-8"/></head>
<body><h1>{title}</h1>{ps}</body></html>"""


# 1. Persian reflowable RTL
fa_par = "ای صبا نکهتی از خاک ره یار بیار؛ ۱۲۳۴۵۶۷۸۹۰ این یک متن آزمایشی فارسی است برای بررسی چینش راست‌به‌چپ و صفحه‌بندی. " * 12
fa_ch = [(f"ch{i}.xhtml", f"فصل {i}") for i in range(1, 6)]
files = {f: chapter(t, [fa_par] * 6, "fa", "rtl") for f, t in fa_ch}
files["nav.xhtml"] = nav("فهرست", fa_ch, "fa")
manifest = '<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>' + "".join(
    f'<item id="c{i}" href="{f}" media-type="application/xhtml+xml"/>' for i, (f, _) in enumerate(fa_ch, 1))
spine = "".join(f'<itemref idref="c{i}"/>' for i in range(1, len(fa_ch) + 1))
write_epub("persian.epub", f"""<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:identifier id="id">wepub-test-fa</dc:identifier><dc:title>کتاب آزمایشی</dc:title>
<dc:language>fa</dc:language><dc:creator>wepub</dc:creator>
<meta property="dcterms:modified">2026-01-01T00:00:00Z</meta></metadata>
<manifest>{manifest}</manifest><spine page-progression-direction="rtl">{spine}</spine></package>""", files)

# 2. Fixed-layout RTL (pre-paginated)
fx_pages = [(f"p{i}.xhtml", f"Page {i}") for i in range(1, 5)]
colors = ["#f4c7c3", "#c3e0f4", "#d5f4c3", "#f4efc3"]
files = {}
for i, (f, t) in enumerate(fx_pages):
    files[f] = f"""<?xml version="1.0" encoding="utf-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>{t}</title><meta name="viewport" content="width=800, height=1200"/>
<style>html,body{{margin:0;padding:0;width:800px;height:1200px;background:{colors[i]}}}
.a{{position:absolute;left:100px;top:100px;font:48px sans-serif}}
.b{{position:absolute;right:60px;bottom:60px;width:300px;height:300px;border:8px solid #333}}</style></head>
<body><div class="a">{t} — fixed layout</div><div class="b"></div></body></html>"""
files["nav.xhtml"] = nav("Pages", fx_pages, "en")
manifest = '<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>' + "".join(
    f'<item id="c{i}" href="{f}" media-type="application/xhtml+xml"/>' for i, (f, _) in enumerate(fx_pages, 1))
spine = "".join(f'<itemref idref="c{i}" properties="page-spread-{"left" if i % 2 == 0 else "right"}"/>' for i in range(1, len(fx_pages) + 1))
write_epub("fixed_rtl.epub", f"""<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:identifier id="id">wepub-test-fx</dc:identifier><dc:title>Fixed RTL test</dc:title>
<dc:language>ar</dc:language>
<meta property="rendition:layout">pre-paginated</meta><meta property="rendition:spread">landscape</meta>
<meta property="dcterms:modified">2026-01-01T00:00:00Z</meta></metadata>
<manifest>{manifest}</manifest><spine page-progression-direction="rtl">{spine}</spine></package>""", files)

# 3. Plain English
en_par = "It was a quiet evening and the lamps along the street were lit one by one. " * 14
en_ch = [(f"ch{i}.xhtml", f"Chapter {i}") for i in range(1, 6)]
files = {f: chapter(t, [en_par] * 6, "en", "ltr") for f, t in en_ch}
files["nav.xhtml"] = nav("Contents", en_ch, "en")
manifest = '<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>' + "".join(
    f'<item id="c{i}" href="{f}" media-type="application/xhtml+xml"/>' for i, (f, _) in enumerate(en_ch, 1))
spine = "".join(f'<itemref idref="c{i}"/>' for i in range(1, len(en_ch) + 1))
write_epub("plain.epub", f"""<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:identifier id="id">wepub-test-en</dc:identifier><dc:title>Plain test</dc:title>
<dc:language>en</dc:language><dc:creator>wepub</dc:creator>
<meta property="dcterms:modified">2026-01-01T00:00:00Z</meta></metadata>
<manifest>{manifest}</manifest><spine>{spine}</spine></package>""", files)
print("ok", sorted(p.name for p in out.iterdir()))
