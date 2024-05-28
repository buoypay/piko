# piko

gtxn fork - self hosted ngrok

## Installing on ec2 instance

```bash
git clone git@github.com:buoypay/piko.git

make image

cd piko/docs/demo


# run the certbot only once
docker-compose run --rm certbot certonly --webroot --webroot-path /var/www/certbot/ --dry-run -d piko.bouypay.com


```
