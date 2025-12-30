# Change: Fix Test Coverage Issues and Implement Edit Functionality

## Why
The project currently has critical test failures preventing reliable development and deployment. The edit functionality is incomplete, only returning "功能建设中" (feature under construction) messages. Additionally, test coverage is at 30.9% vs the target 80%, and conversation history context is not integrated into AI parsing despite infrastructure being in place.

## What Changes
- Fix all test compilation errors and failures in service layer
- Implement complete edit functionality with service methods, handlers, and UI interactions
- Integrate 30-day conversation history context into AI parsing for better natural language understanding
- Improve test coverage from 30.9% to 80%+ across pkg/ai, internal/service, and internal/ai modules
- Add missing mock methods for proper unit testing
- Enhance AI parsing with multi-turn conversation support

## Impact
- Affected specs: reminder-management, conversation-context, ai-parsing
- Affected code:
  - `internal/service/` - Edit functionality and test fixes
  - `internal/bot/handlers/` - Edit handlers and conversation integration
  - `pkg/ai/` - Test coverage improvements
  - `internal/ai/` - Conversation context integration
- **BREAKING**: Some test interfaces may change due to mock method additions