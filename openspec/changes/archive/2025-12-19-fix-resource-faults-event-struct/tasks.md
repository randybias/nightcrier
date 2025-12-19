## 1. Implementation

- [x] 1.1 Add Resource struct for flat resource-faults format
- [x] 1.2 Add flat fields to FaultEvent struct (Resource, Context, FaultType, Severity, Timestamp)
- [x] 1.3 Update GetResourceName() to check both structures
- [x] 1.4 Update GetResourceKind() to check both structures
- [x] 1.5 Update GetNamespace() to check both structures
- [x] 1.6 Update GetSeverity() to return faultType or severity field
- [x] 1.7 Add GetContext() helper method
- [x] 1.8 Add GetTimestamp() helper method
- [x] 1.9 Update client.go to use new helper methods
- [x] 1.10 Build verification passed
