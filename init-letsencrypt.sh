#!/bin/bash

if ! [ -x "$(command -v docker compose)" ]; then
  echo 'Error: docker compose is not installed.' >&2
  exit 1
fi

domains=(familymanageronline.online www.familymanageronline.online)
rsa_key_size=4096
data_path="./certbot"
email="" # Adding a valid address is strongly recommended
staging=0 # Set to 1 if you're testing your setup to avoid hitting request limits

if [ -d "$data_path" ]; then
  read -p "Existing data found for $domains. Continue and replace existing certificate? (y/N) " decision
  if [ "$decision" != "Y" ] && [ "$decision" != "y" ]; then
    exit
  fi
fi


if [ ! -e "$data_path/conf/options-ssl-nginx.conf" ] || [ ! -e "$data_path/conf/ssl-dhparams.pem" ]; then
  echo "### Downloading recommended TLS parameters ..."
  mkdir -p "$data_path/conf"
  curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf > "$data_path/conf/options-ssl-nginx.conf"
  curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot/certbot/ssl-dhparams.pem > "$data_path/conf/ssl-dhparams.pem"
  echo
fi

echo "### Creating dummy certificate for $domains ..."
# Create directories in the certbot volume
docker compose run --rm --entrypoint "\
  mkdir -p /etc/letsencrypt/live/$domains && \
  openssl req -x509 -nodes -newkey rsa:$rsa_key_size -days 1\
    -keyout /etc/letsencrypt/live/$domains/privkey.pem \
    -out /etc/letsencrypt/live/$domains/fullchain.pem \
    -subj '/CN=localhost'" certbot
echo


echo "### Starting nginx with HTTP-only config (for certificate generation) ..."
# Copy no-ssl config temporarily
cp nginx/nginx.conf.no-ssl nginx/nginx.conf
docker compose up --force-recreate -d nginx
echo "Waiting for nginx to be ready..."
sleep 5
echo

echo "### Deleting dummy certificate for $domains ..."
docker compose run --rm --entrypoint "\
  rm -Rf /etc/letsencrypt/live/$domains && \
  rm -Rf /etc/letsencrypt/archive/$domains && \
  rm -Rf /etc/letsencrypt/renewal/$domains.conf" certbot
echo


echo "### Requesting Let's Encrypt certificate for $domains ..."
#Join $domains to -d args
domain_args=""
for domain in "${domains[@]}"; do
  domain_args="$domain_args -d $domain"
done

# Select appropriate email arg
case "$email" in
  "") email_arg="--register-unsafely-without-email" ;;
  *) email_arg="--email $email" ;;
esac

# Enable staging mode if needed
if [ $staging != "0" ]; then staging_arg="--staging"; fi

docker compose run --rm --entrypoint "\
  certbot certonly --webroot -w /var/www/certbot \
    $staging_arg \
    $email_arg \
    $domain_args \
    --rsa-key-size $rsa_key_size \
    --agree-tos \
    --force-renewal" certbot
echo

echo "### Switching to HTTPS config and reloading nginx ..."
# Restore HTTPS config (use template if exists, otherwise keep current)
if [ -f "nginx/nginx.conf.template" ]; then
    cp nginx/nginx.conf.template nginx/nginx.conf
else
    echo "Warning: nginx.conf.template not found. Using current HTTPS config."
fi

# Test and reload nginx
if docker compose exec nginx nginx -t 2>/dev/null; then
    docker compose exec nginx nginx -s reload
    echo
    echo "### Setup complete! Your API is now available at:"
    echo "   https://familymanageronline.online/api/"
else
    echo "Error: nginx config test failed. Check logs: docker compose logs nginx"
    echo "You may need to manually fix the nginx configuration."
fi
echo

