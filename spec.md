# API Spec (Request/Response)

This document lists all HTTP paths from router/router.go with request/response payloads derived from handler structs.

## Response envelope (all JSON responses)

Responses are wrapped by the common response envelope. Successful responses include:

```json
{
  "code": "SUCCESS",
  "message": "SUCCESS",
  "data": {},
  "traceId": "string (optional)"
}
```

Error responses use the same envelope with `code`/`message` indicating the error and `data` omitted or null. HTTP status codes are set per handler.

---

## Health

### GET /liveness

- Request: none
- Response: health payload (version/commit)

### GET /readiness

- Request: none
- Response: readiness payload

### GET /metrics

- Request: none
- Response: Prometheus metrics text

---

## Identity Flow

### POST /api/v1/identity-flow/auth

**Authentication**: Public (no auth required)

Request:

```json
{
  "googleAccessToken": "string"
}
```

Response data:

```json
{
  "username": "string",
  "email": "string",
  "role": "string",
  "profileImage": "string",
  "accessToken": "string"
}
```

### POST /api/v1/identity-flow/get-organization

**Authentication**: Required (JWT)

Request: none (uses claims from JWT)

Response data:

```json
{
  "organizationId": "uuid",
  "name": "string",
  "status": "string",
  "role": "string",
  "joinedAt": "time",
  "createdAt": "time",
  "updatedAt": "time"
}
```

### POST /api/v1/identity-flow/get-member

**Authentication**: Required (JWT)

Request: none (uses claims from JWT)

Response data:

```json
{
  "memberId": "uuid",
  "username": "string",
  "email": "string",
  "hashedEmail": "string",
  "status": "string",
  "createdAt": "time",
  "updatedAt": "time"
}
```

---

## Register Flow

### POST /api/v1/register-flow/register

**Authentication**: Public (no auth required)

Request:

```json
{
  "googleAccessToken": "string"
}
```

Response data:

```json
{
  "username": "string",
  "email": "string",
  "role": "string",
  "profileImage": "string",
  "accessToken": "string"
}
```

---

## Insight Flow

### POST /api/v1/insight-flow/create-research

**Authentication**: Required (JWT)

Request:

```json
{
  "productDescription": "string",
  "productObjective": "string",
  "flows": [
    {
      "flowKey": "string",
      "orderNo": 1,
      "steps": [
        {
          "stepKey": "string",
          "orderNo": 1
        }
      ]
    }
  ]
}
```

Response data:

```json
{
  "insightResearchId": "uuid",
  "flows": [
    {
      "flowId": "uuid",
      "flowKey": "string",
      "orderNo": 1,
      "steps": [
        {
          "stepId": "uuid",
          "stepKey": "string",
          "orderNo": 1,
          "status": "string"
        }
      ]
    }
  ]
}
```

### POST /api/v1/insight-flow/submit-research

**Authentication**: Required (JWT)

Request:

```json
{
  "insightResearchId": "uuid"
}
```

Response data:

```json
{
  "insightResearchId": "uuid",
  "status": "string"
}
```

### POST /api/v1/insight-flow/get-research

**Authentication**: Required (JWT)

Request:

```json
{
  "insightResearchId": "uuid"
}
```

Response data:

```json
{
  "insightResearchId": "uuid",
  "organizationId": "uuid",
  "productDescription": "string",
  "productObjective": "string",
  "status": "string",
  "flows": [
    {
      "flowId": "uuid",
      "flowKey": "string",
      "orderNo": 1,
      "steps": [
        {
          "stepId": "uuid",
          "stepKey": "string",
          "orderNo": 1,
          "status": "string"
        }
      ]
    }
  ]
}
```

### POST /api/v1/insight-flow/list-research

**Authentication**: Required (JWT)

Request: none (uses organizationId from JWT claims)

Response data:

```json
{
  "researches": [
    {
      "insightResearchId": "string",
      "organizationId": "string",
      "productDescription": "string",
      "productObjective": "string",
      "status": "string"
    }
  ]
}
```

### POST /api/v1/insight-flow/list-persona

**Authentication**: Required (JWT)

Request:

```json
{
  "insightResearchId": "uuid"
}
```

Response data:

```json
{
  "personas": [
    {
      "personaId": "string",
      "insightResearchId": "string",
      "description": "string",
      "goals": ["string"],
      "frustrationsPainPoints": ["string"],
      "productUsageBehaviors": ["string"],
      "createdAt": "time",
      "updatedAt": "time"
    }
  ]
}
```

### POST /api/v1/insight-flow/get-persona

**Authentication**: Required (JWT)

Request:

```json
{
  "personaId": "uuid",
  "insightResearchId": "uuid"
}
```

Response data:

```json
{
  "persona": {
    "personaId": "string",
    "insightResearchId": "string",
    "description": "string",
    "goals": ["string"],
    "frustrationsPainPoints": ["string"],
    "productUsageBehaviors": ["string"],
    "createdAt": "time",
    "updatedAt": "time"
  }
}
```
