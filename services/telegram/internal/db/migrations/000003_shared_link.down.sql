TRUNCATE telegram_links;

ALTER TABLE telegram_links DROP CONSTRAINT telegram_links_pkey;
ALTER TABLE telegram_links ADD COLUMN bot TEXT NOT NULL;
ALTER TABLE telegram_links ADD PRIMARY KEY (bot, telegram_user_id);
