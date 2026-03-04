-- Seed data for Lab 2 demo

INSERT INTO users (username, password_hash, is_moderator)
VALUES
  ('demo', 'demo', FALSE),
  ('moderator', 'moderator', TRUE)
ON CONFLICT (username) DO NOTHING;

-- Note: store Minio object keys (not full URLs) for simplicity
INSERT INTO transfer_routes
  (title, from_body, to_body, description, is_deleted, image_url, video_url, from_orbit_radius_km, to_orbit_radius_km)
VALUES
  (
    'Юпитер', 'Земля', 'Юпитер',
    'Упрощённый перелёт к газовому гиганту. Модель: переход Гомана вокруг Солнца (без гравитационных манёвров, без учёта наклонения орбит).',
    FALSE, 'Jupiter.jpeg', 'Jupiter_vid.mp4',
    149600000.0, 778500000.0
  ),
  (
    'Сатурн', 'Земля', 'Сатурн',
    'Дальняя цель: перелёт к Сатурну. В реальности часто используют гравитационные манёвры (как у Voyager), но здесь считаем идеализированный переход Гомана.',
    FALSE, 'Saturn.jpg', 'Saturn_vid.mp4',
    149600000.0, 1433500000.0
  ),
  (
    'Уран', 'Земля', 'Уран',
    'Перелёт к ледяному гиганту (идеализированная гелиоцентрическая траектория).',
    FALSE, 'Uranus.jpg', 'Uranus_vid.mp4',
    149600000.0, 2872500000.0
  ),
  (
    'Нептун', 'Земля', 'Нептун',
    'Одна из самых «дорогих» целей по Δv в упрощённой модели. Voyager 2 долетел до Нептуна с помощью гравитационных манёвров — здесь их не учитываем.',
    FALSE, 'Neptune.jpg', 'Neptune_vid.mp4',
    149600000.0, 4495100000.0
  )
ON CONFLICT DO NOTHING;

