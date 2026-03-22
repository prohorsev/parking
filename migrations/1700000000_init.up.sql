CREATE TABLE IF NOT EXISTS parking_lots (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id    VARCHAR(255)  NOT NULL,
    provider       VARCHAR(100)  NOT NULL,
    name           VARCHAR(255)  NOT NULL,
    address        VARCHAR(500)  NOT NULL DEFAULT '',
    lat            DOUBLE PRECISION NOT NULL,
    lng            DOUBLE PRECISION NOT NULL,
    total_spots    INT           NOT NULL DEFAULT 0,
    free_spots     INT           NOT NULL DEFAULT 0,
    price_per_hour DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency       VARCHAR(10)   NOT NULL DEFAULT 'USD',
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (external_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_parking_lots_lat_lng ON parking_lots (lat, lng);
