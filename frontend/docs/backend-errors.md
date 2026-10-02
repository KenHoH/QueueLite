# API error handling

Application errors use a JSON envelope:

```json
{"error":{"code":"ACTIVE_QUEUE_EXISTS","message":"user already has active queue in this business"}}
```

Some authentication middleware returns plain text. The frontend normalizes JSON and plain-text failures into `APIError`, which carries the HTTP status, code, message and failure kind. Do not depend on message text for control flow or display raw internal failures to users.

## Status mapping

The shared backend error writer maps application error kinds as follows:

| Kind | HTTP status |
| --- | --- |
| Invalid input | 400 |
| Unauthorized | 401 |
| Forbidden | 403 |
| Not found | 404 |
| Conflict | 409 |
| Not implemented | 501 |
| Internal or unclassified | 500 |

Internal errors use the public message `internal server error`. Error codes alone do not determine the status: handlers and services select the error kind for each operation.

## Common failures

| Codes | Meaning and handling |
| --- | --- |
| `INVALID_FORMAT`, `INVALID_PHONE_NUMBER`, `INVALID_EMAIL`, `INVALID_PASSWORD` | Correct the submitted fields. Phone validation accepts Indonesian mobile numbers. |
| `INVALID_CREDENTIALS` | Ask the user to check their login details. |
| `UNAUTHORIZED` | Authentication is required. Distinguish account access from guest or queue-cookie access. |
| `BUSINESS_ACCESS_DENIED` | The account lacks the required business permission. |
| `BUSINESS_NOT_FOUND`, `QUEUE_NOT_FOUND`, `COUNTER_NOT_FOUND` | The requested resource is unavailable. |
| `ACTIVE_QUEUE_EXISTS`, `PHONE_ALREADY_REGISTERED` | An active ticket already exists for the account or phone number. |
| `BUSINESS_UNAVAILABLE`, `BUSINESS_QUEUE_FULL` | Queue admission is currently unavailable. |
| `COUNTER_STATE_CHANGED` | Refresh counter state before another action. |
| `SUBSCRIPTION_ADMIN_REQUIRED` | Account sessions cannot administer subscriptions or quotas. |

Frontend-only failures include `CLIENT_NETWORK_ERROR`, `CLIENT_TIMEOUT` and `CLIENT_INVALID_RESPONSE`. A failed or malformed mutation response can be ambiguous; check the current state before submitting again. Requests are not retried automatically.

## Source references

The current implementation is authoritative for operation-specific codes and permissions:

- [Error kinds](../../backend/internal/apperror/error.go), [status mapping](../../backend/internal/adapter/http/error_mapper.go) and [response writer](../../backend/internal/adapter/http/error_writer.go).
- [Route registration](../../backend/internal/bootstrap/routes.go), [business and counter permissions](../../backend/internal/operations/http.go) and [subscription account routes](../../backend/internal/subscription/inbound/http/account_routes.go).
- [Frontend error normalization](../src/api/errors.ts) and [response validation](../src/api/responseSchema.ts).
