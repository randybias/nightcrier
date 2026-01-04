# Tasks: Enhance Admin UI

## 1. Clusters Pane

- [x] 1.1 Add ClusterInfo type to adminui package (name, environment, endpoint, triage_enabled)
- [x] 1.2 Update Config to accept cluster list
- [x] 1.3 Update adminData struct to include Clusters
- [x] 1.4 Add clusters pane to admin.html template (above Running Triages)
- [x] 1.5 Pass monitored_clusters config to admin UI in main.go

## 2. Delete Incident

- [x] 2.1 Add DeleteIncident method to adminui Store
- [x] 2.2 Add POST /admin/incidents/{id}/delete handler
- [x] 2.3 Add Delete button to incidents table in template
- [x] 2.4 Add JavaScript confirm() before delete

## 3. Cancel Triage

- [x] 3.1 Add K8sJobCanceller interface to adminui package
- [x] 3.2 Add CancelExecution method to adminui Store (marks cancelled)
- [x] 3.3 Implement job cancellation in k8s package (CancelJob method)
- [x] 3.4 Add POST /admin/triages/{id}/cancel handler
- [x] 3.5 Add Cancel button to running triages table in template
- [x] 3.6 Add JavaScript confirm() before cancel

## 4. Testing

- [ ] 4.1 Manual test: clusters pane displays correctly
- [ ] 4.2 Manual test: delete incident removes from list
- [ ] 4.3 Manual test: cancel triage kills job and removes from running list
