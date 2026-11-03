# LocalCloud Architecture Decision Records (ADR)

## ADR-001: Go as Core Gateway and Emulation Engine
- **Context**: AWS wire protocols require parsing multiple formats (Query XML, JSON 1.0/1.1, REST-XML, REST-JSON, EC2 Query) and computing HMAC-SHA256 signatures with sub-millisecond latencies.
- **Decision**: Implement the Edge Gateway, protocol engine, and service adapters in Go.
- **Consequence**: Single self-contained static binary, near-instant startup (< 20ms), minimal memory footprint (< 30 MB RAM), and zero external runtime dependencies.

## ADR-002: Dual Protocol Handling for Amazon SQS
- **Context**: Older AWS SDKs and CLI versions query SQS using XML form-urlencoded bodies, while modern SDKs send `application/x-amz-json-1.0` with `X-Amz-Target: AmazonSQS.*`.
- **Decision**: SQS adapter detects the content-type and target header to dynamically serialize responses in either JSON 1.0 or Query XML.
- **Consequence**: Maximum backward and forward compatibility with any version of the AWS CLI and boto3.

## ADR-003: In-Memory Ring Buffer + JSONL CloudTrail Persistence
- **Context**: The Live API Inspector requires instant real-time event streaming via Server-Sent Events (SSE), while learners also need audit logs saved to disk.
- **Decision**: Maintain a concurrent ring buffer of recent events in memory and simultaneously append each event to `./data/cloudtrail/events.jsonl`.
- **Consequence**: Zero database overhead, high throughput, and seamless live inspection in the browser console.
