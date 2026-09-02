# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.7] - 2026-09-02

### Changed

- Expand fake data generation with additional names, cities, countries, and job titles
- Update .gitignore to exclude all database files

## [0.0.6] - 2026-08-30

### Added

- Add govulncheck target, enhance error handling, and improve schema validation
- Implement persistent db
- Add benchmark target to Makefile for performance testing

### Changed

- Update readme
- Enhance database persistence and performance with new SQLite options, add InsertReturning and UpdateReturning functions, and improve seeding efficiency
- Merge pull request #4 from arias9306/persist-db
- Update changelog for v0.0.6

## [0.0.5] - 2026-08-29

### Added

- Add TableEndpoint and Join types for enhanced schema definition
- Add table endpoint functionality with request handling and response generation
- Initialize column map with 'id' for table validation and add test for endpoint joins
- Implement table endpoint registration and handler logic in API
- Add 'format' to 'title' and 'created_at' column for posts schema
- Add Modern Go Guidelines skill with installation scripts and versioning

### Changed

- Improve formatting and punctuation in README.md
- Simplify error handling and status assignment using cmp package
- Replace custom error and JSON response handling with httputil package
- Streamline endpoint registration and context handling using httputil package
- Enhance schema definitions and examples
- Merge pull request #3 from arias9306/support-query-tables
- Update changelog for v0.0.5

## [0.0.4] - 2026-08-22

### Added

- Add example JSON schemas for blog, ecommerce, mock, and validation

### Changed

- Enhance data generation with additional formats and types in fakegen
- Update changelog for v0.0.4

## [0.0.3] - 2026-08-19

### Added

- Add MIT License to the project
- Add fake data generation and mock endpoint handling
- Implement endpoint collection and printing functionality
- Add linting target to Makefile
- Add error handling for JSON encoding in writeJSON function
- Add CORS support and improve server shutdown handling with request logging
- Add CORS origin flag to server configuration in README
- Add race testing and coverage targets to Makefile; enhance endpoint printing with formatted output
- Add tests for schema validation, rendering, and seeding functionality

### Changed

- Include documentation files in build artifacts
- Enhance schema validation and add endpoint support
- Update README
- Improve output formatting for endpoints and server messages
- Handle edge cases in random value generation for integers and floats
- Update Seed function to use database interface and remove insertRow helper
- Enhance schema validation with comprehensive checks for tables and columns
- Enhance CreateTable and query functions with proper identifier quoting and foreign key support
- Improve handler structure and error handling in API methods
- Update Makefile to generate coverage reports in a dedicated directory
- Merge pull request #1 from arias9306/endpoints-support
- Merge pull request #2 from arias9306/testing
- Update CHANGELOG for version 0.0.3 with added features, changes, and fixes

### Fixed

- Correct validation logic for string length and regex matching in validateColumn function

## [0.0.2] - 2026-08-15

### Added

- Add CORS support to the HTTP server

### Changed

- Update changelog for v0.0.2

## [0.0.1] - 2026-08-15

### Added

- Initialize project with go.mod and main.go; add .gitignore
- Add schema parsing functionality and initial schema definition
- Implement database initialization and table creation functionality
- Add seed func
- Implement API handler and seed functionality with schema support
- Add value generation functions for seeding
- Implement pagination and sorting in SelectAll function; enhance List handler for table data retrieval
- Implement Get handler for retrieving a row by ID; add SelectByID function in db package
- Implement Create handler for inserting a new row; add validation for input data
- Implement Update handler for modifying existing rows; add validation for update data
- Implement Delete handler for removing a row by ID; add error handling for deletion scenarios
- Add package comments for api, db, schema, seed, and validation packages
- Add length validation for generated strings to ensure compliance with column constraints
- Add versioning support with build-time information and version flag
- Add changelog generation support with git-cliff configuration
- Add README.md file

### Changed

- Refactor API handler and enhance seed functionality with row insertion
- Refactor main function to improve server setup and signal handling
- Update default row count and enhance data generation with address, city, and country fields
- Update .gitignore to exclude dist directory and migrate from mattn/go-sqlite3 to modernc.org/sqlite
- Update changelog for v0.0.1

### Fixed

- Correct receiver name in HasErrors method for consistency
- Correct property names for min_length and max_length in users table

[0.0.7]: https://github.com/arias9306/schema-api/compare/v0.0.6..v0.0.7
[0.0.6]: https://github.com/arias9306/schema-api/compare/v0.0.5..v0.0.6
[0.0.5]: https://github.com/arias9306/schema-api/compare/v0.0.4..v0.0.5
[0.0.4]: https://github.com/arias9306/schema-api/compare/v0.0.3..v0.0.4
[0.0.3]: https://github.com/arias9306/schema-api/compare/v0.0.2..v0.0.3
[0.0.2]: https://github.com/arias9306/schema-api/compare/v0.0.1..v0.0.2
[0.0.1]: https://github.com/arias9306/schema-api/releases/tag/v0.0.1

<!-- generated by git-cliff -->
