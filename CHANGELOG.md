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
- A backup is created before the schema of an existing database is upgraded; the three most recent backups are kept.
- The home screen shows how many books are in the catalog, when the last import ran, and a grid of recent arrivals.
- The catalog can be searched and browsed in virtualized lists, with tile or table layout.
- Authors, series and genres have their own lists with an alphabet index.
- Opening a book from the catalog, the command palette or a random pick shows a side panel on the current page; Back closes it.
- Book covers load for cards that are on screen and fall back to the coloured title plate when a cover is missing.
- The book page and side panel show the annotation and previous/next books in a series.
- Settings can clear the cover cache and preload covers for rated and recently added books.
- First-run setup walks through choosing the library folder, the `.inpx` dump and starting the import.
- A command palette (Ctrl+K) finds books, authors, series and recent searches as you type.
- While a database schema update is applied at startup, a dedicated screen explains that a backup was made and asks you to wait.
- While an import is running the window is blocked and shows the current stage, the number of records processed and a progress bar based on the size of the dump; the import can be cancelled until the data is written to the database.
- An import report is shown afterwards: how many works and editions were added or changed, which archives are missing from disk, which genres have no name, the file encodings that were detected and how long each stage took.

### Changed

- Catalog filters open as a sliding panel and can be set by language, genre, author and series; the selection stays in the address bar.
- The command palette, sidebar and drop-down lists use the same themed controls as the rest of the window, including keyboard movement in the palette and on the book grid.
- Sidebar, filter and list pictograms use a single icon set instead of mixed letter marks and ad-hoc drawings.
- Book tiles always show the title and authors under the cover; the coloured plate replaces a missing image, not the captions.
- The book panel and book page show the cover in full on a blurred, darkened copy of the same picture; the page uses a narrow cover column beside the text.
- Language is shown as a name, not a dump code; a missing author is labelled in plain language and is not a link.
- If a cover or annotation takes about a second to read from disk, the panel says so instead of waiting on a skeleton.
- A tile that has several files only shows the labelled count, not a bare number next to it.
- Untitled books appear at the end of the catalog, not at the start.

### Fixed

- If the data folder already contains a catalog created by another program, this is stated plainly: the file is neither opened nor converted.
- The collapsed sidebar no longer leaves an empty strip beside the icons.
- Catalog lists no longer come up blank while a result count is shown.
- Filter chips show the selected name, not the filter kind.
- Highlighting a row in a searchable list no longer draws a second, clipped focus ring.
- Closing the window during an import or cover preload now cancels that work, so a library disk can be ejected afterwards.
- Closing the window no longer waits tens of seconds for background work.
- After a successful import, books that previously had no cover are checked again instead of staying blank.
- Clearing the cover cache is visible without reloading the window.
- If a cover cannot be loaded right now, the title plate is shown; opening the book again or returning to the list tries once more.
- Annotations are no longer stored as mojibake; already stored garbage is cleared so the book is read again (database migration).
- Garbled annotations saved after that first cleanup are cleared again (database migration).
- Annotation paragraphs stay as separate lines.
- The tile/table switch no longer uses the accent colour that marks a primary action.

### Removed

- The home screen no longer ends with a button that only opened the library folder settings.

### Performance

### Security
