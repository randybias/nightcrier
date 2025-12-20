# Implementation Tasks

## 1. Simplify FaultEvent Structure

- [x] 1.1 Rename EventID field to FaultID in FaultEvent struct
- [x] 1.2 Update JSON tag from `eventId` to `faultId`
- [x] 1.3 Remove deprecated fields from FaultEvent struct (Event, Logs)
- [x] 1.4 Remove EventData type definition
- [x] 1.5 Remove InvolvedObject type definition
- [x] 1.6 Remove ContainerLog type definition
- [x] 1.7 Update struct comments to remove dual-mode references
- [x] 1.8 Update field comment for FaultID to clarify it identifies the raw fault notification

## 2. Update FaultID Generation

- [x] 2.1 Update parseFaultEvent() in client.go to use FaultID field name
- [x] 2.2 Update UUID generation comment to reference FaultID not EventID
- [x] 2.3 Search codebase for references to event.EventID and update to event.FaultID

## 3. Simplify Helper Methods

- [x] 3.1 Remove fallback logic from GetResourceName()
- [x] 3.2 Remove fallback logic from GetResourceKind()
- [x] 3.3 Remove fallback logic from GetNamespace()
- [x] 3.4 Remove fallback logic from GetSeverity()
- [x] 3.5 Remove fallback logic from GetContext()
- [x] 3.6 Remove fallback logic from GetTimestamp()
- [x] 3.7 Remove fallback logic from GetReason()
- [x] 3.8 Delete IsResourceFaultsMode() method entirely
- [x] 3.9 Update method comments to remove mode references

## 4. Simplify Incident Creation

- [x] 4.1 Remove fallback logic from extractResourceInfo() in incident.go
- [x] 4.2 Simplify to only extract from event.Resource field
- [x] 4.3 Update NewFromEvent() comments to remove dual-mode references
- [x] 4.4 Update extractResourceInfo() comments to remove mode references

## 5. Update Specifications

- [x] 5.1 Remove "faults mode" scenarios from walking-skeleton spec
- [x] 5.2 Remove "resource-faults mode" qualifiers (it's the only mode)
- [x] 5.3 Update event parsing requirement to single mode
- [x] 5.4 Remove helper method compatibility requirement
- [x] 5.5 Update configuration table to remove SUBSCRIBE_MODE or clarify it's always resource-faults
- [x] 5.6 Update references to EventID to use FaultID in specs

## 6. Validation

- [x] 6.1 Run `go build ./...` to verify compilation
- [x] 6.2 Run `go test ./...` to verify tests pass
- [x] 6.3 Search for any remaining references: `rg -i "eventid" --type go`
- [x] 6.4 Review git diff to ensure no unintended changes
- [x] 6.5 Verify no references to "faults mode" remain (except in archived proposals)
- [x] 6.6 Run `openspec validate remove-faults-mode-compat --strict`
