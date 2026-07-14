-- version the cached media row so a binary/extractor change can
-- invalidate previously cached (and possibly buggy) results.
ALTER TABLE media ADD COLUMN cache_version TEXT NOT NULL DEFAULT '1';
