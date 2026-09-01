-- Drop in reverse dependency order: the children of notes first, then the share tables, then
-- notes (which references notebooks), then notebooks last. Every index above belongs to one
-- of these tables and goes with it, so none is named here.

DROP TABLE IF EXISTS note_activity;
DROP TABLE IF EXISTS note_comments;
DROP TABLE IF EXISTS notebook_shares;
DROP TABLE IF EXISTS note_shares;
DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS notebooks;
