# Fake LaunchDarkly Service

A standalone HTTP service that emulates LaunchDarkly's public API endpoints for local development and testing. This service reads feature flag configurations from a JSON file and provides the `/api/eval/contexts` endpoint that Go-based services expect.

## Features

- ✅ Emulates LaunchDarkly API flag evaluation endpoints
- ✅ JSON-based configuration (no external API dependencies)
- ✅ Per-institution flag targeting
- ✅ Hot-reload configuration via admin endpoint
- ✅ Simple HTTP REST API
- ✅ Docker support
- ✅ Health check endpoint

## Quick Start

### Local Development

```bash
cd fake-launchdarkly-service

# Build the service
go build -o fake-launchdarkly-service main.go

# Run with default config
./fake-launchdarkly-service

# Or specify custom port/config
./fake-launchdarkly-service -port 4000 -config config.json
```

The service will start on `http://localhost:4000` by default.

### Docker

```bash
# Build the Docker image
docker build -t fake-launchdarkly-service .

# Run the container
docker run -p 4000:4000 -v $(pwd)/config.json:/app/config.json fake-launchdarkly-service
```

### Docker Compose

Add to your `docker-compose.yml`:

```yaml
fake-launchdarkly-service:
  build:
    context: ./fake-launchdarkly-service
  ports:
    - "4000:4000"
  volumes:
    - ./fake-launchdarkly-service/config.json:/app/config.json
  environment:
    - CONFIG_PATH=/app/config.json
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:4000/health"]
    interval: 10s
    timeout: 3s
    retries: 3
    start_period: 5s
```

## API Endpoints

### Flag Evaluation

**POST /api/eval/contexts**

Evaluates a feature flag for a given institution.

Request:
```json
{
  "context": {
    "kind": "institution",
    "key": "episystest"
  },
  "flag": "wires-cycle-date-override"
}
```

Response:
```json
{
  "value": true,
  "variation": 1,
  "reason": {
    "kind": "TARGETING_MATCH"
  },
  "trackEvents": false,
  "debugEventsUntilDate": null
}
```

### List All Flags

**GET /api/flag**

Returns all configured flags and their definitions.

### Get Single Flag

**GET /api/flag/{key}**

Returns a specific flag definition.

Example:
```bash
curl http://localhost:4000/api/flag/wires-cycle-date-override
```

### Reload Configuration

**POST /admin/reload**

Reloads the configuration from the JSON file. Useful for updating flags without restarting.

```bash
curl -X POST http://localhost:4000/admin/reload
```

### Health Check

**GET /health**

Returns service health status.

```bash
curl http://localhost:4000/health
```

## Configuration

Edit `config.json` to define feature flags:

```json
{
  "flags": {
    "flag-key": {
      "key": "flag-key",
      "name": "Human Readable Name",
      "description": "What this flag does",
      "variations": [false, true],
      "offVariation": 0,
      "targets": {
        "institution-id-1": [1],
        "institution-id-2": [0]
      },
      "version": 1,
      "deleted": false
    }
  }
}
```

### Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `key` | string | Unique flag identifier |
| `name` | string | Human-readable name |
| `description` | string | What the flag does |
| `variations` | array | Possible flag values (typically `[false, true]`) |
| `offVariation` | int | Index of the "off" variation (default: 0) |
| `targets` | object | Institution ID → variation index mappings |
| `version` | int | Configuration version |
| `deleted` | bool | Whether flag is deleted |

### Example: Targeting Specific Institutions

```json
"wires-cycle-date-override": {
  "key": "wires-cycle-date-override",
  "variations": [false, true],
  "offVariation": 0,
  "targets": {
    "a3badfa6-00eb-4b0f-9881-d9c6ba95ca8b": [1],
    "episystest": [1],
    "bdb": [0]
  }
}
```

This enables the flag for institutions `a3badfa6-00eb-4b0f-9881-d9c6ba95ca8b` and `episystest`, but keeps it disabled (off) for `bdb`.

## Using with Elec Services

### Environment Variable Configuration

Set the LaunchDarkly API base URL to point to the fake service:

```bash
export LAUNCHDARKLY_BASE_URL=http://localhost:4000
export LAUNCHDARKLY_KEY=sdk-fake-key
```

### In docker-compose.yml

```yaml
elec-wire-service:
  environment:
    - LAUNCHDARKLY_BASE_URL=http://fake-launchdarkly-service:4000
    - LAUNCHDARKLY_KEY=sdk-fake-key
  depends_on:
    fake-launchdarkly-service:
      condition: service_healthy
```

## Updating Flags at Runtime

You can modify `config.json` and then reload without restarting the service:

```bash
# Edit config.json
vim config.json

# Reload
curl -X POST http://localhost:4000/admin/reload
```

Or use the environment variable to specify a config path:

```bash
CONFIG_PATH=/path/to/custom-config.json ./fake-launchdarkly-service
```

## Testing

### With curl

```bash
# Check health
curl http://localhost:4000/health

# Evaluate a flag
curl -X POST http://localhost:4000/api/eval/contexts \
  -H "Content-Type: application/json" \
  -d '{
    "context": {"kind": "institution", "key": "episystest"},
    "flag": "wires-cycle-date-override"
  }'

# List all flags
curl http://localhost:4000/api/flag

# Get specific flag
curl http://localhost:4000/api/flag/wires-cycle-date-override

# Reload config
curl -X POST http://localhost:4000/admin/reload
```

### With Go Tests

```go
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestFlagEvaluation(t *testing.T) {
	// Assume service is running on localhost:4000
	client := &http.Client{}
	
	req := map[string]interface{}{
		"context": map[string]string{
			"kind": "institution",
			"key":  "episystest",
		},
		"flag": "wires-cycle-date-override",
	}
	
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "http://localhost:4000/api/eval/contexts", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, _ := client.Do(httpReq)
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}
```

## Limitations

This is a **development tool only**:

- ✅ Flag evaluation based on institution targeting
- ✅ Simple JSON-based configuration
- ❌ No user-level targeting
- ❌ No custom rules
- ❌ No analytics
- ❌ No A/B testing
- ❌ No prerequisite flags
- ❌ No experiments

For production environments, use the real LaunchDarkly service with proper credentials.

## Troubleshooting

### Port Already in Use

```bash
# Find what's using port 4000
lsof -i :4000

# Or use a different port
./fake-launchdarkly-service -port 5000
```

### Config Not Loading

Check that the JSON is valid:

```bash
# Validate JSON
cat config.json | jq .
```

### Service Not Responding

```bash
# Check if it's running
curl http://localhost:4000/health

# Check logs (if running in Docker)
docker logs <container-id>
```

## License

Part of the JH Wires testing infrastructure.
