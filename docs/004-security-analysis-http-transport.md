# Security Analysis: HTTP Transport Authentication

**Date:** 2026-01-31
**Document ID:** 004
**Subject:** Security analysis of multi-user authentication approaches for OData MCP Bridge
**Related:** PR #24 (Header Forwarding), Issue #12 (SAP OData Integration)
**Status:** ✅ IMPLEMENTED (Phase 1: Secure-by-Default Hardening)

---

## Executive Summary

This document analyzes authentication approaches for the OData MCP Bridge HTTP transport. After evaluating multiple solutions, we recommend a **secure-by-default hardening approach** that protects against network-level attacks while acknowledging that host-level compromise is out of scope for application-level fixes.

---

## Issue Analysis

### PR #24: Header Forwarding

**What it proposes:** Forward HTTP headers (including `Authorization`, `Cookie`) from MCP client requests to the OData service.

**Assessment: Architectural Shortcut**

This approach turns the MCP server into a credential-forwarding proxy rather than an authenticated service boundary.

| Proper Architecture | PR #24 Approach |
|---------------------|-----------------|
| MCP server authenticates users | No MCP-level auth |
| Credential vault/mapping | Blind header passthrough |
| Audit trail per user | No visibility into who's doing what |
| Per-user rate limiting | None |
| Revoke specific user access | Can't without changing OData service |

**Security Concerns:**
1. MCP server becomes credential aggregation point
2. No validation of forwarded headers
3. Over-sharing (browser cookies may exceed intent)
4. Audit gap
5. Trust boundary violation

**Recommendation:** Do not merge as-is. If merged, mark as experimental with strong warnings.

---

## Multi-User Authentication Options Evaluated

### Option 1: Per-Instance Model (vsp approach)

```
User A → their own odata-mcp instance (their creds) → OData
User B → their own odata-mcp instance (their creds) → OData
```

| Aspect | Assessment |
|--------|------------|
| Code changes | Zero |
| Security model | Clean - one instance = one identity |
| Trade-off | Each user needs own server process |

**Verdict:** Architecturally proper. No credential translation.

### Option 2: API Key → Credential Mapping

```yaml
# credentials.yaml
users:
  - api_key: "dev-alice-2024"
    odata:
      username: "ALICE"
      password: "encrypted:xxxxx"
```

| Aspect | Assessment |
|--------|------------|
| Code changes | ~200-300 LOC |
| Security model | Pragmatic workaround |
| Issues | Managing two sets of secrets, manual sync |

**Verdict:** Not proper. Still credential translation.

### Option 3: JWT/Bearer Token with Claims

| Aspect | Assessment |
|--------|------------|
| Code changes | ~400-500 LOC |
| Security model | Better - cryptographic identity |
| Issues | Still need credential mapping on server side |

**Verdict:** Better but still not true federation.

### Option 4: OAuth Token Exchange / Identity Federation

```
User → IdP (Azure AD, Okta, SAP IAS) → Token
MCP Server → Passes Token → OData validates with same IdP
```

| Aspect | Assessment |
|--------|------------|
| Code changes | ~800+ LOC + infrastructure |
| Security model | Proper - true identity federation |
| Issues | Requires SAP Basis config, corporate IdP |

**Verdict:** The only truly proper solution. Requires infrastructure investment.

---

## Recommended Solution: Secure-by-Default Hardening

### Rationale

Rather than implementing complex multi-user auth, we recommend:

1. **Per-instance model** (one instance = one identity)
2. **Secure-by-default** guards to prevent accidental exposure
3. **Documentation** for enterprise deployment patterns

### Current Vulnerabilities

```bash
# Currently possible - zero warnings
odata-mcp --transport streamable-http --http-addr 0.0.0.0:3000 \
          --user SAP_ADMIN --password secret
# Result: Credentials exposed to entire network
```

### Proposed Hardening (~100-150 LOC)

```go
func validateHTTPSecurity(cfg *HTTPConfig) error {
    host, _, err := net.SplitHostPort(cfg.Addr)
    if err != nil {
        return err
    }

    ip := net.ParseIP(host)
    isLoopback := ip != nil && ip.IsLoopback()

    // Reject 0.0.0.0/:: without explicit flag
    if ip != nil && ip.IsUnspecified() {
        if !cfg.ExplicitAllowAllInterfaces {
            return fmt.Errorf(
                "binding to all interfaces (0.0.0.0/::) is dangerous\n"+
                "Use specific interface IP or --allow-all-interfaces")
        }
    }

    // Require TLS + token for non-localhost
    if !isLoopback {
        if cfg.Token == "" {
            return fmt.Errorf("--mcp-token required for non-localhost")
        }
        if !cfg.TLS.Enabled {
            return fmt.Errorf("--tls required for non-localhost")
        }
    }

    return nil
}

// Constant-time token comparison (prevent timing attacks)
func validateToken(provided, expected string) bool {
    return subtle.ConstantTimeCompare(
        []byte(provided),
        []byte(expected),
    ) == 1
}
```

### Security Matrix After Hardening

| Binding | Token | TLS | Allowed? |
|---------|-------|-----|----------|
| 127.0.0.1:3000 | - | - | Yes |
| 192.168.1.x:3000 | Yes | Yes | Yes |
| 192.168.1.x:3000 | Yes | No | No - TLS required |
| 192.168.1.x:3000 | No | - | No - token required |
| 0.0.0.0:3000 | - | - | No - explicit flag required |

### Additional Hardening Measures

1. **Token from file, not CLI** (avoid `ps` exposure)
   ```bash
   --mcp-token-file /run/secrets/mcp-token
   ```

2. **Proper address parsing** (handle IPv6, shorthand)
   ```go
   // Handle: 127.0.0.2, 127.1, [::1], ::, etc.
   ```

3. **Rate limiting on token attempts** (prevent brute force)

---

## Threat Model Analysis

### What Hardening Protects Against

| Threat | Current | With Hardening |
|--------|---------|----------------|
| Accidental network exposure | Vulnerable | **Protected** |
| Network attacker (no host access) | Vulnerable | **Protected** |
| SSRF from same host | Vulnerable | Vulnerable |
| Host compromise | Vulnerable | Vulnerable |
| Container sidecar attack | Vulnerable | Vulnerable |

### Why Host Compromise is Out of Scope

**Security Boundary Principle:** You cannot defend against threats operating at a higher privilege level than your code.

```
┌─────────────────────────────────────────────────────────────┐
│ PHYSICAL ACCESS                                              │
│  └─► App-level fix? Impossible.                             │
├─────────────────────────────────────────────────────────────┤
│ HYPERVISOR / CLOUD PROVIDER                                  │
│  └─► App-level fix? Impossible.                             │
├─────────────────────────────────────────────────────────────┤
│ HOST OS (root/admin)                                         │
│  └─► App-level fix? Impossible.                             │
├─────────────────────────────────────────────────────────────┤
│ HOST OS (same user)                                          │
│  └─► App-level fix? Impossible.                             │
├─────────────────────────────────────────────────────────────┤
│ NETWORK ACCESS            ◄── Hardening protects HERE       │
│  └─► App-level fix? YES                                     │
└─────────────────────────────────────────────────────────────┘
```

If an attacker has host access, they can:
```bash
# Read credentials directly
cat /proc/$(pgrep odata-mcp)/environ | tr '\0' '\n' | grep PASSWORD
cat /proc/$(pgrep odata-mcp)/cmdline

# Attach debugger
gdb -p $(pgrep odata-mcp)

# Replace binary
cp malicious /usr/bin/odata-mcp
```

**No application code can prevent this.** Host security is the responsibility of:
- OS hardening
- Access controls
- Container isolation
- Infrastructure security

---

## Implementation Recommendation

### Phase 1: Secure by Default (~100-150 LOC)

1. Localhost-only binding by default
2. Require `--mcp-token` + `--tls` for non-localhost
3. Block `0.0.0.0`/`::` without explicit flag
4. Token from file option (avoid CLI exposure)
5. Constant-time token comparison

### Phase 2: Documentation

1. Secure deployment guide
2. Enterprise patterns (API gateway, OAuth proxy)
3. Container/Kubernetes deployment examples
4. Threat model documentation

### What NOT to Implement

1. Complex multi-user auth system
2. Credential vault integration
3. OAuth/SAML at application level
4. Any "fix" for host-level compromise

---

## Comparison: vsp vs odata_mcp_go

| Aspect | vsp (vibing-steampunk) | odata_mcp_go |
|--------|------------------------|--------------|
| MCP Transport | stdio only | stdio, SSE, HTTP |
| Security risk | None (process-local) | HTTP endpoints exposed |
| Needs hardening? | No | Yes |
| Multi-user model | Per-instance | Per-instance (recommended) |

vsp's stdio-only approach is inherently secure. odata_mcp_go's HTTP transports require the hardening described in this document.

---

## Conclusion

1. **PR #24 (header forwarding)** is a security shortcut - not recommended
2. **True proper auth** (identity federation) requires infrastructure, not code
3. **Pragmatic proper solution** is per-instance + secure-by-default hardening
4. **~100-150 LOC** closes network attack surface
5. **Host compromise** is out of scope - different security boundary

The recommended approach protects against the most common and likely threats (accidental exposure, network attackers) while being honest about what application-level code cannot fix (host compromise).

---

## Related Documents

- **Document 005:** Issue #12 - SAP OData Multi-Schema Parser Bug (separate document)
- **PR #24:** Header forwarding (security concerns documented above)
