## MODIFIED Requirements
### Requirement: Conversation Context Management
The system SHALL maintain 30-day conversation history and integrate it into AI parsing for context-aware natural language understanding.

#### Scenario: Multi-turn conversation for reminder editing
- **WHEN** user says "取消我昨天的提醒" after creating multiple reminders
- **THEN** system analyzes conversation history to identify the specific reminder
- **AND** asks for confirmation with reminder details
- **AND** cancels the correct reminder based on context

#### Scenario: Context-aware reminder creation
- **WHEN** user says "和昨天一样的时间提醒我吃药"
- **THEN** system retrieves yesterday's reminder time from conversation history
- **AND** creates new reminder with the same time pattern
- **AND** confirms the inferred time with user

#### Scenario: Conversation context for ambiguous requests
- **WHEN** user makes ambiguous request like "改一下那个提醒"
- **THEN** system uses conversation history to disambiguate
- **AND** asks clarifying questions based on recent interactions
- **AND** provides relevant suggestions

## ADDED Requirements
### Requirement: Context-Aware AI Parsing
The system SHALL build context-aware prompts that include conversation history to improve natural language understanding accuracy.

#### Scenario: AI parsing with conversation context
- **WHEN** AI parser service receives user message
- **THEN** system retrieves 30-day conversation history for the user
- **AND** builds context-rich prompt including previous reminders and interactions
- **AND** returns more accurate parsing results

#### Scenario: Context building for complex requests
- **WHEN** user refers to previous interactions ("上次的", "昨天的", "之前那个")
- **THEN** system identifies referential expressions in conversation history
- **AND** resolves them to specific reminders or time patterns
- **AND** provides appropriate responses

### Requirement: Conversation History Integration
The system SHALL seamlessly integrate conversation history into the AI parsing pipeline without degrading performance or accuracy.

#### Scenario: Efficient context retrieval
- **WHEN** AI parsing request includes conversation context flag
- **THEN** system efficiently retrieves relevant conversation history
- **AND** formats it appropriately for AI model consumption
- **AND** maintains parsing response time under 3 seconds

#### Scenario: Context relevance filtering
- **WHEN** building context from 30-day history
- **THEN** system filters for relevant conversations based on recency and similarity
- **AND** limits context size to prevent token overflow
- **AND** prioritizes most relevant interactions