ALTER TABLE flight_request_routes
ADD COLUMN IF NOT EXISTS segment_dry_mass_kg DOUBLE PRECISION NULL;

ALTER TABLE flight_request_routes
ADD COLUMN IF NOT EXISTS segment_isp_sec DOUBLE PRECISION NULL;
