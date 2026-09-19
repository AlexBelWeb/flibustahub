-- Empty sort keys collate before every real title or name. Store U+FFFF
-- so they sort last while (sort_title, id) remains a range on the index.
UPDATE works SET sort_title = char(65535) WHERE sort_title = '';
UPDATE authors SET sort_name = char(65535) WHERE sort_name = '';
UPDATE series SET sort_name = char(65535) WHERE sort_name = '';
