import re
import zipfile
from pathlib import Path


def main() -> None:
    docx_path = Path(
        r"C:\Users\admin\OneDrive\Desktop\Education\RIP_space_2026\Отчёты\ИУ5-65Б Крылов РПЗ.docx"
    )
    with zipfile.ZipFile(docx_path) as z:
        xml = z.read("word/document.xml").decode("utf-8", errors="ignore")

    texts = re.findall(r"<w:t[^>]*>(.*?)</w:t>", xml)
    joined = "\n".join(
        t.replace("&lt;", "<").replace("&gt;", ">").replace("&amp;", "&") for t in texts
    )
    print(f"TEXT_LINES={len(texts)}")

    for kw in ["Лаборат", "swagger", "JWT", "заяв", "межплан", "Крылов", "ИУ5", "Redis"]:
        if kw.lower() not in joined.lower():
            continue
        print(f"\n--- matches for {kw} ---")
        pat = re.compile(r"(.{0,80}" + re.escape(kw) + r".{0,80})", re.I)
        for i, m in enumerate(pat.finditer(joined)):
            if i >= 15:
                break
            print(m.group(1).replace("\n", " "))


if __name__ == "__main__":
    main()

