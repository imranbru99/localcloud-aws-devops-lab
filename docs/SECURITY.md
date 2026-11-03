# LocalCloud Security Model

LocalCloud is engineered for educational and local development usage. It incorporates AWS-compatible security models to teach secure cloud development practices:

1. **Default Localhost Binding**: The gateway binds exclusively to local interfaces (`127.0.0.1`) unless explicitly configured with environment variables.
2. **SigV4 Canonical Request Verification**: All requests evaluate SigV4 signatures, preventing request tampering and teaching cryptographic signature validation.
3. **IAM Least Privilege Engine**: Access control is evaluated against attached policies before delegating to service handlers. Deny rules take precedence over allow rules.
4. **Isolated Data Directory**: All customer data and resources are stored under `${LOCALCLOUD_DATA_DIR:-./data}` with strict local file permissions.
5. **No External AWS Credentials Required**: LocalCloud uses local credentials (`test` / `test` or generated local keys). Real AWS secret keys must never be entered into LocalCloud.
