import struct
import zlib
from pathlib import Path


SIG_LOCAL = b"PK\x03\x04"
SIG_CDIR = b"PK\x01\x02"
SIG_EOCD = b"PK\x05\x06"


def _try_decode_filename(raw: bytes) -> str | None:
    for enc in ("utf-8", "cp1251", "latin-1"):
        try:
            s = raw.decode(enc)
        except Exception:
            continue
        # sanity: docx zip entries are mostly ascii-ish
        if "\x00" in s:
            continue
        return s
    return None


def _safe_join(base: Path, rel: str) -> Path:
    rel = rel.replace("\\", "/")
    rel = rel.lstrip("/").replace("../", "")
    return base / rel


def _find_eocd(data: bytes) -> int | None:
    # EOCD is within last 64KB + comment
    start = max(0, len(data) - (65536 + 22))
    idx = data.rfind(SIG_EOCD, start)
    return idx if idx >= 0 else None


def _parse_central_directory(data: bytes) -> list[dict]:
    """
    Returns list of entries with: name, comp_method, csize, usize, local_off.
    Uses EOCD when possible, otherwise scans for central directory signatures.
    """
    entries: list[dict] = []
    eocd = _find_eocd(data)
    if eocd is not None and eocd + 22 <= len(data):
        # EOCD: sig(4) disk(2) cdisk(2) ndisk(2) ntotal(2) cdsize(4) cdoff(4) comlen(2)
        _, _, _, _, _, cdsize, cdoff, comlen = struct.unpack_from("<4sHHHHIIH", data, eocd)
        if 0 <= cdoff < len(data) and 0 < cdsize <= len(data) - cdoff:
            cd_start = cdoff
            cd_end = cdoff + cdsize
            pos = cd_start
            while pos + 46 <= cd_end and data[pos : pos + 4] == SIG_CDIR:
                (
                    _sig,
                    _ver_made,
                    _ver_need,
                    _flag,
                    comp,
                    _mtime,
                    _mdate,
                    _crc,
                    csize,
                    usize,
                    fnl,
                    exl,
                    coml,
                    _disk,
                    _iattr,
                    _eattr,
                    loff,
                ) = struct.unpack_from("<4sHHHHHHIIIHHHHHII", data, pos)
                name_start = pos + 46
                name_end = name_start + fnl
                extra_end = name_end + exl
                comment_end = extra_end + coml
                if comment_end > cd_end:
                    break
                name = _try_decode_filename(data[name_start:name_end])
                if name:
                    entries.append(
                        dict(
                            name=name,
                            comp=comp,
                            csize=csize,
                            usize=usize,
                            loff=loff,
                        )
                    )
                pos = comment_end
            if entries:
                return entries

    # Fallback: scan entire file for central directory entries
    pos = 0
    while True:
        idx = data.find(SIG_CDIR, pos)
        if idx < 0 or idx + 46 > len(data):
            break
        try:
            (
                _sig,
                _ver_made,
                _ver_need,
                _flag,
                comp,
                _mtime,
                _mdate,
                _crc,
                csize,
                usize,
                fnl,
                exl,
                coml,
                _disk,
                _iattr,
                _eattr,
                loff,
            ) = struct.unpack_from("<4sHHHHHHIIIHHHHHII", data, idx)
        except struct.error:
            pos = idx + 4
            continue
        name_start = idx + 46
        name_end = name_start + fnl
        extra_end = name_end + exl
        comment_end = extra_end + coml
        if comment_end > len(data):
            pos = idx + 4
            continue
        name = _try_decode_filename(data[name_start:name_end])
        if name and len(name) <= 400 and all(c not in name for c in "\r\n"):
            entries.append(dict(name=name, comp=comp, csize=csize, usize=usize, loff=loff))
        pos = idx + 4

    # Deduplicate by name keeping first
    uniq: dict[str, dict] = {}
    for e in entries:
        uniq.setdefault(e["name"], e)
    return list(uniq.values())


def _extract_from_local(data: bytes, loff: int, csize: int) -> tuple[str | None, int, int, bytes] | None:
    if loff < 0 or loff + 30 > len(data) or data[loff : loff + 4] != SIG_LOCAL:
        return None
    try:
        (
            _sig,
            _ver,
            _flag,
            comp,
            _mtime,
            _mdate,
            _crc,
            _csize_lh,
            _usize_lh,
            fnl,
            exl,
        ) = struct.unpack_from("<4sHHHHHIIIHH", data, loff)
    except struct.error:
        return None
    name_start = loff + 30
    name_end = name_start + fnl
    extra_end = name_end + exl
    if extra_end > len(data):
        return None
    name = _try_decode_filename(data[name_start:name_end])
    file_start = extra_end
    file_end = file_start + csize
    if file_end > len(data) or csize < 0:
        return None
    return name, comp, file_start, data[file_start:file_end]


def recover_docx(input_path: Path, output_path: Path) -> tuple[int, list[str]]:
    data = input_path.read_bytes()
    extracted: dict[str, bytes] = {}
    errors: list[str] = []

    cd_entries = _parse_central_directory(data)
    if not cd_entries:
        raise RuntimeError("Central directory not found; cannot recover.")

    for e in cd_entries:
        name = e["name"]
        csize = int(e["csize"])
        loff = int(e["loff"])
        comp = int(e["comp"])

        info = _extract_from_local(data, loff, csize)
        if not info:
            errors.append(f"Local header not found for {name}")
            continue
        local_name, local_comp, _file_start, comp_bytes = info
        if local_name and local_name != name:
            # prefer central directory name
            pass
        if local_comp != comp:
            comp = local_comp

        if name.endswith("/"):
            extracted.setdefault(name, b"")
            continue

        try:
            if comp == 0:
                out = comp_bytes
            elif comp == 8:
                out = zlib.decompress(comp_bytes, -zlib.MAX_WBITS)
            else:
                errors.append(f"Unsupported compression {comp} for {name}")
                continue
        except Exception as ex:
            errors.append(f"Decompress failed for {name}: {ex}")
            continue

        extracted[name] = out

    if not extracted:
        raise RuntimeError("No entries recovered from file.")

    # Rebuild a correct docx zip
    import zipfile

    output_path.parent.mkdir(parents=True, exist_ok=True)
    if output_path.exists():
        output_path.unlink()

    # Ensure directories are present in zip in a stable order
    names = sorted(extracted.keys(), key=lambda s: (s.count("/"), s))
    with zipfile.ZipFile(output_path, "w", compression=zipfile.ZIP_DEFLATED) as z:
        for name in names:
            if name.endswith("/"):
                z.writestr(name, b"")
                continue
            z.writestr(name, extracted[name])

    return len(extracted), errors


def main() -> None:
    # Work on an ASCII-only copy to avoid OneDrive locks/encoding issues.
    input_docx = Path(r"D:\Temp\rpd.docx")
    out_docx = Path(
        r"C:\Users\admin\OneDrive\Desktop\Education\RIP_space_2026\Отчёты\ИУ5-65Б Крылов РПЗ_RECOVERED.docx"
    )

    count, errors = recover_docx(input_docx, out_docx)
    print(f"Recovered entries: {count}")
    if errors:
        print("Errors:")
        for e in errors[:30]:
            print(" -", e)
        if len(errors) > 30:
            print(f" ... {len(errors)-30} more")
    print("Output:", out_docx)


if __name__ == "__main__":
    main()

