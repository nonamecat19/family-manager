# Quick Fix for SSL Certificate Issues

## Current Problem

The Let's Encrypt certificate generation is failing because port 80 is not accessible from the internet (firewall issue).

## Immediate Solution

### Step 1: Open Port 80 in Firewall

**For Ubuntu/Debian (most common):**
```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw status
```

**For DigitalOcean/Cloud Providers:**
- Also check your cloud provider's firewall/security groups
- Ensure port 80 is open in the cloud console

### Step 2: Use HTTP-Only Config Temporarily

While you fix the firewall, use HTTP-only config:

```bash
# Stop current nginx
docker compose stop nginx

# Use HTTP-only config
cp nginx/nginx.conf.no-ssl nginx/nginx.conf

# Restart nginx
docker compose up -d nginx
```

Your API will work at: `http://familymanageronline.online/api/`

### Step 3: After Opening Port 80

1. **Verify port 80 is accessible:**
   ```bash
   # From another machine or online tool
   curl -I http://familymanageronline.online
   ```

2. **Run SSL setup again:**
   ```bash
   ./init-letsencrypt.sh
   ```

## Alternative: Manual Certificate Setup

If automatic setup continues to fail:

1. **Use staging mode first (for testing):**
   Edit `init-letsencrypt.sh` and set:
   ```bash
   staging=1
   ```

2. **Or use DNS challenge instead of HTTP:**
   ```bash
   docker compose run --rm certbot certonly \
     --manual --preferred-challenges dns \
     -d familymanageronline.online \
     -d www.familymanageronline.online
   ```

## Check Current Status

```bash
# Check if nginx is running
docker compose ps

# Check nginx logs
docker compose logs nginx

# Check certbot logs
docker compose logs certbot

# Test API endpoint
curl http://familymanageronline.online/api/health
```


