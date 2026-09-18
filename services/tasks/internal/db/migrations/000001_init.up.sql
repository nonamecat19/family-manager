CREATE TABLE IF NOT EXISTS family_settings (
    family_id   UUID PRIMARY KEY,
    timezone    TEXT NOT NULL CHECK (length(btrim(timezone)) > 0),
    digest_time TEXT NOT NULL DEFAULT '08:00' CHECK (digest_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tasks (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id            UUID NOT NULL,
    title                TEXT NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 200),
    notes                TEXT NOT NULL DEFAULT '' CHECK (length(notes) <= 4000),
    priority             TEXT NOT NULL DEFAULT 'medium' CHECK (priority IN ('low','medium','high','urgent')),
    status               TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','done')),
    due_on               DATE,
    due_time             TIME,
    due_at               TIMESTAMPTZ,
    created_by_user_id   UUID NOT NULL,
    completed_by_user_id UUID,
    completed_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (due_time IS NULL OR due_on IS NOT NULL),
    CHECK ((due_on IS NULL) = (due_at IS NULL)),
    CHECK ((status = 'done') = (completed_at IS NOT NULL)),
    CHECK ((status = 'done') = (completed_by_user_id IS NOT NULL)),
    UNIQUE (id, family_id)
);

CREATE INDEX IF NOT EXISTS tasks_family_status_due ON tasks (family_id, status, due_at);
CREATE INDEX IF NOT EXISTS tasks_open_due ON tasks (due_at) WHERE status = 'open' AND due_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS task_assignees (
    task_id   UUID NOT NULL,
    family_id UUID NOT NULL,
    user_id   UUID NOT NULL,
    PRIMARY KEY (task_id, user_id),
    FOREIGN KEY (task_id, family_id) REFERENCES tasks (id, family_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS task_assignees_family_user ON task_assignees (family_id, user_id);

CREATE TABLE IF NOT EXISTS birthdays (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id          UUID NOT NULL,
    name               TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    day                INT  NOT NULL CHECK (day BETWEEN 1 AND 31),
    month              INT  NOT NULL CHECK (month BETWEEN 1 AND 12),
    year               INT  CHECK (year IS NULL OR year BETWEEN 1900 AND 2200),
    remind_days_before INT  NOT NULL DEFAULT 0 CHECK (remind_days_before BETWEEN 0 AND 60),
    created_by_user_id UUID NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS birthdays_family_month_day ON birthdays (family_id, month, day);

CREATE TABLE IF NOT EXISTS google_connections (
    family_id      UUID NOT NULL,
    user_id        UUID NOT NULL,
    email          TEXT NOT NULL DEFAULT '',
    refresh_token  BYTEA NOT NULL,
    calendar_id    TEXT NOT NULL DEFAULT '',
    calendar_name  TEXT NOT NULL DEFAULT '',
    sync_token     TEXT NOT NULL DEFAULT '',
    last_synced_at TIMESTAMPTZ,
    last_error     TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id)
);

CREATE TABLE IF NOT EXISTS calendar_links (
    family_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('task','birthday')),
    item_id     UUID NOT NULL,
    calendar_id TEXT NOT NULL,
    event_id    TEXT NOT NULL,
    etag        TEXT NOT NULL DEFAULT '',
    synced_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id, kind, item_id),
    UNIQUE (user_id, calendar_id, event_id)
);

CREATE INDEX IF NOT EXISTS calendar_links_item ON calendar_links (family_id, kind, item_id);

CREATE TABLE IF NOT EXISTS task_reminders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id     UUID NOT NULL,
    user_id       UUID NOT NULL,
    kind          TEXT NOT NULL CHECK (kind IN ('task_due','birthday')),
    item_id       UUID NOT NULL,
    occurrence    TEXT NOT NULL,
    remind_at     TIMESTAMPTZ NOT NULL,
    snoozed_until TIMESTAMPTZ,
    acked_at      TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (family_id, user_id, kind, item_id, occurrence)
);

CREATE INDEX IF NOT EXISTS task_reminders_user_pending ON task_reminders (family_id, user_id, remind_at) WHERE acked_at IS NULL;
CREATE INDEX IF NOT EXISTS task_reminders_due ON task_reminders (remind_at) WHERE acked_at IS NULL;

CREATE TABLE IF NOT EXISTS reminder_deliveries (
    reminder_id  UUID NOT NULL REFERENCES task_reminders (id) ON DELETE CASCADE,
    family_id    UUID NOT NULL,
    channel      TEXT NOT NULL CHECK (channel IN ('push','telegram')),
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (reminder_id, channel)
);

CREATE TABLE IF NOT EXISTS digest_sends (
    family_id UUID NOT NULL,
    user_id   UUID NOT NULL,
    sent_on   DATE NOT NULL,
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (family_id, user_id, sent_on)
);
