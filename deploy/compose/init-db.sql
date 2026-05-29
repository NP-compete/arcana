CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE DATABASE arcana_skills;
CREATE DATABASE arcana_temporal;
CREATE DATABASE arcana_temporal_visibility;

GRANT ALL PRIVILEGES ON DATABASE arcana_skills TO arcana;
GRANT ALL PRIVILEGES ON DATABASE arcana_temporal TO arcana;
GRANT ALL PRIVILEGES ON DATABASE arcana_temporal_visibility TO arcana;
