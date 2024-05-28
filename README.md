# piko

gtxn fork - self hosted ngrok

## Installing on ec2 instance

```bash
git clone git@github.com:buoypay/piko.git

make image

cd piko/docs/demo


# set up certs for the first time
docker-compose up -d load-balancer

# run the certbot only once
docker-compose run --rm certbot certonly --webroot --webroot-path /var/www/certbot/ -d piko.bouypay.com -d lewis.piko.bouypay.com

# check the certs got created
ls ./certbot/www





```

## Renew the certs (eventually make a cron job...)

```bash

```
