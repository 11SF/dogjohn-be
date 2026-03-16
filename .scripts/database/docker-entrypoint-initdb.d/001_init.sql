-- price_options
CREATE TABLE IF NOT EXISTS price_options (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    price         NUMERIC(10, 2)   NOT NULL,
    description   TEXT             NOT NULL,
    display_order INT              NOT NULL DEFAULT 0
);

-- orders
CREATE TABLE IF NOT EXISTS orders (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    customer_name TEXT             NOT NULL,
    price_id      TEXT             NOT NULL REFERENCES price_options (id),
    slip_image        BYTEA            NOT NULL,
    payment_txn_ref   TEXT,
    order_status  TEXT             NOT NULL DEFAULT 'PENDING'
                      CHECK (order_status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    failure_reason TEXT,
    ordered_at    BIGINT           NOT NULL
);

-- payment_txn_logs
CREATE TABLE IF NOT EXISTS payment_txn_logs (
    id         BIGSERIAL    PRIMARY KEY,
    order_id   TEXT         NOT NULL REFERENCES orders (id),
    response   JSONB        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- payment_config
CREATE TABLE IF NOT EXISTS payment_config (
    id             SERIAL PRIMARY KEY,
    prompt_pay_id  TEXT NOT NULL
);

-- feeder_config
CREATE TABLE IF NOT EXISTS feeder_config (
    id         SERIAL PRIMARY KEY,
    dog_count  INT NOT NULL DEFAULT 0
);

-- feeder_status
CREATE TABLE IF NOT EXISTS feeder_status (
    id         SERIAL PRIMARY KEY,
    available  BOOLEAN NOT NULL DEFAULT TRUE,
    reason     TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
