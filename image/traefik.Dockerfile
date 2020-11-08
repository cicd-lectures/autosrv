FROM traefik:2.3

COPY ./config/traefik.yml /traefik.conf.d/
COPY ./config/certificates/ /certificates/
