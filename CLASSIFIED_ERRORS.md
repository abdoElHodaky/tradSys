# Classified Critical Errors in TradSys v3

## Classification Legend
- **CRITICAL**: Security vulnerabilities, data loss risks, system compromise
- **HIGH**: Functional failures, missing critical functionality
- **MEDIUM**: Code quality issues, maintainability problems
- **LOW**: Minor issues, code style problems

---

## CRITICAL Security Issues

### 1. Hardcoded Default Credentials (CRITICAL)
**File**: `internal/auth/service.go:90-92, 111-113`
**Severity**: CRITICAL
**Issue**: Default users are created with hardcoded passwords `admin123` and `trader123`.
**Impact**: Anyone with source code access can gain admin access.
**Recommendation**: Use environment variables or secure credential storage, eliminate default users in production.

```go
// Line 90
hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
```

### 2. Sensitive Data Logging (CRITICAL)
**File**: `cmd/tradsys/main.go:167` (throughout codebase)
**Severity**: CRITICAL
**Issue**: Request body is logged which may contain sensitive data (passwords, tokens, PII).
**Impact**: Sensitive data exposure in logs.
**Recommendation**: Sanitize logs before logging, use structured logging with sensitive field masking.

```go
// Line 167 - logging request body
bodyBytes, err := io.ReadAll(c.Request.Body)
```

### 3. CORS Configuration Too Permissive (CRITICAL)
**File**: `internal/gateway/server.go:48-55`
**Severity**: CRITICAL
**Issue**: CORS allows all origins (`*`) and credentials, which can lead to CSRF attacks.
**Impact**: Cross-site request forgery vulnerability.
**Recommendation**: Explicitly define allowed origins, never use wildcard with credentials.

```yaml
# config/tradsys.yaml:176-178
cors:
  allowed_origins: ["*"]
```

---

## CRITICAL Functional Issues

### 4. Order Service Placeholder Implementation (CRITICAL)
**File**: `internal/orders/handler.go:78-95`
**Severity**: CRITICAL
**Issue**: Order creation has no actual implementation - just returns placeholder data.
**Impact**: System cannot process real orders.
**Recommendation**: Implement actual order matching engine integration.

```go
// Line 78-95
// Implementation would go here
// For now, just return a placeholder response
rsp := &orders.OrderResponse{
    Id:            uuid.New().String(),
    // ...
}
return rsp, nil
```

### 5. Risk Validation Stub (CRITICAL)
**File**: `internal/risk/handler.go:43-49`
**Severity**: CRITICAL
**Issue**: Risk validation returns hardcoded `true` with no actual checks.
**Impact**: Trades execute regardless of risk limits.
**Recommendation**: Implement actual risk checks using RiskEngine.

```go
// Line 43-49
// Implementation would go here
// For now, just return a placeholder response
rsp := &risk.ValidateOrderResponse{
    IsValid: true,
}
return rsp, nil
```

### 6. WebSocket Service Not Started (CRITICAL)
**File**: `cmd/tradsys/main.go:349-375`
**Severity**: CRITICAL
**Issue**: WebSocket service `Start()` is called but function doesn't block/wait - program continues and exits.
**Impact**: WebSocket service never runs.
**Recommendation**: Add signal handling and blocking wait.

```go
// Line 372
if err := wsServer.Start(context.Background()); err != nil {
    log.Fatalf("Failed to start WebSocket server: %v", err)
}
// Missing: <-quit (wait for signal)
```

---

## HIGH Severity Issues

### 7. Duplicate Logging Statements (HIGH)
**File**: `internal/ws/handler.go` (multiple locations)
**Severity**: HIGH
**Issue**: Same log statements appear twice, indicating copy-paste errors.
**Impact**: Confusing logs, potential duplicate processing.

### 8. Missing Request Body Redirect in Proxy (HIGH)
**File**: `internal/gateway/proxy.go:114-121`
**Severity**: HIGH
**Issue**: Request body is read but not included in proxy request.
**Impact**: POST/PUT requests lose their body data when proxied.
**Recommendation**: Create request body from read bytes.

```go
// Line 114-119
proxyReq, err := http.NewRequest(c.Request.Method, targetURL.String(), nil)
// Should be: bytes.NewBuffer(bodyBytes) instead of nil
```

### 9. Unused Variable in Risk Engine (HIGH)
**File**: `internal/risk/engine.go:125-127`
**Severity**: HIGH
**Issue**: `re.mu.RLock()` is called but the lock is only released in the defer, and the data is used after unlock.
**Impact**: Potential race condition if data is modified between unlock and use.
**Recommendation**: Move data access inside the lock scope or restructure.

### 10. HTTP Body Closed After Buffering (HIGH)
**File**: `internal/gateway/proxy.go:137-148`
**Severity**: HIGH
**Issue**: Request body is read and converted but original body is not consumed, which can cause issues.
**Impact**: Request body may not be properly forwarded.
**Recommendation**: Use `io.TeeReader` or properly consume and replace.

---

## MEDIUM Severity Issues

### 11. Duplicate WebSocket Log Statements (MEDIUM)
**File**: `internal/architecture/fx/websocket.go:80-88, 120-122`
**Severity**: MEDIUM
**Issue**: Identical log statements in block suggest copy-paste error.

### 12. WebSocket Handler Has No Actual Implementation (MEDIUM)
**File**: `internal/ws/handler.go:97-184`
**Severity**: MEDIUM
**Issue**: WebSocket handlers have placeholder comments and no actual business logic.

### 13. Missing Context Cancellation Handling (MEDIUM)
**File**: `internal/risk/engine.go:104-105, 289`
**Severity**: MEDIUM
**Issue**: Context timeout errors are logged but not properly handled.
**Impact**: Could lead to inconsistent state.

### 14. Rate Limiter Uses Memory Store (MEDIUM)
**File**: `internal/gateway/security.go:30-31`
**Severity**: MEDIUM
**Issue**: Rate limiter uses in-memory store, loses state on restart.
**Recommendation**: Use Redis or persistent store.

### 15. Duplicate Code in WebSocket Handler (MEDIUM)
**File**: `internal/ws/handler.go` and `internal/architecture/fx/websocket.go`
**Severity**: MEDIUM
**Issue**: Similar functionality appears in multiple files.
**Recommendation**: Consolidate WebSocket registration logic.

---

## LOW Severity Issues

### 16. Unused Import in config/manager.go (LOW)
**File**: `internal/config/manager.go`
**Severity**: LOW
**Issue**: `gopkg.in/yaml.v3` import may not be used.

### 17. Inconsistent Naming (LOW)
**File**: Throughout codebase
**Severity**: LOW
**Issue**: Mix of `UserID`, `user_id`, `user-id` conventions.

### 18. Magic Numbers (LOW)
**File**: `internal/gateway/server.go:54`
**Severity**: LOW
**Issue**: CORS max age hardcoded to 12 hours without explanation.

---

## Architecture Issues

### 19. Two-Tier Risk Checking (ARCHITECTURE)
**Issue**: Both `internal/risk/engine.go` and `internal/risk/service.go` have similar code.
**Recommendation**: Consolidate to single risk engine with proper service layer.

### 20. Hardcoded Account ID in Orders (ARCHITECTURE)
**File**: `internal/orders/handler.go:62`
**Severity**: HIGH
**Issue**: Account ID is hardcoded to "default" - should come from auth context.
**Impact**: All orders treated as same account.

---

## Recommendations Summary

1. **Immediate**: Fix risk validation and order processing - completely non-functional
2. **Security**: Remove hardcoded credentials, fix CORS, sanitize logs
3. **Functionality**: Fix proxy body forwarding, add WebSocket signal handling
4. **Architecture**: Consolidate duplicate code, implement proper context handling

---

*Generated: 2026-10-06*
*Classification done for educational/security awareness purposes*