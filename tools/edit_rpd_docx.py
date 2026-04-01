import html
import io
import re
import zipfile
from dataclasses import dataclass
from pathlib import Path
import xml.etree.ElementTree as ET


NS = {
    "w": "http://schemas.openxmlformats.org/wordprocessingml/2006/main",
}


def _iter_text_nodes(elem: ET.Element):
    for t in elem.findall(".//w:t", NS):
        yield t


def _get_elem_text(elem: ET.Element) -> str:
    return "".join((t.text or "") for t in _iter_text_nodes(elem))


def _set_elem_text(elem: ET.Element, new_text: str) -> bool:
    """
    Replace text in elem by writing it into the first w:t and clearing the rest.
    Keeps existing run/paragraph structure and most formatting.
    """
    nodes = list(_iter_text_nodes(elem))
    if not nodes:
        return False
    nodes[0].text = new_text
    for n in nodes[1:]:
        n.text = ""
    return True


def _has_page_break(elem: ET.Element) -> bool:
    # <w:lastRenderedPageBreak/> is common; also handle <w:br w:type="page"/>
    if elem.find(".//w:lastRenderedPageBreak", NS) is not None:
        return True
    for br in elem.findall(".//w:br", NS):
        t = br.attrib.get(f"{{{NS['w']}}}type")
        if t == "page":
            return True
    return False


@dataclass(frozen=True)
class ReplaceRule:
    contains: str
    replacement: str


def _apply_contains_replacements(text: str, rules: list[ReplaceRule]) -> str:
    out = text
    for r in rules:
        if r.contains in out:
            out = out.replace(r.contains, r.replacement)
    return out


def main() -> None:
    in_docx = Path(r"D:\Downloads\Telegram Desktop\ИУ5-65Б Бердников РПЗ.docx")
    out_docx = Path(r"D:\Downloads\Telegram Desktop\ИУ5-65Б Бердников РПЗ_edited.docx")

    with zipfile.ZipFile(in_docx, "r") as zin:
        files = {name: zin.read(name) for name in zin.namelist()}

    doc_xml = files.get("word/document.xml")
    if not doc_xml:
        raise RuntimeError("word/document.xml not found")

    root = ET.fromstring(doc_xml)
    body = root.find("w:body", NS)
    assert body is not None

    # We won't touch anything before the first page break marker.
    after_title = False

    # High-signal structured replacements.
    # 1) Domain description + DB tables (replace the whole block textually).
    block_domain_old = (
        "Данные о подразделениях хранятся в таблице\u00A0departments\u00A0(таблица 1). "
        "Для хранения\u00A0состава одной заявки по подразделениям используется таблица\u00A0"
        "department_application_departments\u00A0(таблица 2). Таблица\u00A0department_applications\u00A0"
        "(таблица 3) представляет собой список всех заявок по подразделениям. "
        "Данные о\u00A0сотрудниках и руководителях\u00A0(модераторах заявок) хранятся в таблице\u00A0"
        "users\u00A0(таблица\u00A04)."
    )
    block_domain_new = (
        "Данные о межпланетных перелётах (услугах) хранятся в таблице transfer_routes (таблица 1). "
        "Для хранения состава одной заявки на расчёт межпланетного перелёта используется таблица "
        "flight_request_routes (таблица 2), реализующая связь многие-ко-многим между заявками и маршрутами. "
        "Таблица flight_requests (таблица 3) представляет собой список всех заявок на расчёт, включая статус "
        "черновика/сформированной/завершённой заявки и итоговую массу топлива. "
        "Данные о пользователях (создателях заявок) и модераторах хранятся в таблице users (таблица 4)."
    )

    # 2) Rename tables headings exactly as in doc
    table_heading_map = {
        "Таблица 1 – Таблица departments": "Таблица 1 – Таблица transfer_routes",
        "Таблица 2 – Таблица department_applications_department": "Таблица 2 – Таблица flight_request_routes",
        "Таблица 3 – Таблица department_applications": "Таблица 3 – Таблица flight_requests",
        "Таблица 4 – Таблица users": "Таблица 4 – Таблица users",
    }

    # 3) Replace table row content by exact attribute names (safe in cells)
    # We'll later do targeted per-cell substitutions.
    cell_replacements = [
        ReplaceRule("department_id", "id"),
    ]

    # 4) Replace calculation block with explicit formulas and computed numbers
    calc_old_prefix = "На основе анализа структурных\u00A0данных и предполагаемой нагрузки был произведён расчёт аппаратных требований"
    calc_new_block = (
        "На основе анализа структурных данных и предполагаемой нагрузки был произведён расчёт аппаратных требований "
        "для системы заявок на расчёт межпланетного перелёта. Система рассчитана на одновременную работу до 100 пользователей "
        "(создателей заявок) и 10 модераторов.\n"
        "Пусть один пользователь формирует 30 заявок в день, рабочий день длится 8 часов. Тогда число заявок в сутки:\n"
        "N_requests_day = N_users · N_per_user = 100 · 30 = 3000.\n"
        "На каждую заявку по диаграмме последовательности приходится 14 HTTP-запросов. Тогда средний поток запросов от пользователей:\n"
        "RPS_users = (N_requests_day · N_http_per_request) / (T_work · 3600) = (3000 · 14) / (8 · 3600) ≈ 1.46 rps.\n"
        "С учётом polling от модераторов примем дополнительно 10 rps:\n"
        "RPS_total = RPS_users + RPS_poll = 1.46 + 10 = 11.46 rps.\n"
        "Рассчитаем число ядер при k_reserve = 1.5 и пропускной способности одного ядра 50 rps:\n"
        "C = ceil(RPS_total / 50 · k_reserve) = ceil(11.46 / 50 · 1.5) = 1.\n"
        "Оперативная память (4 ГБ на ядро): RAM = 4 · C = 4 ГБ.\n"
        "Оценим прирост БД. Размер записи flight_requests (id 8, status 16, created_at 8, created_by 8, formed_at 8, "
        "completed_at 8, moderated_by 8, spacecraft_dry_mass_kg 8, engine_isp_sec 8, total_fuel_mass_kg 8) ≈ 88 байт. "
        "Размер записи flight_request_routes (flight_request_id 8, route_id 8, quantity 4, segment_order 4, is_primary 1, "
        "payload_mass_kg 8, delta_v_kms 8, fuel_mass_kg 8, segment_dry_mass_kg 8, segment_isp_sec 8) ≈ 72 байт.\n"
        "Примем среднее число сегментов (маршрутов) в заявке 3, тогда в сутки вставляется 3000 записей flight_requests и 9000 "
        "записей flight_request_routes. Годовой объём данных:\n"
        "V_year ≈ 365 · (3000 · 88 + 9000 · 72) ≈ 365 · (264000 + 648000) ≈ 332 880 000 байт ≈ 0.31 ГБ.\n"
        "За 5 лет: V_5years ≈ 1.55 ГБ. С запасом N_reserve = 2 ГБ примем требуемый объём диска 4 ГБ."
    )

    # 5) Update goal/назначение in ТЗ section to match interplanetary energy theme.
    tz_goal_old = (
        "Необходимо разработать систему, включающую в себя вебсервис, вебприложение и\u00A0нативное приложение, "
        "которая позволит сотрудникам формировать и отправлять заявки по подразделениям организации."
    )
    tz_goal_new = (
        "Необходимо разработать систему, включающую веб‑сервис, веб‑приложение и нативное приложение, "
        "которая позволит пользователям формировать заявки на расчёт параметров межпланетного перелёта, "
        "включая оценку энергии, необходимой для выполнения перелёта."
    )

    tz_purpose_old_prefix = "Необходимо разработать систему, которая будет автоматизировать работу с заявками по\u00A0подразделениям"
    tz_purpose_new = (
        "Необходимо разработать систему, которая будет автоматизировать работу с заявками на расчёт межпланетного перелёта "
        "и предоставлять актуальную информацию о них. Пользователи смогут создавать заявки, выбирая межпланетные маршруты "
        "и задавая параметры аппарата (масса сухого аппарата, удельный импульс), после чего система рассчитает "
        "характеристическую скорость (Δv), массу топлива и энергию. Модератор сможет принять заявку к рассмотрению, "
        "завершить её либо отклонить. Гости системы имеют доступ к списку маршрутов и могут пройти регистрацию для "
        "последующей работы с заявками."
    )

    # 6) Replace HTTP methods list with actual API routes (Swagger basePath=/api)
    http_methods_old_start = "Методы HTTP"
    http_methods_new_block = (
        "Методы HTTP\n"
        "POST Зарегистрировать пользователя (POST /api/interplanetaryflightusers/register)\n"
        "POST Аутентификация пользователя (POST /api/interplanetaryflightusers/login)\n"
        "POST Деавторизация пользователя (POST /api/interplanetaryflightusers/logout)\n"
        "GET Получить список межпланетных перелётов (GET /api/interplanetaryflights)\n"
        "GET Получить один межпланетный перелёт (GET /api/interplanetaryflights/:id)\n"
        "POST Создать межпланетный перелёт (POST /api/interplanetaryflights)\n"
        "POST Добавить маршрут в черновик заявки (POST /api/interplanetaryflightrequests/draft/items)\n"
        "GET Получить иконку корзины (GET /api/interplanetaryflightrequests/cart-icon)\n"
        "GET Получить список заявок (GET /api/interplanetaryflightrequests)\n"
        "GET Получить одну заявку с расчётами (GET /api/interplanetaryflightrequests/:id)\n"
        "PUT Изменить поля черновика заявки (PUT /api/interplanetaryflightrequests/:id)\n"
        "PUT Сформировать заявку (PUT /api/interplanetaryflightrequests/:id/form)\n"
        "PUT Модерация заявки (PUT /api/interplanetaryflightrequests/:id/moderate)\n"
        "DELETE Удалить черновик заявки (DELETE /api/interplanetaryflightrequests/:id)\n"
        "DELETE Удалить маршрут из заявки (DELETE /api/interplanetaryflightrequests/:id/items/:routeId)\n"
        "PUT Обновить строку м‑м (PUT /api/interplanetaryflightrequests/:id/items/:routeId)"
    )

    # Iterate over body children; modify paragraphs and table cells after title page.
    children = list(body)
    for child in children:
        if not after_title and _has_page_break(child):
            after_title = True

        if not after_title:
            continue

        tag = child.tag.rsplit("}", 1)[-1]
        if tag == "p":
            text = _get_elem_text(child)
            if not text:
                continue

            if block_domain_old in text:
                _set_elem_text(child, text.replace(block_domain_old, block_domain_new))
                continue

            if text in table_heading_map:
                _set_elem_text(child, table_heading_map[text])
                continue

            if text.startswith(calc_old_prefix):
                _set_elem_text(child, calc_new_block)
                # Clear subsequent calculation placeholder paragraphs until the next figure heading.
                # (The template keeps old "(1)…(9)" paragraphs after the intro.)
                idx = children.index(child)
                j = idx + 1
                while j < len(children):
                    n = children[j]
                    if n.tag.rsplit("}", 1)[-1] != "p":
                        j += 1
                        continue
                    t = _get_elem_text(n).strip()
                    if t.startswith("Рисунок 3"):
                        break
                    _set_elem_text(n, "")
                    j += 1
                continue

            if text == tz_goal_old:
                _set_elem_text(child, tz_goal_new)
                continue

            if text.startswith(tz_purpose_old_prefix):
                _set_elem_text(child, tz_purpose_new)
                continue

            if text == http_methods_old_start:
                _set_elem_text(child, http_methods_new_block)
                continue

        # Tables are rewritten in a dedicated pass below.

    def set_table(table: ET.Element, rows: list[list[str]]) -> None:
        # Expect first row to be header (Атрибут/Тип данных/Описание)
        trs = [tr for tr in list(table) if tr.tag.rsplit("}", 1)[-1] == "tr"]
        if not trs:
            return
        header = trs[0]
        data_trs = trs[1:]

        # Ensure enough rows
        while len(data_trs) < len(rows):
            # clone last data row or header if none
            src = data_trs[-1] if data_trs else header
            table.append(ET.fromstring(ET.tostring(src, encoding="utf-8")))
            trs = [tr for tr in list(table) if tr.tag.rsplit("}", 1)[-1] == "tr"]
            data_trs = trs[1:]

        # Remove extra rows
        for tr in data_trs[len(rows) :]:
            table.remove(tr)
        trs = [tr for tr in list(table) if tr.tag.rsplit("}", 1)[-1] == "tr"]
        data_trs = trs[1:]

        for tr, row in zip(data_trs, rows, strict=False):
            tcs = [tc for tc in tr.findall("./w:tc", NS)]
            for col_idx, value in enumerate(row[: len(tcs)]):
                _set_elem_text(tcs[col_idx], value)

    # Rewrite the 4 DB tables by locating headings and the following table element.
    db_tables = {
        "Таблица 1 – Таблица transfer_routes": [
            ["id", "BIGSERIAL", "Первичный ключ, идентификатор маршрута"],
            ["title", "VARCHAR(64)", "Название маршрута"],
            ["from_body", "VARCHAR(64)", "Тело отправления"],
            ["to_body", "VARCHAR(64)", "Тело назначения"],
            ["description", "TEXT", "Полное описание маршрута"],
            ["is_deleted", "BOOLEAN", "Флаг «мягкого удаления» маршрута"],
            ["image_url", "VARCHAR(512)", "Ссылка на изображение маршрута"],
            ["video_url", "VARCHAR(512)", "Ссылка на видео маршрута"],
            ["from_orbit_radius_km", "DOUBLE PRECISION", "Радиус орбиты отправления, км"],
            ["to_orbit_radius_km", "DOUBLE PRECISION", "Радиус орбиты назначения, км"],
        ],
        "Таблица 2 – Таблица flight_request_routes": [
            ["flight_request_id", "BIGINT", "Идентификатор заявки"],
            ["route_id", "BIGINT", "Идентификатор маршрута"],
            ["quantity", "INTEGER", "Количество (по умолчанию 1)"],
            ["segment_order", "INTEGER", "Порядок сегмента в заявке"],
            ["is_primary", "BOOLEAN", "Признак основного сегмента"],
            ["payload_mass_kg", "DOUBLE PRECISION", "Масса полезной нагрузки, кг (опционально)"],
            ["delta_v_kms", "DOUBLE PRECISION", "Рассчитанная Δv сегмента (м/с), хранится при формировании"],
            ["fuel_mass_kg", "DOUBLE PRECISION", "Рассчитанная масса топлива для сегмента, кг"],
            ["segment_dry_mass_kg", "DOUBLE PRECISION", "Масса сухого аппарата для сегмента, кг"],
            ["segment_isp_sec", "DOUBLE PRECISION", "Удельный импульс для сегмента, с"],
        ],
        "Таблица 3 – Таблица flight_requests": [
            ["id", "BIGSERIAL", "Первичный ключ, идентификатор заявки"],
            ["status", "VARCHAR(16)", "Статус заявки (draft/formed/completed/rejected/deleted)"],
            ["created_at", "TIMESTAMPTZ", "Дата и время создания заявки"],
            ["created_by", "BIGINT", "Идентификатор создателя заявки"],
            ["formed_at", "TIMESTAMPTZ", "Дата и время формирования заявки"],
            ["completed_at", "TIMESTAMPTZ", "Дата и время завершения обработки заявки"],
            ["moderated_by", "BIGINT", "Идентификатор модератора"],
            ["spacecraft_dry_mass_kg", "DOUBLE PRECISION", "Масса сухого аппарата, кг"],
            ["engine_isp_sec", "DOUBLE PRECISION", "Удельный импульс двигателя, с"],
            ["total_fuel_mass_kg", "DOUBLE PRECISION", "Итоговая масса топлива по заявке, кг"],
        ],
        "Таблица 4 – Таблица users": [
            ["id", "BIGSERIAL", "Первичный ключ, идентификатор пользователя"],
            ["username", "VARCHAR(64)", "Логин пользователя"],
            ["password_hash", "TEXT", "Пароль/хэш (учебный вариант хранения)"],
            ["is_moderator", "BOOLEAN", "Флаг «является ли пользователь модератором»"],
        ],
    }

    after_title_db = False
    for idx, child in enumerate(children):
        if not after_title_db and _has_page_break(child):
            after_title_db = True
        if not after_title_db:
            continue
        if child.tag.rsplit("}", 1)[-1] != "p":
            continue
        heading = _get_elem_text(child).strip()
        if heading not in db_tables:
            continue
        # Find next table
        j = idx + 1
        while j < len(children) and children[j].tag.rsplit("}", 1)[-1] != "tbl":
            j += 1
        if j < len(children) and children[j].tag.rsplit("}", 1)[-1] == "tbl":
            set_table(children[j], db_tables[heading])

    # Replace generic domain word occurrences after title (safe, limited set)
    contains_rules = [
        ReplaceRule("заявок по подразделениям", "заявок на расчёт межпланетного перелёта"),
        ReplaceRule("заявки по подразделениям", "заявки на расчёт межпланетного перелёта"),
        ReplaceRule("подразделений", "межпланетных маршрутов"),
        ReplaceRule("подразделения", "маршрута"),
        ReplaceRule("подразделение", "маршрут"),
        ReplaceRule("администратору", "модератору"),
        ReplaceRule("администратор", "модератор"),

        # Old API names from template -> current lab4 API.
        ReplaceRule("/api/users/signup", "/api/interplanetaryflightusers/register"),
        ReplaceRule("/api/users/signin", "/api/interplanetaryflightusers/login"),
        ReplaceRule("/api/users/signout", "/api/interplanetaryflightusers/logout"),
        ReplaceRule("/api/departments", "/api/interplanetaryflights"),
        ReplaceRule("/api/department/{id}", "/api/interplanetaryflights/{id}"),
        ReplaceRule("/api/department/create-department", "/api/interplanetaryflights"),
        ReplaceRule("/api/dep_app_dep/add/{department_id}", "/api/interplanetaryflightrequests/draft/items"),
        ReplaceRule("/api/department_application/department_application-cart", "/api/interplanetaryflightrequests/cart-icon"),
        ReplaceRule("/api/department_application/all-department_applications", "/api/interplanetaryflightrequests"),
        ReplaceRule("/api/department_application/{id}", "/api/interplanetaryflightrequests/{id}"),
        ReplaceRule("/api/department_application/{id}/edit-department_application", "/api/interplanetaryflightrequests/{id}"),
        ReplaceRule("/api/department_application/{id}/form-department_application", "/api/interplanetaryflightrequests/{id}/form"),
        ReplaceRule("/api/department_application/{id}/delete-department_application", "/api/interplanetaryflightrequests/{id}"),
    ]

    # Apply to all paragraphs after title by checking page-break flag again and mutating their text nodes.
    after_title = False
    for child in list(body):
        if not after_title and _has_page_break(child):
            after_title = True
        if not after_title:
            continue
        for t in child.findall(".//w:t", NS):
            if not t.text:
                continue
            t.text = _apply_contains_replacements(t.text, contains_rules)

    # Regex-level fixes for appendix method tables (strings may be split across multiple w:t nodes).
    regex_rules: list[tuple[re.Pattern[str], str]] = [
        (re.compile(r"/api/departments"), "/api/interplanetaryflights"),
        (re.compile(r"/api/department/\{id\}"), "/api/interplanetaryflights/{id}"),
        (re.compile(r"/api/department/create-department"), "/api/interplanetaryflights"),
        (re.compile(r"/api/dep_app_dep/add/\{department_id\}"), "/api/interplanetaryflightrequests/draft/items"),
        (re.compile(r"/api/dep_app_dep/\{department_id\}/\{department_application_id\}"), "/api/interplanetaryflightrequests/{id}/items/{routeId}"),
        (re.compile(r"/api/department_application/department_application-cart"), "/api/interplanetaryflightrequests/cart-icon"),
        (re.compile(r"/api/department_application/all-department_applications"), "/api/interplanetaryflightrequests"),
        (re.compile(r"/api/department_application/\{id\}/edit-department_application"), "/api/interplanetaryflightrequests/{id}"),
        (re.compile(r"/api/department_application/\{id\}/form-department_application"), "/api/interplanetaryflightrequests/{id}/form"),
        (re.compile(r"/api/department_application/\{id\}/finish-department_application"), "/api/interplanetaryflightrequests/{id}/moderate"),
        (re.compile(r"/api/department_application/\{id\}/delete-department_application"), "/api/interplanetaryflightrequests/{id}"),
        (re.compile(r"/api/department_application/\{id\}"), "/api/interplanetaryflightrequests/{id}"),
        (re.compile(r"/api/users/signup"), "/api/interplanetaryflightusers/register"),
        (re.compile(r"/api/users/signin"), "/api/interplanetaryflightusers/login"),
        (re.compile(r"/api/users/signout"), "/api/interplanetaryflightusers/logout"),
    ]

    def apply_regex_to_elem(elem: ET.Element) -> None:
        text = _get_elem_text(elem)
        if not text:
            return
        new_text = text
        for pat, repl in regex_rules:
            new_text = pat.sub(repl, new_text)
        if new_text != text:
            _set_elem_text(elem, new_text)

    after_title = False
    for child in list(body):
        if not after_title and _has_page_break(child):
            after_title = True
        if not after_title:
            continue
        tag = child.tag.rsplit("}", 1)[-1]
        if tag == "p":
            apply_regex_to_elem(child)
        elif tag == "tbl":
            for tc in child.findall(".//w:tc", NS):
                apply_regex_to_elem(tc)

    # Fix HTTP methods block: in this template it's a list of separate paragraphs.
    # We'll replace the whole section by finding the paragraph "Методы HTTP" and clearing
    # until "Меню", then inserting the new block into that paragraph.
    def rewrite_http_section() -> None:
        nonlocal body
        after_title_local = False
        children = list(body)
        for idx, child in enumerate(children):
            if not after_title_local and _has_page_break(child):
                after_title_local = True
            if not after_title_local:
                continue
            if child.tag.rsplit("}", 1)[-1] != "p":
                continue
            if _get_elem_text(child).strip() != "Методы HTTP":
                continue
            # Found start
            _set_elem_text(child, http_methods_new_block)
            # Clear following paragraphs until we hit "Меню"
            j = idx + 1
            while j < len(children):
                n = children[j]
                if n.tag.rsplit("}", 1)[-1] == "p":
                    txt = _get_elem_text(n).strip()
                    if txt == "Меню":
                        break
                    # clear old method bullets (the template repeats them as separate paragraphs)
                    if txt.startswith(("GET", "POST", "PUT", "DELETE", "GET\u00A0", "POST\u00A0", "PUT\u00A0", "DELETE\u00A0")):
                        _set_elem_text(n, "")
                    elif txt in {"Методы HTTP", ""}:
                        _set_elem_text(n, "")
                j += 1
            break

    rewrite_http_section()

    # Serialize back
    new_doc_xml = ET.tostring(root, encoding="utf-8", xml_declaration=True)
    files["word/document.xml"] = new_doc_xml

    if out_docx.exists():
        out_docx.unlink()
    with zipfile.ZipFile(out_docx, "w", compression=zipfile.ZIP_DEFLATED) as zout:
        for name, content in files.items():
            zout.writestr(name, content)

    print("Wrote", out_docx)


if __name__ == "__main__":
    main()

