# Troubleshooting

## Agent Failures

**Problem**: Receiving "AI Agent System Degraded" alerts in Slack
- This indicates the circuit breaker threshold has been reached (default: 3 consecutive failures)
- Common causes:
  - LLM API key issues (expired, missing, or incorrect)
  - LLM API rate limiting or service outages
  - Agent timeout (default 300 seconds)
  - Network connectivity issues
  - Resource constraints (CPU, memory)

**Diagnosis Steps**:

1. Check runner logs for failure details:
```bash
# Look for agent failure messages
./runner --log-level debug

# Sample log output:
# WARN agent execution failed validation incident_id=abc-123 reason="agent exited with non-zero code: 1"
# WARN circuit breaker threshold reached, system degraded failure_count=3
```

2. Check the workspace for investigation artifacts:
```bash
# List failed incidents
ls -la ./incidents/

# Check specific incident
cat ./incidents/<incident-id>/incident.json

# Check if output directory exists and has content
ls -la ./incidents/<incident-id>/output/
cat ./incidents/<incident-id>/output/investigation.md
```

3. Verify LLM API key configuration:
```bash
# Check if API key is set
echo $ANTHROPIC_API_KEY | wc -c  # Should be > 50 characters
echo $OPENAI_API_KEY | wc -c
echo $GEMINI_API_KEY | wc -c

# Test API key validity (for Anthropic)
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-3-5-sonnet-20241022","max_tokens":10,"messages":[{"role":"user","content":"test"}]}'
```

4. Check agent execution logs:
```bash
# Check Job logs via kubectl
kubectl logs -n nightcrier -l app=nc-agent-runner --tail=100

# Or watch a specific Job
kubectl logs -n nightcrier job/nc-agent-<incident-id> -f
```

**Common Solutions**:

1. **API Key Issues**: Verify correct API key is set and not expired
2. **Rate Limiting**: Increase delay between incidents or upgrade API tier
3. **Timeouts**: Increase `AGENT_TIMEOUT` (default: 300s)
```bash
export AGENT_TIMEOUT=600
```
4. **Network Issues**: Check firewall rules and proxy settings
5. **Resource Constraints**: Increase container memory/CPU limits

**Recovery**:
Once the underlying issue is fixed, the system will automatically detect the next successful investigation and send a "System Recovered" alert.

**Problem**: Agent failures not triggering system alerts
- Check `NOTIFY_ON_AGENT_FAILURE` is set to `true` (default)
- Verify `SLACK_WEBHOOK_URL` is configured
- Check failure count hasn't reached threshold yet (default: 3 consecutive failures)

**Problem**: Want to inspect failed investigation artifacts
- Set `UPLOAD_FAILED_INVESTIGATIONS=true` to upload failed attempts to storage
- Failed investigations remain in local workspace: `./incidents/<incident-id>/`

## Object Storage Issues

**Problem**: "failed to initialize object storage" or "failed to open bucket"
- Check `OBJECT_STORAGE_URL` format is correct
- Verify credentials are set (Azure: `AZURE_STORAGE_ACCOUNT`/`AZURE_STORAGE_KEY`, S3: `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`)
- Ensure bucket/container exists
- Check network connectivity to storage service
- For S3-compatible: Verify endpoint URL is correct and accessible

**Problem**: "failed to upload blob" or "failed to upload artifact"
- Verify bucket/container exists and has correct permissions
- Check storage account access keys or AWS credentials
- Ensure bucket/container name is valid (lowercase for Azure, DNS-compatible for S3)
- For MinIO: Verify `use_path_style=true` is in the URL

**Problem**: Signed URL returns 403 Forbidden or Access Denied
- Check URL hasn't expired (default: 7 days)
- Verify credentials have permissions to generate signed URLs
- For Azure: Ensure storage key (not SAS token) is used for credentials
- For S3: Ensure IAM permissions allow `s3:GetObject`
- Ensure blob/object exists at the path

**Problem**: Connection refused to MinIO
- Ensure MinIO is running: `docker-compose ps`
- Check ports 9000 (API) and 9001 (Console) are accessible
- Verify endpoint URL in `OBJECT_STORAGE_URL`
- For local development: Use `http://localhost:9000` not `https://`

**Problem**: Connection refused to Azurite
- Ensure Azurite is running: `docker-compose ps`
- Check port 10000 is accessible
- Verify Azurite endpoint in connection (uses `http://` not `https://`)

**Problem**: Bucket/Container not found
- For MinIO: Create bucket via Console (http://localhost:9001) or AWS CLI
  ```bash
  aws --endpoint-url http://localhost:9000 s3 mb s3://incident-reports
  ```
- For Azurite: Create container via Azure CLI
  ```bash
  az storage container create --name incident-reports \
    --connection-string "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
  ```
- For AWS S3: Ensure bucket exists in specified region

## Debug Mode

Enable debug logging to see detailed storage operations:
```bash
./runner --log-level debug
```

Look for these log entries:
- "storage backend initialized" - Shows which backend is active
- "incident artifacts saved to storage" - Shows successful upload
- "failed to save incident to storage" - Shows upload errors

## Multi-Cluster Troubleshooting

### Connection Issues

**Problem**: MCP server unreachable or connection timeouts

**Symptoms**:
```
level=ERROR msg="cluster connection failed" cluster=prod-us-east-1 error="dial tcp: connection refused"
level=INFO msg="cluster reconnection scheduled" cluster=prod-us-east-1 retry_in="2s"
```

**Diagnosis Steps**:

1. Test MCP endpoint from Nightcrier host:
```bash
curl -v http://kubernetes-mcp-server:8080/mcp
# Should return: 405 Method Not Allowed (MCP server only accepts SSE connections)
```

2. Check kubernetes-mcp-server is running:
```bash
kubectl get pods -n mcp-system
kubectl logs -n mcp-system deployment/kubernetes-mcp-server
```

3. Verify network connectivity:
```bash
# If MCP server is in-cluster
kubectl get svc -n mcp-system kubernetes-mcp-server

# If MCP server is external
ping <mcp-server-host>
telnet <mcp-server-host> 8080
```

4. Check firewall rules and network policies

**Solutions**:
- Verify endpoint URL in config (include `/mcp` path)
- Check DNS resolution for cluster service names
- Ensure network policies allow ingress to MCP server
- Verify MCP server logs show no errors
- Connection will automatically retry with exponential backoff

### Permission Issues

**Problem**: kubectl auth can-i failures during startup

**Symptoms**:
```
level=ERROR msg="permission validation failed" cluster=prod error="kubectl auth can-i failed: exit status 1"
```

**Diagnosis Steps**:

1. Test kubeconfig manually:
```bash
KUBECONFIG=./kubeconfigs/prod-us-east-1-readonly.yaml kubectl get pods
```

2. Check ServiceAccount exists:
```bash
kubectl get sa -n kube-system kubernetes-triage-readonly
kubectl get secret -n kube-system kubernetes-triage-readonly-token
```

3. Verify RBAC bindings:
```bash
kubectl get clusterrolebinding | grep kubernetes-triage
kubectl describe clusterrolebinding kubernetes-triage-readonly-view
```

4. Test specific permissions:
```bash
KUBECONFIG=./kubeconfigs/prod-readonly.yaml \
  kubectl auth can-i get pods
KUBECONFIG=./kubeconfigs/prod-readonly.yaml \
  kubectl auth can-i pods/log
```

**Solutions**:
- Recreate ServiceAccount and token secret
- Reapply RBAC ClusterRoleBindings
- Regenerate kubeconfig using extraction script
- Check ServiceAccount token hasn't expired
- Verify kubectl is in PATH and accessible

**Problem**: Insufficient permissions warning

**Symptoms**:
```
level=WARN msg="cluster has permission warnings" cluster=prod warnings="cannot get nodes (cluster-wide visibility limited)"
```

**Impact**:
- Agent will still run but with limited investigation capabilities
- Some diagnostic commands may fail inside agent container
- Investigation report may be incomplete

**Solutions**:
- Grant additional RBAC permissions (nodes, secrets, etc.)
- Accept limited functionality for security-conscious deployments
- Review `incident_cluster_permissions.json` to see exact limitations

### Kubeconfig Problems

**Problem**: Kubeconfig file not found

**Symptoms**:
```
level=ERROR msg="failed to initialize connection manager" error="cluster prod-us-east-1: kubeconfig not found: ./kubeconfigs/prod-readonly.yaml"
```

**Solutions**:
- Verify file exists: `ls -la ./kubeconfigs/`
- Check file path in config matches actual filename
- Ensure kubeconfig was extracted using setup script
- Check file permissions (should be readable by Nightcrier process)

**Problem**: Kubeconfig authentication fails

**Symptoms**:
```
level=ERROR msg="permission validation failed" error="Unable to connect to the server: x509: certificate signed by unknown authority"
```

**Solutions**:
- Verify cluster CA certificate in kubeconfig
- Test with kubectl: `kubectl --kubeconfig=<file> cluster-info`
- Regenerate kubeconfig from ServiceAccount token
- Check cluster API server is accessible from Nightcrier host

### Triage Disabled Behavior

**Problem**: Events received but no investigations performed

**Symptoms**:
```
level=INFO msg="fault event received" cluster=staging namespace=default resource=pod/webapp
level=INFO msg="triage disabled for cluster - skipping agent execution" cluster=staging reason="triage.enabled=false or no kubeconfig"
```

**Expected Behavior**: This is intentional when `triage.enabled: false`

**When this happens**:
- Fault events are logged for visibility
- No workspace is created
- No AI agent is spawned
- No Slack notification is sent
- No storage upload occurs

**To enable triage**:
1. Create kubeconfig for cluster (see Kubeconfig Setup section)
2. Set `triage.enabled: true` in config
3. Restart Nightcrier
4. Verify permission validation succeeds

### Reading Logs for Cluster-Specific Issues

Enable debug logging:
```bash
./nightcrier --log-level debug
```

**Key log fields to monitor**:
- `cluster` - Which cluster the event came from
- `incident_id` - Unique incident identifier
- `kubeconfig` - Which kubeconfig is being used
- `minimum_met` - Whether minimum permissions are satisfied
- `triage_enabled` - Whether triage is enabled for cluster

**Example log analysis**:

```bash
# Filter logs for specific cluster
./nightcrier 2>&1 | grep 'cluster=prod-us-east-1'

# Check permission validation results
./nightcrier 2>&1 | grep 'permissions validated'

# Monitor connection status
./nightcrier 2>&1 | grep 'cluster connection'

# Track triage skip events
./nightcrier 2>&1 | grep 'triage disabled'
```

### Understanding Connection Status

Connection lifecycle states:

| State | Meaning | Next Steps |
|-------|---------|------------|
| `disconnected` | Initial state | Connecting... |
| `connecting` | TCP connection in progress | Wait for connected |
| `connected` | Connected to MCP server | Subscribing... |
| `subscribing` | Requesting fault event stream | Wait for active |
| `active` | Receiving events | Normal operation |
| `failed` | Connection error occurred | Auto-retry with backoff |

**Check connection status** (future feature in Phase 4):
```bash
curl http://localhost:9090/health/clusters
```

**Reconnection behavior**:
- Initial backoff: 1 second
- Maximum backoff: 60 seconds
- Multiplier: 2.0 (exponential)
- Jitter: 10% (randomization)
- Continues indefinitely until successful

