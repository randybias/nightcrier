## 1. Create Incident Package

- [ ] 1.1 Create `internal/incident/incident.go` with Incident struct
- [ ] 1.2 Add status constants (pending, investigating, resolved, failed, agent_failed)
- [ ] 1.3 Add `NewFromEvent()` constructor to create Incident from FaultEvent
- [ ] 1.4 Add `WriteToFile()` method to write/update incident.json
- [ ] 1.5 Add `MarkCompleted()` method to update status and result fields

## 2. Clean Up Event Struct

- [ ] 2.1 Remove `IncidentID` field from FaultEvent (revert recent addition)
- [ ] 2.2 Add `EventID` field to FaultEvent (generated on receipt)
- [ ] 2.3 Add `ReceivedAt` field to FaultEvent
- [ ] 2.4 Add `DeduplicationKey()` method to FaultEvent
- [ ] 2.5 Generate EventID in client.go handleLoggingMessage

## 3. Update Runner Main Loop

- [ ] 3.1 Update processEvent to create Incident from Event
- [ ] 3.2 Write incident.json before agent starts (status=investigating)
- [ ] 3.3 Update incident.json after agent completes (status, exitCode, times)
- [ ] 3.4 Remove separate result.json writing
- [ ] 3.5 Update logging to use both eventId and incidentId

## 4. Update Agent Context

- [ ] 4.1 Rename WriteEventContext to WriteIncidentContext
- [ ] 4.2 Update to accept Incident instead of FaultEvent
- [ ] 4.3 Write incident.json instead of event.json

## 5. Update Storage

- [ ] 5.1 Update IncidentArtifacts to use IncidentJSON instead of EventJSON
- [ ] 5.2 Update readIncidentArtifacts to read incident.json
- [ ] 5.3 Remove ResultJSON from IncidentArtifacts (now in incident.json)
- [ ] 5.4 Update Azure storage blob names
- [ ] 5.5 Update filesystem storage file names

## 6. Update Reporting

- [ ] 6.1 Delete internal/reporting/result.go (absorbed into incident)
- [ ] 6.2 Update SlackNotifier to read from Incident struct
- [ ] 6.3 Update markdown converter to use incidentId

## 7. Update Agent Container

- [ ] 7.1 Update AGENTS.md to reference incident.json
- [ ] 7.2 Update CLAUDE.md to reference incident.json
- [ ] 7.3 Update configs/triage-system-prompt.md to reference incident.json

## 8. Cleanup

- [ ] 8.1 Remove any remaining event.json references
- [ ] 8.2 Run go build to verify compilation
- [ ] 8.3 Update openspec walking-skeleton spec
