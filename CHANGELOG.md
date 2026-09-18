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
- The home screen lets you choose the library folder and import the catalog from an `.inpx` file.
- While an import is running the window is blocked and shows the current stage, the number of records processed and a progress bar based on the size of the dump; the import can be cancelled until the data is written to the database.
- An import report is shown afterwards: how many works and editions were added or changed, which archives are missing from disk, which genres have no name, the file encodings that were detected and how long each stage took.

### Changed

### Fixed

- If the data folder already contains a catalog created by another program, this is stated plainly: the file is neither opened nor converted.

### Removed

### Performance

### Security
