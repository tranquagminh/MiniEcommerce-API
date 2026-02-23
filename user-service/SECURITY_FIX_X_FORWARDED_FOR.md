# Security Fix: X-Forwarded-For IP Spoofing Prevention

## Overview

This document describes the security fix implemented to prevent IP spoofing via the `X-Forwarded-For` HTTP header in the rate limiting middleware.

## The Vulnerability

### Before the Fix

The previous implementation blindly trusted the `X-Forwarded-For` header:

```go
// VULNERABLE CODE (before fix)
xff := r.Header.Get("X-Forwarded-For")
if xff != "" {
    ips := strings.Split(xff, ",")
    return strings.TrimSpace(ips[0])  // ⚠️ Trusts any client-provided IP
}
```

**Attack Scenario:**
An attacker could bypass rate limiting by setting a fake `X-Forwarded-For` header:

```bash
# Attacker sends multiple requests with different fake IPs
curl -H "X-Forwarded-For: 1.2.3.4" https://api.example.com/login
curl -H "X-Forwarded-For: 5.6.7.8" https://api.example.com/login
curl -H "X-Forwarded-For: 9.10.11.12" https://api.example.com/login
# Each request appears to come from a different IP → Rate limit bypassed! ⚠️
```

## The Fix

### Secure Implementation

The new implementation only trusts `X-Forwarded-For` when:
1. Explicitly configured to trust proxies (`TRUST_PROXIES=true`)
2. The direct connection comes from a trusted proxy IP range

```go
// SECURE CODE (after fix)
func (rl *RateLimiter) getClientIP(r *http.Request) string {
    // Get the direct connection IP
    remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
    directIP := net.ParseIP(remoteIP)

    // Only trust X-Forwarded-For if:
    // 1. Explicitly configured to trust proxies
    // 2. Request comes from a trusted proxy IP
    if rl.trustProxies && rl.isTrustedProxy(directIP) {
        xff := r.Header.Get("X-Forwarded-For")
        if xff != "" {
            clientIP := strings.TrimSpace(strings.Split(xff, ",")[0])
            if net.ParseIP(clientIP) != nil {
                return clientIP  // ✅ Safe to use
            }
        }
    }

    // Default: use direct connection IP
    return remoteIP  // ✅ Cannot be spoofed
}
```

## Configuration

### Environment Variables

Add these to your `.env` file:

```bash
# Trusted Proxy Configuration
TRUST_PROXIES=false  # Set to true only when behind a trusted reverse proxy
TRUSTED_PROXY_CIDRS=  # Comma-separated list of trusted proxy IPs in CIDR notation
```

### Production Examples

#### Behind Nginx on Same Server
```bash
TRUST_PROXIES=true
TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128
```

#### Behind Cloud Load Balancer (AWS ALB Example)
```bash
TRUST_PROXIES=true
# AWS ALB private IP ranges
TRUSTED_PROXY_CIDRS=10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
```

#### Behind Cloudflare
```bash
TRUST_PROXIES=true
# Cloudflare IP ranges (update periodically from https://www.cloudflare.com/ips/)
TRUSTED_PROXY_CIDRS=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22,103.31.4.0/22,141.101.64.0/18,108.162.192.0/18,190.93.240.0/20,188.114.96.0/20,197.234.240.0/22,198.41.128.0/17,162.158.0.0/15,104.16.0.0/13,104.24.0.0/14,172.64.0.0/13,131.0.72.0/22
```

#### Multiple Proxy Layers
```bash
TRUST_PROXIES=true
# Trust both local nginx and upstream load balancer
TRUSTED_PROXY_CIDRS=127.0.0.1/32,10.0.0.0/8
```

### Development (No Proxy)
```bash
# Default - DO NOT trust X-Forwarded-For in development
TRUST_PROXIES=false
TRUSTED_PROXY_CIDRS=
```

## Security Best Practices

### ✅ DO:
- **Only enable `TRUST_PROXIES=true` in production** when behind a known reverse proxy
- **Use specific CIDR ranges** - don't use `0.0.0.0/0` (would trust all IPs)
- **Keep proxy IP ranges updated** if using cloud providers
- **Use the most restrictive CIDR possible** (e.g., `/32` for single IP)
- **Validate configuration on deployment** - misconfiguration = security risk

### ❌ DON'T:
- **Don't enable proxy trust in development** unless testing proxy behavior
- **Don't trust public IP ranges** (only trust your proxy infrastructure)
- **Don't use `0.0.0.0/0`** - this defeats the entire security fix
- **Don't forget to update** cloud provider IP ranges (they change)

## Testing the Fix

### Test 1: Without Proxy Trust (Default - Secure)
```bash
# Set in .env
TRUST_PROXIES=false

# Try to spoof IP
curl -H "X-Forwarded-For: 1.2.3.4" http://localhost:8081/api/login

# Expected: X-Forwarded-For is IGNORED, uses actual connection IP ✅
```

### Test 2: With Proxy Trust from Untrusted IP
```bash
# Set in .env
TRUST_PROXIES=true
TRUSTED_PROXY_CIDRS=10.0.0.0/8

# Make request from non-trusted IP (e.g., 192.168.1.100)
curl -H "X-Forwarded-For: 1.2.3.4" http://localhost:8081/api/login

# Expected: X-Forwarded-For is IGNORED, uses actual connection IP ✅
```

### Test 3: With Proxy Trust from Trusted IP
```bash
# Set in .env
TRUST_PROXIES=true
TRUSTED_PROXY_CIDRS=127.0.0.1/32

# Make request from trusted IP (127.0.0.1)
curl -H "X-Forwarded-For: 1.2.3.4" http://localhost:8081/api/login

# Expected: Uses 1.2.3.4 from X-Forwarded-For ✅
```

## Impact

### Files Modified
1. `/internal/config/config.go` - Added proxy configuration
2. `/internal/interfaces/http/middleware/ratelimit.go` - Secure IP extraction
3. `/internal/interfaces/http/middleware/redis_ratelimit.go` - Secure IP extraction
4. `.env` - Configuration examples

### Backwards Compatibility
✅ **Fully backwards compatible** - defaults to secure behavior (`TRUST_PROXIES=false`)

### Performance Impact
✅ **Negligible** - Only adds IP validation when proxy trust is enabled

## Migration Guide

### For Existing Deployments

1. **Direct Internet Exposure** (No Proxy):
   ```bash
   # No changes needed - already secure by default
   TRUST_PROXIES=false
   ```

2. **Behind Reverse Proxy**:
   ```bash
   # Add these to .env
   TRUST_PROXIES=true
   TRUSTED_PROXY_CIDRS=<your-proxy-ips>  # Get from your proxy config
   ```

3. **Verify** rate limiting still works correctly after deployment

## References

- [OWASP - IP Spoofing](https://owasp.org/www-community/attacks/Spoofing_Attack)
- [RFC 7239 - Forwarded HTTP Extension](https://tools.ietf.org/html/rfc7239)
- [Cloudflare IP Ranges](https://www.cloudflare.com/ips/)
- [AWS IP Ranges](https://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html)

## Questions?

For questions or issues with this security fix, please review:
1. Is `TRUST_PROXIES` set correctly for your deployment?
2. Are your proxy IP ranges in CIDR notation correct?
3. Are you using the latest proxy IP ranges from your provider?
