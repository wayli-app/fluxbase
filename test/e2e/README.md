# End-to-End Test Suite

This directory contains comprehensive end-to-end tests for Fluxbase, covering all major features with a focus on **Row-Level Security (RLS)**, **Authentication**, and **Multi-Tenancy**.

## Test Coverage

**54 test files** covering all major Fluxbase features. Tests require a running PostgreSQL instance with the Fluxbase schema bootstrapped.

### Test Categories

#### Authentication & Security (8 files)
| File | Coverage |
|------|----------|
| `auth_test.go` | Signup, signin, signout, token refresh, session management |
| `auth_rest_test.go` | REST API authentication flows, token propagation |
| `two_factor_test.go` | TOTP/MFA enrollment, verification, recovery codes |
| `oauth_test.go` | OAuth2 flow (Google, GitHub, etc.), token exchange |
| `oauth_dex_test.go` | OIDC via Dex integration test |
| `otp_hash_test.go` | OTP code hashing (SHA-256), verification |
| `invitation_hash_test.go` | Invitation token hashing, dual-read migration |
| `ssrf_protection_test.go` | SSRF blocking for metadata endpoints |
| `ssrf_allowed_domains_test.go` | SSRF allowlist functionality |

#### REST API & Query (4 files)
| File | Coverage |
|------|----------|
| `rest_test.go` | CRUD operations, 20+ query operators, aggregations, upsert, batch |
| `graphql_test.go` | GraphQL queries, mutations, filtering, relations |
| `rpc_test.go` | RPC procedure invocation, scheduling, RBAC |
| `sql_editor_test.go` | SQL editor execution |

#### Row-Level Security (12 files)
| File | Coverage |
|------|----------|
| `rls_test.go` | User isolation, insert/update/delete protection |
| `rls_new_test.go` | Expanded RLS scenarios |
| `storage_rls_providers_test.go` | RLS across storage providers |
| `storage_rls_public_test.go` | Public bucket access |
| `storage_rls_service_key_test.go` | Service key bypass |
| `storage_rls_isolation_test.go` | Cross-user isolation |
| `storage_rls_ownership_test.go` | Object ownership |
| `storage_rls_sharing_test.go` | Shared object access |
| `storage_rls_admin_test.go` | Admin access patterns |
| `storage_rls_bucket_settings_test.go` | Bucket-level policy enforcement |

#### Multi-Tenancy (8 files)
| File | Coverage |
|------|----------|
| `tenant_isolation_test.go` | Cross-tenant data isolation |
| `tenant_config_test.go` | Per-tenant configuration |
| `tenant_deletion_cascade_test.go` | Tenant deletion cleanup |
| `tenant_migrations_isolation_test.go` | Migration isolation per tenant |
| `tenant_migration_schema_isolation_test.go` | Schema migration isolation |
| `has_tenant_access_test.go` | Access checking |
| `realtime_tenant_isolation_test.go` | Realtime tenant scoping |

#### Storage (4 files)
| File | Coverage |
|------|----------|
| `storage_local_test.go` | Local filesystem backend |
| `storage_s3_test.go` | S3/MinIO backend |
| `storage_transform_test.go` | Image transformations |

#### Edge Functions (4 files)
| File | Coverage |
|------|----------|
| `functions_execution_test.go` | Function invoke, response handling |
| `functions_reload_test.go` | Hot reload from filesystem |
| `functions_auth_test.go` | Auth context injection |
| `functions_auth_simple_test.go` | Basic auth propagation |

#### AI & MCP (4 files)
| File | Coverage |
|------|----------|
| `ai_chatbot_test.go` | Chatbot creation, conversation, SQL generation |
| `ai_chatbot_config_test.go` | Chatbot configuration, provider settings |
| `ai_sql_validation_test.go` | SQL validation safety checks |
| `mcp_test.go` | MCP tool execution, resources, scopes |

#### Jobs & Webhooks (4 files)
| File | Coverage |
|------|----------|
| `job_retry_test.go` | Retry logic, backoff |
| `jobs_get_test.go` | Job listing, status |
| `webhook_trigger_test.go` | Webhook delivery on DB events |
| `webhook_trigger_debug_test.go` | Debug mode, delivery logging |

#### Infrastructure (8 files)
| File | Coverage |
|------|----------|
| `health_test.go` | Health endpoints |
| `setup_test.go` | Initial setup flow |
| `admin_test.go` | Admin API endpoints |
| `ratelimit_test.go` | Rate limiting enforcement |
| `logging_postgres_test.go` | Log ingestion and querying |
| `migration_lock_test.go` | Concurrent migration locking |
| `postgis_test.go` | PostGIS extension support |
| `realtime_test.go` | WebSocket connections, presence, broadcast |

#### Examples (2 files)
| File | Coverage |
|------|----------|
| `example_test.go` | Basic usage examples |
| `transaction_example_test.go` | Transaction patterns |

## Running Tests

### Prerequisites

```bash
# Start the database (DevContainer handles this automatically)
docker compose up -d postgres

# Ensure schema is bootstrapped
make db-reset
```

### Run All E2E Tests

```bash
# Run all end-to-end tests
go test -v ./test/e2e/...

# Run with coverage
go test -v -tags=integration -cover -coverprofile=coverage.out ./test/e2e/...
go tool cover -html=coverage.out -o coverage.html

# Run with race detection
go test -v -race ./test/e2e/...
```

### Run Specific Test Suites

```bash
# REST API tests
go test -v ./test/e2e/ -run TestREST

# Authentication tests
go test -v ./test/e2e/ -run TestAuth

# RLS tests
go test -v ./test/e2e/ -run TestRLS

# Tenant isolation tests
go test -v ./test/e2e/ -run TestTenant
```

## Related Documentation

- [RLS Implementation](../../internal/middleware/rls.go)
- [Auth Service](../../internal/auth/service.go)
- [REST Handler](../../internal/api/rest_handler.go)
- [Functions Handler](../../internal/functions/handler.go)
