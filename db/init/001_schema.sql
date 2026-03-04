-- Lab 2 schema (PostgreSQL)
-- No cascade deletes. Soft delete via status / is_deleted flags.

CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR(64) NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  is_moderator BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS transfer_routes (
  id BIGSERIAL PRIMARY KEY,
  title VARCHAR(64) NOT NULL,
  from_body VARCHAR(64) NOT NULL,
  to_body VARCHAR(64) NOT NULL,
  description TEXT NOT NULL,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,

  image_url VARCHAR(512) NULL,
  video_url VARCHAR(512) NULL,

  from_orbit_radius_km DOUBLE PRECISION NOT NULL,
  to_orbit_radius_km DOUBLE PRECISION NOT NULL
);

CREATE TABLE IF NOT EXISTS flight_requests (
  id BIGSERIAL PRIMARY KEY,
  status VARCHAR(16) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

  formed_at TIMESTAMPTZ NULL,
  completed_at TIMESTAMPTZ NULL,
  moderated_by BIGINT NULL REFERENCES users(id) ON DELETE RESTRICT,

  spacecraft_dry_mass_kg DOUBLE PRECISION NOT NULL,
  engine_isp_sec DOUBLE PRECISION NOT NULL,

  total_fuel_mass_kg DOUBLE PRECISION NULL
);

-- m-m between request and route
CREATE TABLE IF NOT EXISTS flight_request_routes (
  flight_request_id BIGINT NOT NULL REFERENCES flight_requests(id) ON DELETE RESTRICT,
  route_id BIGINT NOT NULL REFERENCES transfer_routes(id) ON DELETE RESTRICT,

  quantity INTEGER NOT NULL DEFAULT 1,
  segment_order INTEGER NOT NULL DEFAULT 1,
  is_primary BOOLEAN NOT NULL DEFAULT FALSE,

  payload_mass_kg DOUBLE PRECISION NULL,

  -- computed fields (can be recalculated)
  delta_v_kms DOUBLE PRECISION NULL,
  fuel_mass_kg DOUBLE PRECISION NULL,

  PRIMARY KEY (flight_request_id, route_id)
);

-- statuses for flight_requests:
-- draft, deleted, formed, completed, rejected

-- one draft per creator
CREATE UNIQUE INDEX IF NOT EXISTS ux_flight_requests_one_draft_per_user
  ON flight_requests(created_by)
  WHERE status = 'draft';

-- helpful indexes
CREATE INDEX IF NOT EXISTS ix_transfer_routes_active
  ON transfer_routes(is_deleted);

CREATE INDEX IF NOT EXISTS ix_flight_requests_status
  ON flight_requests(status);

