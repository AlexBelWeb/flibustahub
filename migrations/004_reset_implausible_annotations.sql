-- Repeat the plausibility reset. 003 already ran on existing catalogs; an
-- older extractor could still write mojibake afterwards, and
-- annotation_checked_at then blocked another pass.

UPDATE works
   SET annotation = NULL,
       annotation_checked_at = NULL
 WHERE annotation IS NOT NULL
   AND trim(annotation) != ''
   AND text_plausible(annotation) = 0;
