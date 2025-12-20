# Implementation Tasks

## Prerequisites

- [x] 0.1 Verify kubernetes-mcp-server is updated to emit `faultId` field
- [x] 0.2 Verify kubernetes-mcp-server includes `uid` in resource reference

## 1. Update FaultEvent Struct

- [x] 1.1 Update FaultID field comment to reflect upstream source
- [x] 1.2 Remove `omitempty` from FaultID JSON tag (field is required)
- [x] 1.3 Add UID field to ResourceInfo struct with JSON tag `"uid,omitempty"`

## 2. Remove Local FaultID Generation

- [x] 2.1 Remove `github.com/google/uuid` import from client.go (if no longer needed)
- [x] 2.2 Remove `faultEvent.FaultID = uuid.New().String()` line from parseFaultEvent()
- [x] 2.3 Add validation: log warning if FaultID is empty after unmarshal

## 3. Remove DeduplicationKey Method

- [x] 3.1 Delete DeduplicationKey() method from event.go
- [x] 3.2 Search codebase for any usages and update to use FaultID directly

## 4. Update Specifications

- [x] 4.1 Update walking-skeleton spec to reflect FaultID from upstream
- [x] 4.2 Update event parsing scenario to mention FaultID source
- [x] 4.3 Remove any references to local FaultID generation

## 5. Update Documentation

- [x] 5.1 Update project.md to clarify FaultID ownership
- [ ] 5.2 Update README if it mentions FaultID generation (N/A - README does not mention FaultID)

## 6. Validation

- [x] 6.1 Run `go build ./...` to verify compilation
- [x] 6.2 Run `go test ./...` to verify tests pass
- [x] 6.3 Test with updated kubernetes-mcp-server to verify FaultID populated
- [x] 6.4 Verify same fault condition produces same FaultID across re-emissions
- [x] 6.5 Run `openspec validate adopt-upstream-faultid --strict`
