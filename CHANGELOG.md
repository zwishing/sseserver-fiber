# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [0.0.3] - 2026-04-01

### Added
- Added topic-aware APIs: `HandlerWithTopic`, `SubscribeWithTopic`, `PublishEventWithTopic`, and `PublishJSONWithTopic`
- Added topic coverage tests for exact namespace/topic matching and namespace-level broadcast fallback

### Changed
- Extended message routing from namespace-only to namespace + topic
- Kept existing namespace-only APIs as backward-compatible shortcuts using empty-topic routing
- Updated README and example app to demonstrate topic-based subscription and publishing

## [0.0.2] - 2026-03-31

### Added
- Added project-level changelog tracking
- Added `.gitignore` for macOS metadata files

### Changed
- Simplified internal code comments to concise English documentation

## [0.0.1] - 2026-03-31

### Added
- Instance-based API with `New`, `Handler`, `Publish`, `PublishEvent`, and `PublishJSON`
- Configurable connection and publish buffers
- Configurable keepalive interval
- Basic tests covering options, shutdown behavior, message formatting, and namespace routing

### Changed
- Migrated Fiber integration to v3
- Simplified the public API around explicit `Server` lifecycle management
- Improved SSE framing for multiline payloads
- Switched namespace routing to exact-match semantics
- Added buffered publish queue to reduce publisher blocking under load
- Avoided an extra payload copy in `PublishJSON`

### Fixed
- Removed panic paths around uninitialized or closed server state
- Prevented unregister goroutines from blocking during shutdown races
- Made hub shutdown idempotent
