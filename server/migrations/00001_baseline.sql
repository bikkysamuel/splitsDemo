-- Baseline. Case-insensitive email columns use citext (doc 06). The extension
-- lives in public so that test schemas can share it.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;
