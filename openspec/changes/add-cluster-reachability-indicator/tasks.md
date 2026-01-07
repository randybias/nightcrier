# Tasks: Add Cluster Reachability Indicator

## 1. Backend Integration

- [ ] 1.1 Add `ConnectionStatusProvider` interface to adminui package
- [ ] 1.2 Add `statusProvider` field to Server struct and Config
- [ ] 1.3 Create `clusterView` struct that combines ClusterInfo with live status
- [ ] 1.4 Update `handleAdmin` to query connection status for each cluster
- [ ] 1.5 Pass the ConnectionManager to admin server in main.go

## 2. Template Updates

- [ ] 2.1 Add `StatusIndicator()` method to return green/amber/red based on status
- [ ] 2.2 Add STATUS column to Monitored Clusters table header
- [ ] 2.3 Add indicator cell with colored dot to each cluster row
- [ ] 2.4 Add hover tooltip showing the actual status string

## 3. Testing

- [ ] 3.1 Manual test: verify green indicator when cluster is active
- [ ] 3.2 Manual test: verify red indicator when MCP endpoint is unreachable
- [ ] 3.3 Manual test: verify amber indicator during reconnection
