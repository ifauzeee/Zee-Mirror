-- Migration to add quality column to tasks table
ALTER TABLE tasks ADD COLUMN quality TEXT;
