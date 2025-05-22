# Changelog

## [0.1.0] - 2025-05-23
### Added
- Initial release of **ExecuteTurn1Combined**
- Implements S3 state management for system prompts and reference images
- Generates turn‑1 prompt using templated prompts
- Invokes Amazon Bedrock (Claude Sonnet) via the Converse API
- Stores raw and processed responses back to S3
- Updates DynamoDB with status and conversation history
- Exposes `HandleTurn1Combined` entrypoint for Step Functions
