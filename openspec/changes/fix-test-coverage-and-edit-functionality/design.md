## Context
The MMemory project has reached a critical point where core functionality needs completion and testing infrastructure requires significant improvement. The edit functionality is incomplete, test coverage is substantially below target (30.9% vs 80%), and conversation history integration is not fully implemented despite infrastructure being in place.

Key stakeholders:
- End users: Need reliable reminder editing capabilities
- Development team: Requires stable test infrastructure for confident development
- Operations: Needs comprehensive monitoring and error handling

Technical constraints:
- Must maintain existing API compatibility
- Testing must use proper mocking to avoid external service dependencies
- Performance requirements: <3s response time maintained
- Chinese language support must be preserved

## Goals / Non-Goals

Goals:
- Fix all test compilation errors and achieve 80%+ test coverage
- Implement complete edit functionality with intuitive UI
- Integrate conversation history into AI parsing
- Maintain system stability and performance
- Provide comprehensive error handling and user feedback

Non-Goals:
- Adding new AI providers beyond OpenAI
- Major architectural changes to existing systems
- Performance optimization beyond current requirements
- New reminder types or scheduling patterns
- UI/UX redesign beyond edit functionality

## Decisions

### Decision 1: Incremental Test Coverage Improvement
**What**: Improve test coverage incrementally starting with critical paths
**Why**: Reduces risk of introducing new bugs while fixing existing issues
**Implementation**:
- Start with service layer test fixes
- Add unit tests for new edit functionality
- Gradually improve coverage across all modules

### Decision 2: Service-Layer Edit Implementation
**What**: Implement edit functionality at service layer first, then expose through handlers
**Why**: Provides clean separation of concerns and enables comprehensive testing
**Implementation**:
- Add EditReminder method to ReminderService
- Implement validation and authorization checks
- Build handler layer on top of service layer

### Decision 3: Context-Aware AI Parsing
**What**: Integrate conversation history as optional context parameter to AI parsing
**Why**: Maintains backward compatibility while enabling enhanced functionality
**Implementation**:
- Modify AIParserService to accept optional context
- Build context from 30-day conversation history
- Use context only when beneficial for parsing accuracy

### Decision 4: Mock-Based Testing Strategy
**What**: Use comprehensive mocking for external dependencies (OpenAI API, Telegram API)
**Why**: Enables reliable, fast unit testing without external service dependencies
**Implementation**:
- Mock OpenAI API responses for different scenarios
- Mock Telegram API for bot handler testing
- Use interface-based design for easy mocking

Alternatives considered:
- Integration testing with real APIs: Rejected due to reliability and speed concerns
- Full rewrite of test suite: Rejected as too risky and time-consuming
- Minimal test coverage improvement: Rejected as doesn't meet project standards

## Risks / Trade-offs

**Risk 1**: Test fixes may reveal hidden bugs in existing code
- Mitigation: Comprehensive testing of fixes, staged rollout

**Risk 2**: Edit functionality may introduce data consistency issues
- Mitigation: Transaction-based updates, comprehensive validation, audit logging

**Risk 3**: Conversation context may degrade AI parsing performance
- Mitigation: Efficient context retrieval, relevance filtering, performance monitoring

**Risk 4**: High test coverage requirement may slow development
- Mitigation: Incremental approach, focusing on critical paths first

## Migration Plan

### Phase 1: Test Infrastructure (Week 1)
1. Fix existing test compilation errors
2. Add missing mock methods
3. Ensure all existing tests pass

### Phase 2: Edit Functionality (Week 2-3)
1. Implement service layer edit methods
2. Add comprehensive edit tests
3. Build handler layer integration
4. Add UI components (inline keyboards)

### Phase 3: Context Integration (Week 3-4)
1. Modify AI parser for context support
2. Implement conversation history retrieval
3. Add context-aware prompt building
4. Test multi-turn conversations

### Phase 4: Coverage Improvement (Week 4-6)
1. Achieve 80%+ coverage on pkg/ai
2. Achieve 80%+ coverage on internal/service
3. Achieve 80%+ coverage on internal/ai
4. Add edge case and integration tests

Rollback plan:
- Each phase can be rolled back independently
- Git tags created at phase boundaries
- Feature flags for new functionality
- Database migrations are backward compatible

## Open Questions

1. Should we implement edit history/undo functionality for reminders?
2. How should we handle concurrent edit requests to the same reminder?
3. What's the optimal context window size for conversation history?
4. Should we implement A/B testing for AI parsing improvements?
5. How do we measure the impact of context integration on parsing accuracy?