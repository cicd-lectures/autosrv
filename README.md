# autosrv

This project is a stack to simulate a set of "production" environments based on Docker containers and used for pedagocial and learning purposes.

The autosrv stack serves web pages behind a (configurable) domain referenced as `hostname` in this documentation.

Autosrv is built on the concept of an "environment".
For an environment named `webapp-1`, you have the following elements:

- A webservice available at `https://<hostname>/webapp-1` representing your production web application
- A docker image named `registry.<hostname>/apps/webapp-1` which packages the webservice.
- A docker container named `webapp-1` which runs the webservice,
  based on the latest version of the image `registry.<hostname>/apps/webapp-1`.

When you push a new version of the docker image `registry.<hostname>/webapp-1`,
then the webservice `webapp-1` is updated in place by starting a new container `registry.<hostname>_webapp-1` based on this latest version.

## Get Started

Check that the following requirements are met on the machine where you want to run the autosrv stack:

- An unrestricted Internet access (to download Docker images and go modules)
- You have an available domain `<hostname>` which points to the IP of your Docker engine. The subdomain `registry.<hostname>` is also required to point to the same IP.
  - The 2 domains can be set up through `/etc/hosts` or through DNS.
  - It can be `localhost`: `registry.localhost` and `localhost` should point to the Docker Engine's public IP.
- Docker Engine 20.03+ is installed and running with:
  - Local File Sharing capability
  - Access to the Docker socket `/var/run/docker`
  - The domain `registry.<hostname>` is in the list of the insecure registries
    (ref. <https://docs.docker.com/engine/reference/commandline/dockerd/#insecure-registries>)
- The command line `docker-compose` 1.27+ on your PATH

With the requirements checked, clone this repository and start the stack:

```shell
git clone https://github.com/jlevesy/autosrv
cd ./autosrv
docker-compose up -d
```

You can now pull the Docker image of a webservice and push it to the local registry:

```shell
docker pull containous/whoami:v1.4.0
docker tag containous/whoami:v1.4.0 registry.localhost/foobiz/whoami:latest
docker push registry.localhost/foobiz/whoami:latest
```

You can now access the web application at <http://registry.localhost/apps/foobiz/whoami>.
A new container named `foobiz_whoami` should be running.

```shell
$ curl http://registry.localhost/apps/foobiz/whoami
Hostname: 78ce36854ca8
IP: 127.0.0.1
IP: 172.18.0.5
RemoteAddr: 172.18.0.2:51472
GET /apps/foobiz/whoami HTTP/1.1
Host: registry.localhost
User-Agent: curl/7.64.1
Accept: */*
Accept-Encoding: gzip
X-Forwarded-For: 172.18.0.1
X-Forwarded-Host: registry.localhost
X-Forwarded-Port: 80
X-Forwarded-Proto: http
X-Forwarded-Server: de8ef26d1c1c
X-Real-Ip: 172.18.0.1

$ docker ps | grep whoami
78ce36854ca8   registry.localhost/foobiz/whoami:latest   "/whoami"                6 minutes ago    Up 6 minutes    80/tcp               foobiz_whoami
```
## Components

The stack contains the following elements:

- `ingress`: this is a reverse proxy and the entrypoint for all requests.
  This service must be published to allow access to the stack.

- `deployer`: this is an application written in Golang, responsible to deploy the container based on the `registry` push events.

- `registry`: this is a Docker registry which hosts the Docker images for the environments and sends events to `deployer` through webhooks.
