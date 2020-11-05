# autosrv

This project is a stack to simulate a set of "production" environments based on Docker containers and used for pedagocial and learning purposes.

The autosrv stack serves web pages behind a (configurable) domain referenced as `hostname` in this documentation.

Autosrv is built on the concept of an "environment".
For an environment named `webapp-1`, you have the following elements:

- A webservice available at `https://<hostname>/webapp-1` representing your production web application
- A docker image named `<hostname>/apps/webapp-1` which packages the webservice.
- A docker container named `webapp-1` which runs the webservice,
  based on the latest version of the image `<hostname>/apps/webapp-1`.

When you push a new version of the docker image `<hostname>/webapp-1`,
then the webservice `webapp-1` is updated in place by starting a new container `<hostname>_webapp-1` based on this latest version.

## Get Started

Check that the following requirements are met on the machine where you want to run the autosrv stack:

- An unrestricted Internet access (to download Docker images and go modules)
- By default, the autosrv stack is published to `localhost`. If you plan to use another domain `<hostname>`:
  - Ensure that the domain `<hostname>` points to the public IP of your Docker Engine through `/etc/hosts` or through DNS
  - Use the environment variable `AUTOSRV_HOSTNAME` (either export it on your shell, or change the default value in the file `.env`)
    to specify the new domain `<hostname>` instead of `localhost`.
  - Replace any occurence of `localhost` by `<hostname>` on the new instructions.
- Docker Engine 20.03+ is installed and running with:
  - Local File Sharing capability
  - Access to the Docker socket `/var/run/docker`
  - The domain `<hostname>` is in the list of the insecure registries
    (ref. <https://docs.docker.com/engine/reference/commandline/dockerd/#insecure-registries>)
- The command line `docker-compose` 1.27+ on your PATH

With the requirements checked, clone this repository, check the hostname and start the stack:

```shell
git clone https://github.com/jlevesy/autosrv
cd ./autosrv
docker-compose up -d
```

You can now pull the Docker image of a webservice and push it to the local registry:

```shell
docker pull containous/whoami:v1.4.0
docker tag containous/whoami:v1.4.0 localhost/foobiz/whoami:latest
docker push localhost/foobiz/whoami:latest
```

You can now access the web application at <http://localhost/apps/foobiz/whoami>.
A new container named `foobiz_whoami` should be running.

```shell
$ curl http://localhost/apps/foobiz/whoami
Hostname: 78ce36854ca8
IP: 127.0.0.1
IP: 172.18.0.5
RemoteAddr: 172.18.0.2:51472
GET /apps/foobiz/whoami HTTP/1.1
Host: localhost
User-Agent: curl/7.64.1
Accept: */*
Accept-Encoding: gzip
X-Forwarded-For: 172.18.0.1
X-Forwarded-Host: localhost
X-Forwarded-Port: 80
X-Forwarded-Proto: http
X-Forwarded-Server: de8ef26d1c1c
X-Real-Ip: 172.18.0.1

$ docker ps | grep whoami
78ce36854ca8   localhost/foobiz/whoami:latest   "/whoami"                6 minutes ago    Up 6 minutes    80/tcp               foobiz_whoami
```

## Components

The stack contains the following elements:

- `ingress`: this is a reverse proxy and the entrypoint for all requests.
  This service must be published to allow access to the stack.

- `deployer`: this is an application written in Golang, responsible to deploy the container based on the `registry` push events.

- `registry`: this is a Docker registry which hosts the Docker images for the environments and sends events to `deployer` through webhooks.
