-- Clear stored annotations that fail the text plausibility gate so they are
-- extracted again with the corrected decoder. A backup is taken before this
-- file runs when the catalog already exists.

UPDATE works
   SET annotation = NULL,
       annotation_checked_at = NULL
 WHERE annotation IS NOT NULL
   AND trim(annotation) != ''
   AND text_plausible(annotation) = 0;
