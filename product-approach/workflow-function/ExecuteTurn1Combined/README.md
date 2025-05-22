# ExecuteTurn1Combined Lambda Function

`ExecuteTurn1Combined` performs the first turn of the vending machine verification conversation with Amazon Bedrock. It loads prompts and reference images from S3, generates a turn‑1 prompt, invokes Claude Sonnet via the Bedrock Converse API, stores responses, updates DynamoDB, and returns lightweight S3 references to Step Functions.

This function follows the **S3 State Management** pattern: all workflow state lives in S3 and only small references are passed between Step Functions states.

## Project Structure

```
ExecuteTurn1Combined/
├── cmd/
│   └── main.go               # Lambda bootstrap
├── internal/
│   ├── config/               # Environment configuration
│   │   └── config.go
│   ├── handler.go            # Request orchestration
│   ├── services/             # Bedrock, DynamoDB, S3, Prompt
│   │   ├── bedrock.go
│   │   ├── dynamodb.go
│   │   ├── prompt.go
│   │   └── s3.go
│   ├── models/               # Request and response types
│   ├── errors/               # Error helpers
│   ├── logger/               # Structured logging
│   └── utils/                # Correlation ID helpers
├── go.mod
└── go.sum
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AWS_REGION` | AWS region for all clients | `us-east-1` |
| `STATE_BUCKET` | S3 bucket for workflow state | *(required)* |
| `BEDROCK_MODEL` | Claude model ID | *(required)* |
| `ANTHROPIC_VERSION` | Bedrock API version | `bedrock-2023-05-31` |
| `DYNAMODB_VERIFICATION_TABLE` | DynamoDB table for verification status | *(required)* |
| `DYNAMODB_CONVERSATION_TABLE` | DynamoDB table for conversation turns | *(required)* |
| `MAX_TOKENS` | Maximum tokens for Bedrock responses | `24000` |
| `BUDGET_TOKENS` | Token budget for thinking | `16000` |
| `THINKING_TYPE` | Thinking extraction mode | `enable` |
| `MAX_RETRIES` | Maximum retry attempts | `3` |
| `BEDROCK_TIMEOUT_SECONDS` | Bedrock API timeout in seconds | `120` |
| `LOG_LEVEL` | Log level | `INFO` |
| `LOG_FORMAT` | Log output format | `json` |
| `TURN1_PROMPT_VERSION` | Prompt template version | `v1.0` |
| `TEMPLATE_BASE_PATH` | Path to prompt templates | `/opt/templates` |

## Example Input

```json
{
  "verificationId": "verif-12345",
  "verificationContext": {
    "verificationType": "LAYOUT_VS_CHECKING",
    "layoutMetadata": { "layoutId": 42 }
  },
  "s3Refs": {
    "prompts": {
      "system": { "bucket": "state-bucket", "key": "prompts/verif-12345/system-prompt.json" }
    },
    "images": {
      "referenceBase64": { "bucket": "state-bucket", "key": "images/verif-12345/reference-base64.json" }
    }
  }
}
```

## Example Output

```json
{
  "s3Refs": {
    "rawResponse": {
      "bucket": "state-bucket",
      "key": "responses/verif-12345/turn1-raw-response.json"
    },
    "processedResponse": {
      "bucket": "state-bucket",
      "key": "responses/verif-12345/turn1-processed-response.json"
    }
  },
  "status": "TURN1_COMPLETED",
  "summary": {
    "analysisStage": "reference_analysis",
    "processingTimeMs": 12345,
    "tokenUsage": { "input": 1200, "output": 2000, "thinking": 0, "total": 3200 },
    "bedrockRequestId": "abc123"
  }
}
```

## Building

1. Build the Docker image:
   ```bash
   docker build -t execute-turn1-combined .
   ```
2. Push to ECR and update the Lambda (if using the helper script):
   ```bash
   ./retry-docker-build.sh
   ```

See [CHANGELOG.md](CHANGELOG.md) for version history.
