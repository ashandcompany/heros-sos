# syntax=docker/dockerfile:1

# Image de `./make` (voir le script du même nom à la racine) : GNU make, bash et openssl à côté du client Docker et
# de Compose, pour lancer les cibles du Makefile sans autre installation que Docker. Rien ne tourne « dans » cette
# image : les cibles pilotent le Docker de la machine, dont ./make monte le socket.
#
# Client Docker épinglé, comme les autres images du dépôt : un tag flottant ferait évoluer Compose d'une machine à
# l'autre. Il négocie la version de l'API avec le démon, donc un démon un peu plus ancien ou plus récent convient.
ARG DOCKER_CLI_IMAGE=docker:29.9.2-cli

FROM ${DOCKER_CLI_IMAGE}

# bash : le SHELL des Makefile. openssl : `make env` et `make secrets` y génèrent les secrets. setpriv : celui de
# BusyBox ne sait pas changer d'utilisateur, or l'entrypoint en a besoin.
RUN apk add --no-cache bash make openssl setpriv

COPY --chmod=755 entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
