# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/).

Conventions:

- an entry is added **in the same commit** that changes behaviour;
- entries are written for users, not for commits: "Search no longer drops results while you type quickly", not "refactor debounce";
- internal refactorings with no visible effect are not listed;
- one entry is one line;
- entries that require a database migration are marked explicitly;
- **before the `1.0.0` release only the `Unreleased` section exists** — no versions are cut, no dates are set and no interim releases are published.

## [Unreleased]

### Added

- Desktop application shell: window, interface language switch (Russian and English), dark and light themes.
- The application log is written to the user data folder and rotates instead of growing without bound.
- Starting the application a second time focuses the existing window instead of launching another instance.
- If the settings cannot be loaded at startup, the window still opens, explains what happened and offers to retry.
- The book catalog is stored in a local database, and the schema is upgraded automatically when the application is updated.
- A backup is created before the schema of an existing database is upgraded, before a catalog import, and before personal marks are imported; the three most recent backups of each kind are kept.
- The home screen shows a return-to-book block, how many books, authors and series are in the catalog, when the last import ran, shortcuts to popular genres and series, and carousels of new arrivals and recent ratings.
- The catalog can be searched and browsed in virtualized lists, with tile or table layout.
- Authors, series and genres have their own lists with an alphabet index.
- Opening a book from the catalog, the command palette or a random pick shows a side panel on the current page; Back closes it.
- Book covers load for cards that are on screen and fall back to a coloured monogram when a cover is missing.
- The book page and side panel show the annotation and previous/next books in a series.
- Settings can clear the cover cache and preload covers for rated and recently added books.
- First-run setup walks through choosing the library folder, the `.inpx` dump and starting the import.
- Books can be rated with half-stars, given a note, and marked want-to-read from the book panel and page.
- Catalog filters include books rated by you and books marked want-to-read; sorting by personal rating is offered when the rated filter is on.
- Settings has a Personal data section for exporting and importing ratings, notes and want-to-read marks, with a preview before applying.
- Books can be opened in an external reader or saved to the system downloads folder; a missing archive or disconnected library is explained instead of failing silently.
- Unavailable Read and Download controls show why they are disabled.
- A layout banner reports when the library disk is offline or unreadable, with Check again and Choose folder again; a compact reminder stays after the banner is dismissed.
- Covers of visible cards reload when the library disk comes back, without restarting the app.
- A quieter notice offers to update the catalog when a newer dump is in the library folder; import never starts by itself.
- Settings for the downloads folder and an optional reader application have their own section.
- Settings lists sections down the left side, or in a single control on a narrow window: Interface, Library and database, Downloads folder and reading, Personal data, AI assistant, and Diagnostics.
- The AI assistant section lets you choose Gemini, OpenAI or Ollama, save or delete an API key without showing it, and says whether the key is in the OS store or an encrypted file.
- While a database schema update is applied at startup, a dedicated screen explains that a backup was made and asks you to wait.
- While an import is running the window is blocked and shows the current stage, the number of records processed and a progress bar based on the size of the dump; the import can be cancelled until the data is written to the database.
- An import report is shown afterwards: how many works and editions were added or changed, which archives are missing from disk, which genres have no name, the file encodings that were detected and how long each stage took.
- The catalog can be optimized from Library and database settings after several incremental imports; this refreshes planner statistics and does not always free disk space.
- A manual copy of the catalog can be saved from the same settings into the backup folder.
- Diagnostics settings list paths, versions and catalog stats, save a support archive without ratings, notes or books, and prepare a GitHub issue or a markdown report you can edit first.

### Changed

- Panels, dialogs, menus and toasts ease in and out instead of appearing all at once.
- Personal marks, library facts, warnings, failures and completed actions use separate colors, and a destructive button is no longer the same color as the focus ring.
- Secondary text on a muted surface is darker, so it stays readable.
- Catalog tiles share one row height, and the title on a tile is a step smaller. Covers stay put while you scroll.
- Settings navigation is a vertical list of sections instead of three tabs.
- Catalog filters open as a sliding panel and can be set by language, genre, author and series; the selection stays in the address bar.
- The command palette, sidebar and drop-down lists use the same themed controls as the rest of the window, including keyboard movement in the palette and on the book grid.
- Sidebar, filter and list pictograms use a single icon set instead of mixed letter marks and ad-hoc drawings.
- The book panel uses a tall 2:3 cover, authors as text links, and language, size and format on one line.
- The book panel keeps the title and authors in a header with the close button, and opens at the start of the book instead of remembering the previous scroll position.
- The book page lists every edition with its archive and size.
- A missing cover shows a coloured monogram from the title, not the same title and author again on the plate.
- The home screen opens on counters, shortcuts and carousels instead of a static arrivals grid.
- Untitled books appear at the end of the catalog, not at the start.
- Lists, the home grid and the book panel leave empty space at the bottom of the scroll area.
- The book panel and book page show the cover in full on a blurred, darkened copy of the same picture; the page uses a narrow cover column beside the text.
- Language is shown as a name, not a dump code; a missing author is labelled in plain language and is not a link.
- If a cover or annotation takes about a second to read from disk, the panel says so instead of waiting on a skeleton.
- A tile that has several files only shows the labelled count, not a bare number next to it.
- Search looks up books by title, author name and series together, so typing an author opens their books instead of an empty list.
- Search results show matching authors and series above the book list; the command palette highlights the first match and offers “all authors” / “all series” when there are more.
- The book page has one primary Read button; edition rows use quiet Read/Download controls, show the file name inside the archive and the added date, and mark the header’s edition as the default. The summary line uses that edition’s size. The files count stays on the card and panel, not next to the edition list.
- Older catalog backups that shared one filename are removed the next time a backup is made; new backups are named by why they were taken.
- Home shortcuts to genres and series scroll with the same chevrons as the book carousels and no longer show a scrollbar strip.
- Want-to-read is a two-state button instead of a switch.
- The note editor says that Markdown is supported.
- The home hero shows why the book is there, its title and authors, a short metadata line, Open as the main action and a quiet Read; a random pick can be swapped for another book. The cover sits in a 2:3 frame instead of a cropped strip.
- The last-import tile shows the date and dump version in the same type as the other counters.
- The book panel and page put the annotation above editions and the note; rating and want-to-read sit on one row under the actions, without a Rating heading. An empty note collapses to “Add a note”.
- The sidebar eases between the full column and the icon column.
- A book’s series is its own line under the cover, with the number from the catalog. Genres stay as separate tags. A series number of 0 is left off: in the catalog that value means the number is missing.
- Cards, menus and dialogs are set off by a shadow. With visual effects off, a fine edge takes the shadow’s place so those surfaces do not blend together.
- A book’s own color tints the title in the panel and on the book page. Pointing at a card lifts it.
- If another copy is running but does not answer and does not release the catalog, startup shows a screen with Retry instead of an empty window.
- Author, genre and series filters can be cleared. An empty filter reads as “any”, and a value that cannot be read stays visible so the list does not look unfiltered.

### Fixed

- Right after the application is forced to quit, a brief database lock on the next start is retried before a disk error is shown.
- Closing the application and opening it again straight away no longer leaves nothing running. The new launch waits until the catalog is free, or offers Retry if shutdown is still in progress.
- If the catalog is still closing when shutdown runs out of time, a new launch does not open the database until that process has exited.
- The dimmed backdrop fades out with the panel. It no longer disappears in one step after the panel has already left.
- The home hero no longer shows an empty strip with a cropped cover.
- If the data folder already contains a catalog created by another program, this is stated plainly: the file is neither opened nor converted.
- The collapsed sidebar no longer leaves an empty strip beside the icons.
- Catalog lists no longer come up blank while a result count is shown.
- Filter chips show the selected name, not the filter kind.
- Highlighting a row in a searchable list no longer draws a second, clipped focus ring.
- Closing the window during an import or cover preload now cancels that work, so a library disk can be ejected afterwards.
- Closing the window no longer waits tens of seconds for background work.
- After a successful import, books that previously had no cover are checked again instead of staying blank.
- Clearing the cover cache is visible without reloading the window.
- If a cover cannot be loaded right now, the monogram plate is shown; opening the book again or returning to the list tries once more.
- Annotations are no longer stored as mojibake; already stored garbage is cleared so the book is read again (database migration).
- Garbled annotations saved after that first cleanup are cleared again (database migration).
- Annotation paragraphs stay as separate lines.
- The tile/table switch no longer uses the accent colour that marks a primary action.
- Empty space around a cover in the panel no longer shows the monogram through the blur.
- The files badge uses the correct plural (1 file, 2 files / 1 файл, 2 файла, 5 файлов).
- A cover in the book panel no longer paints over the text below it.
- Show in folder and Open downloads folder no longer open Documents when the path has spaces or the file is gone; the app reports the error instead.
- The home screen no longer grows a sideways page scrollbar when carousels and shortcuts are wider than the window.
- Home genre shortcuts and book carousels keep their arrows beside the content instead of on top of it.

### Removed

- The home screen no longer ends with a button that only opened the library folder settings.
- The home screen no longer starts with the product name, the word “Home”, or two lines explaining what the catalog is.

### Performance

### Security

- API keys are stored in the operating system credential store, or in an encrypted file when that store is unavailable.
