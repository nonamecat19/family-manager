# Firewall Setup for HTTPS

The Let's Encrypt certificate generation requires port 80 to be accessible from the internet. The error you're seeing indicates that Let's Encrypt cannot reach your server on port 80.

## Check Current Firewall Status

### Ubuntu/Debian (UFW)
```bash
# Check status
sudo ufw status

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Verify
sudo ufw status
```

### CentOS/RHEL (firewalld)
```bash
# Check status
sudo firewall-cmd --list-all

# Allow HTTP and HTTPS
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload

# Verify
sudo firewall-cmd --list-all
```

### iptables
```bash
# Allow HTTP
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Save rules (Ubuntu/Debian)
sudo iptables-save | sudo tee /etc/iptables/rules.v4
```

## Cloud Provider Firewall

If you're using a cloud provider (DigitalOcean, AWS, GCP, Azure), you also need to configure their firewall/security groups:

### DigitalOcean
1. Go to your Droplet → Networking → Firewalls
2. Create/Edit firewall rules
3. Add inbound rules:
   - HTTP (port 80)
   - HTTPS (port 443)
4. Apply to your droplet

### AWS
1. Go to EC2 → Security Groups
2. Edit inbound rules
3. Add:
   - Type: HTTP, Port: 80, Source: 0.0.0.0/0
   - Type: HTTPS, Port: 443, Source: 0.0.0.0/0

### GCP
1. Go to VPC Network → Firewall rules
2. Create rule:
   - Allow tcp:80,443
   - Target: All instances
   - Source: 0.0.0.0/0

## Verify Port Accessibility

Test if port 80 is accessible from outside:
```bash
# From another machine or use an online tool
curl -I http://familymanageronline.online

# Or test locally
curl -I http://localhost
```

## Temporary Workaround

If you can't open port 80 immediately, you can:

1. Use the HTTP-only config temporarily:
   ```bash
   cp nginx/nginx.conf.no-ssl nginx/nginx.conf
   docker compose restart nginx
   ```

2. Your API will be available at `http://familymanageronline.online/api/` (without HTTPS)

3. Once port 80 is open, run the SSL setup script again:
   ```bash
   ./init-letsencrypt.sh
   ```

## After Opening Ports

1. Verify ports are open:
   ```bash
   sudo netstat -tlnp | grep -E ':(80|443)'
   ```

2. Test from outside:
   ```bash
   curl http://familymanageronline.online/.well-known/acme-challenge/test
   ```

3. Run the SSL setup:
   ```bash
   ./init-letsencrypt.sh
   ```


