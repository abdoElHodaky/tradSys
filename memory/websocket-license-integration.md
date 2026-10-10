---
name: websocket-license-integration
description: Documented websocket LicenseValidator integration decision
metadata:
  type: reference
---

# WebSocket LicenseValidator Integration Decision

## Background

The websocket package has its own `LicenseValidator` implementation that differs from the licensing service's `Validator`.

## Design Rationale

### websockets.LicenseValidator (Simplified for WebSocket)
- Uses `int`-based `LicenseTier` for efficient comparisons
- In-memory cache only (no database dependencies)
- Optimized for sub-millisecond response times
- Self-contained, no external dependencies required
- Returns `[]string` for features instead of `[]LicenseFeature`

### licensing.LicenseValidator (Full-Featured)
- Uses `string`-based `LicenseTier` for clarity and serialization
- Requires database, cache, rate limiter, and metrics dependencies
- Supports context, timeouts, and comprehensive metrics
- Full validation with feature checking, quota checking, rate limiting

## Integration Decision

The websocket's `LicenseValidator` is **intentionally kept as-is** because:

1. **Performance**: WebSocket connections require sub-millisecond latency. The simplified validator avoids database calls.

2. **Independence**: The WebSocket service can operate without database dependencies, making it more resilient.

3. **Different Use Case**: WebSocket validation is for connection-level access control, not comprehensive licensing management.

## Type Alias Status in websocket/websocket_gateway.go

The websocket package has its own types that cannot be easily aliased to licensing types:
- `LicenseTier int` vs `licensing.LicenseTier string` - incompatible types
- `LicenseValidator` has different method signatures
- `LicenseValidationResult` has different fields (`[]string` vs `[]LicenseFeature`)

## Recommendation

**Do NOT replace websocket's LicenseValidator with licensing.LicenseValidator** unless there's a specific requirement for:
- Database-backed validation
- Rate limiting
- Metrics collection
- Feature-based access control

## Related Files
- `/services/websocket/websocket_components.go` - websocket LicenseValidator implementation
- `/services/websocket/websocket_gateway.go` - WebSocket gateway using LicenseValidator
- `/services/licensing/implementation.go` - licensing.Validator implementation

**Why:** Understanding the different design choices for different use cases helps maintain both implementations correctly.

**How to apply:** When modifying websocket licensing, keep the simplified design unless comprehensive validation is required.