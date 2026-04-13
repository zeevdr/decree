# Multi-Tenant Auth

**Status:** Planning
**Started:** 2026-04-13

---

## Problem

The current auth model assigns exactly one `tenant_id` per user/admin (via JWT claims or `x-tenant-id` header). This means:

- A user can only access one tenant's config
- An admin can only manage one tenant
- There's no way for a user/admin to switch between tenants without getting a new JWT
- The `GET /v1/tenants` endpoint requires superadmin — non-superadmin can't list even their own tenants

## Proposed Changes

### 1. Support multiple tenant IDs in JWT claims

```json
{
  "sub": "alice",
  "role": "admin",
  "tenant_ids": ["tenant-a-uuid", "tenant-b-uuid"]
}
```

Backward compatible: `tenant_id` (singular) still works, `tenant_ids` (array) is the new field.

### 2. Allow non-superadmin to list their own tenants

`GET /v1/tenants` should return only the tenants the caller has access to, based on their claims:
- superadmin: all tenants
- admin/user with `tenant_ids`: only those tenants
- admin/user with `tenant_id`: only that one tenant

### 3. Per-request tenant scoping

Non-superadmin requests still need to specify which tenant they're acting on (via `x-tenant-id` header or path parameter). The server validates it's in their allowed list.

### 4. Metadata auth (dev mode)

`x-tenant-id` header continues to work as before. For multi-tenant, the dev client can make separate requests per tenant.

## Impact

- Server: auth interceptor, JWT claims parsing, tenant list filtering
- Admin GUI: tenant selector for non-superadmin (currently text input, would become dropdown)
- SDKs: no changes needed (already pass tenant_id per call)

## Key Decisions

1. **Additive** — existing single `tenant_id` claim still works
2. **Server-side filtering** — non-superadmin GET /v1/tenants returns only accessible tenants
3. **No client-side tenant list fetching hacks** — the server handles authorization
