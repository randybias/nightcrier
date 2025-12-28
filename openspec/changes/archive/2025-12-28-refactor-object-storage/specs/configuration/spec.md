# configuration Spec Delta

## ADDED Requirements

### Requirement: Object Storage Configuration Section

The system SHALL support object storage configuration via config file and environment variables, replacing Azure-specific configuration.

#### Scenario: Config file object_storage section
- **GIVEN** config.yaml contains an `object_storage:` section
- **WHEN** the application starts
- **THEN** object storage SHALL be configured from the config file values

#### Scenario: Environment variable override
- **GIVEN** config.yaml contains `object_storage.url: "azblob://container1"`
- **AND** environment variable `OBJECT_STORAGE_URL` is set to `s3://bucket2`
- **WHEN** the application starts
- **THEN** the environment variable SHALL take precedence
- **AND** S3 storage SHALL be configured

#### Scenario: Config file example
- **GIVEN** the example config file `configs/config.example.yaml`
- **THEN** it SHALL include documented `object_storage:` section with:
  - `url` field with examples for Azure, S3, and S3-compatible endpoints
  - `signed_url_expiry` field with default value documented
  - Comments explaining credential requirements per provider

### Requirement: Storage URL Configuration

The system SHALL support URL-based cloud storage configuration via config file (`object_storage.url`) or environment variable (`OBJECT_STORAGE_URL`).

#### Scenario: Storage URL from config file
- **GIVEN** config.yaml contains `object_storage.url: "s3://my-bucket?region=us-east-1"`
- **WHEN** the application starts
- **THEN** S3 storage SHALL be configured using the URL

#### Scenario: Storage URL from environment variable
- **GIVEN** `OBJECT_STORAGE_URL` environment variable is set
- **WHEN** the application starts
- **THEN** cloud storage SHALL be configured using the Go CDK URL format
- **AND** the URL scheme SHALL determine the storage provider

#### Scenario: Storage URL with Azure scheme
- **GIVEN** storage URL is set to `azblob://container-name`
- **WHEN** the application validates configuration
- **THEN** Azure Blob Storage SHALL be configured
- **AND** Azure credentials SHALL be loaded from `AZURE_STORAGE_ACCOUNT` and `AZURE_STORAGE_KEY` environment variables

#### Scenario: Storage URL with S3 scheme
- **GIVEN** storage URL is set to `s3://bucket-name?region=us-east-1`
- **WHEN** the application validates configuration
- **THEN** S3 storage SHALL be configured
- **AND** AWS credentials SHALL be loaded from the standard credential chain (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, or IAM role)

#### Scenario: Storage URL with S3 custom endpoint for MinIO/RustFS
- **GIVEN** storage URL is set to `s3://bucket?endpoint=http://minio:9000&use_path_style=true&disable_https=true&awssdk=v2`
- **WHEN** the application validates configuration
- **THEN** S3-compatible storage SHALL be configured with the custom endpoint
- **AND** path-style URLs SHALL be enabled
- **AND** HTTP (non-TLS) connections SHALL be allowed

#### Scenario: Storage URL not configured (default)
- **GIVEN** neither `object_storage.url` nor `OBJECT_STORAGE_URL` is configured
- **WHEN** the application starts
- **THEN** the application SHALL start successfully
- **AND** filesystem storage SHALL be used (existing behavior)
- **AND** artifacts SHALL be stored under `workspace_root`

#### Scenario: Invalid storage URL scheme
- **GIVEN** storage URL is set to an unsupported scheme (e.g., `gcs://bucket`)
- **WHEN** the application validates configuration
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL indicate supported schemes are `azblob://` and `s3://`

### Requirement: Signed URL Expiry Configuration

The system SHALL support configurable expiration for signed URLs via config file (`object_storage.signed_url_expiry`) or environment variable (`OBJECT_STORAGE_SIGNED_URL_EXPIRY`).

#### Scenario: Custom signed URL expiry from config
- **GIVEN** config.yaml contains `object_storage.signed_url_expiry: "24h"`
- **WHEN** signed URLs are generated
- **THEN** the URLs SHALL expire after 24 hours

#### Scenario: Custom signed URL expiry from environment
- **GIVEN** `OBJECT_STORAGE_SIGNED_URL_EXPIRY` is set to `48h`
- **WHEN** signed URLs are generated
- **THEN** the URLs SHALL expire after 48 hours

#### Scenario: Default signed URL expiry
- **GIVEN** signed URL expiry is not configured
- **WHEN** signed URLs are generated
- **THEN** the URLs SHALL expire after 168 hours (7 days)

#### Scenario: Invalid expiry duration format
- **GIVEN** signed URL expiry is set to an invalid duration string (e.g., `7days`)
- **WHEN** the application validates configuration
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL indicate valid Go duration format (e.g., `168h`, `24h`, `30m`)

### Requirement: Provider Credential Configuration

The system SHALL support credentials in config file and environment variables with standard precedence (env > config).

#### Scenario: S3 credentials in config file
- **GIVEN** config.yaml contains:
  ```yaml
  object_storage:
    url: "s3://bucket"
    aws_access_key_id: "AKIAIOSFODNN7EXAMPLE"
    aws_secret_access_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  ```
- **WHEN** the application starts
- **THEN** the credentials SHALL be loaded from the config file

#### Scenario: Azure credentials in config file
- **GIVEN** config.yaml contains:
  ```yaml
  object_storage:
    url: "azblob://container"
    azure_storage_account: "myaccount"
    azure_storage_key: "base64encodedkey=="
  ```
- **WHEN** the application starts
- **THEN** the credentials SHALL be loaded from the config file

#### Scenario: Environment variable overrides config file
- **GIVEN** config.yaml contains `object_storage.aws_access_key_id: "config-key"`
- **AND** environment variable `AWS_ACCESS_KEY_ID` is set to `"env-key"`
- **WHEN** the application starts
- **THEN** the environment variable value SHALL be used
- **AND** the config file value SHALL be ignored

#### Scenario: IAM role fallback for AWS
- **GIVEN** storage URL uses `s3://` scheme
- **AND** no explicit AWS credentials are configured
- **AND** application is running on EC2/ECS/EKS with IAM role
- **WHEN** the application starts
- **THEN** IAM role credentials SHALL be used automatically

### Requirement: Credential Validation

The system SHALL validate that required credentials are present when object storage is configured.

#### Scenario: Missing Azure credentials
- **GIVEN** storage URL is set to `azblob://container`
- **AND** neither `azure_storage_account` nor `AZURE_STORAGE_ACCOUNT` is configured
- **WHEN** the application validates configuration
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state that Azure credentials are required

#### Scenario: Missing S3 credentials without IAM
- **GIVEN** storage URL is set to `s3://bucket`
- **AND** neither `aws_access_key_id` nor `AWS_ACCESS_KEY_ID` is configured
- **AND** application is NOT running with IAM role
- **WHEN** the application validates configuration
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state that S3 credentials are required

#### Scenario: Partial Azure credentials
- **GIVEN** storage URL is set to `azblob://container`
- **AND** `azure_storage_account` is configured
- **AND** `azure_storage_key` is NOT configured
- **WHEN** the application validates configuration
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state that both account and key are required

#### Scenario: Partial S3 credentials
- **GIVEN** storage URL is set to `s3://bucket`
- **AND** `aws_access_key_id` is configured
- **AND** `aws_secret_access_key` is NOT configured
- **WHEN** the application validates configuration
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state that both access key and secret are required

#### Scenario: Memory storage requires no credentials
- **GIVEN** storage URL is set to `mem://`
- **AND** no credentials are configured
- **WHEN** the application validates configuration
- **THEN** validation SHALL pass
- **AND** the application SHALL start successfully

#### Scenario: Filesystem fallback requires no credentials
- **GIVEN** storage URL is NOT configured
- **WHEN** the application validates configuration
- **THEN** validation SHALL pass
- **AND** filesystem storage SHALL be used
