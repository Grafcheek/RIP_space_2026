import zipfile
from pathlib import Path
import xml.etree.ElementTree as ET


NS = {
    "w": "http://schemas.openxmlformats.org/wordprocessingml/2006/main",
}


def iter_paragraph_text(p: ET.Element) -> str:
    parts: list[str] = []
    for t in p.findall(".//w:t", NS):
        if t.text:
            parts.append(t.text)
    return "".join(parts).strip()


def main() -> None:
    docx_path = Path(r"D:\Downloads\Telegram Desktop\ИУ5-65Б Бердников РПЗ.docx")
    out_txt = Path(r"D:\Downloads\Telegram Desktop\_rpd_text_clean.txt")

    with zipfile.ZipFile(docx_path) as z:
        xml = z.read("word/document.xml")

    root = ET.fromstring(xml)
    body = root.find("w:body", NS)
    assert body is not None

    lines: list[str] = []
    for child in list(body):
        tag = child.tag.rsplit("}", 1)[-1]
        if tag == "p":
            text = iter_paragraph_text(child)
            if text:
                lines.append(text)
        elif tag == "tbl":
            # Represent tables as rows with cell separators
            for tr in child.findall(".//w:tr", NS):
                cells = []
                for tc in tr.findall("./w:tc", NS):
                    cell_texts = []
                    for p in tc.findall(".//w:p", NS):
                        t = iter_paragraph_text(p)
                        if t:
                            cell_texts.append(t)
                    cells.append(" ".join(cell_texts).strip())
                if any(cells):
                    lines.append(" | ".join(cells))
            lines.append("")  # blank line after table

    out_txt.write_text("\n".join(lines), encoding="utf-8")
    print("Wrote", out_txt)


if __name__ == "__main__":
    main()

