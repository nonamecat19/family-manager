# HTTPS Setup Guide

This guide explains how to set up HTTPS for `familymanageronline.online` using Let's Encrypt.

## Prerequisites

1. Your domain `familymanageronline.online` must point to your server's IP address
2. Ports 80 and 443 must be open in your firewall
3. Docker and Docker Compose must be installed

## Initial SSL Certificate Setup

1. **Start the services** (nginx will start with a temporary self-signed certificate):
   ```bash
   docker compose up -d
   ```

2. **Run the SSL initialization script**:
   ```bash
   ./init-letsencrypt.sh
   ```

   This script will:
   - Create dummy certificates so nginx can start
   - Request real certificates from Let's Encrypt
   - Reload nginx with the real certificates

3. **Verify the setup**:
   - Visit `https://familymanageronline.online/api/health` in your browser
   - You should see a valid SSL certificate

## Certificate Renewal

Certificates are automatically renewed by the `certbot` service in docker-compose. It runs every 12 hours and renews certificates when they're within 30 days of expiration.

Nginx is configured to reload every 6 hours to pick up renewed certificates.

## Configuration

### Ports

- **HTTP (Port 80)**: Redirects to HTTPS
- **HTTPS (Port 443)**: Main API endpoint

You can customize ports using environment variables:
- `NGINX_HTTP_PORT` (default: 80)
- `NGINX_HTTPS_PORT` (default: 443)

### SSL Configuration

The SSL configuration uses:
- TLS 1.2 and 1.3 only
- Strong cipher suites
- HSTS (HTTP Strict Transport Security)
- OCSP stapling

## Troubleshooting

### Certificate generation fails

1. Ensure your domain DNS is properly configured
2. Check that ports 80 and 443 are accessible
3. Verify the domain is reachable: `curl http://familymanageronline.online`

### Nginx won't start

1. Check nginx logs: `docker compose logs nginx`
2. Verify the certificate files exist:
   ```bash
   docker compose exec nginx ls -la /etc/letsencrypt/live/familymanageronline.online/
   ```

### Certificate renewal issues

1. Check certbot logs: `docker compose logs certbot`
2. Manually renew: `docker compose run --rm certbot renew`

## Testing with Staging

To test without hitting Let's Encrypt rate limits, edit `init-letsencrypt.sh` and set:
```bash
staging=1
```

Then use the staging certificates for testing. When ready, set `staging=0` and run the script again to get production certificates.


