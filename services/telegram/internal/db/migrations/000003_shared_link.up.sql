TRUNCATE telegram_links;

ALTER TABLE telegram_links DROP CONSTRAINT telegram_links_pkey;
ALTER TABLE telegram_links DROP COLUMN bot;
ALTER TABLE telegram_links ADD PRIMARY KEY (telegram_user_id);
