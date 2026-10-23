#!/bin/sh
# Lance `make` dans un conteneur : pour travailler sur le projet sans installer make.
#
#   ./make up             équivaut à   make up
#   ./make seed RESET=1   équivaut à   make seed RESET=1
#   ./make                liste les cibles (make help)
#
# Seul Docker est nécessaire. Au premier appel, le script construit une image de 250 Mo (docker/make/Dockerfile :
# make, bash, openssl, client Docker et Compose), puis lance make dedans. Les cibles pilotent le Docker de la
# machine, pas un Docker imbriqué : le socket est monté, et le dépôt l'est au MÊME chemin que sur la machine. C'est ce
# qui permet à Compose, qui tourne dans le conteneur, de donner au démon des chemins qu'il comprend (./frontend:/app…).
# make tourne avec l'uid et le gid de l'utilisateur (docker/make/entrypoint.sh) : .env et secrets/ ne sont pas à root.
#
# Limites :
#   - les cibles qui lancent Go sur la machine (perf-prepare, seed-perf, perf-run, perf, et celles de
#     backend/Makefile comme run, test ou lint) exigent Go installé et ne passent pas par ce script ;
#   - le contrôle des ports déjà pris de `make up` (lsof) est ignoré : Docker signale lui-même un port occupé.
set -eu

# Git Bash (Windows) réécrirait sinon les chemins passés à docker ; sans effet ailleurs.
export MSYS_NO_PATHCONV=1

die() {
  echo "$*" >&2
  exit 1
}

root=$(cd "$(dirname "$0")" && pwd -P)
image_dir=$root/docker/make

command -v docker >/dev/null 2>&1 || die "docker est introuvable : installez Docker Desktop (ou Docker Engine), puis recommencez."
docker info >/dev/null 2>&1 || die "Docker ne répond pas : lancez Docker Desktop (ou le démon) puis recommencez."

# Le socket du démon : /var/run/docker.sock, ou celui que désigne DOCKER_HOST, ou celui du contexte Docker actif
# (Docker Desktop sans le socket par défaut, Colima…). Un démon distant (tcp://, ssh://) n'est pas pris en charge :
# les fichiers du dépôt ne s'y trouvent pas.
sock=/var/run/docker.sock
case ${DOCKER_HOST:-} in
  '')
    if [ ! -S "$sock" ]; then
      endpoint=$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)
      case $endpoint in unix://*) sock=${endpoint#unix://} ;; esac
    fi
    ;;
  unix://*) sock=${DOCKER_HOST#unix://} ;;
  *) die "DOCKER_HOST=$DOCKER_HOST : ./make ne sait utiliser qu'un démon Docker local (socket Unix)." ;;
esac
[ -S "$sock" ] || die "Socket Docker introuvable : $sock. S'il est ailleurs : DOCKER_HOST=unix:///chemin/du/socket ./make …"

# L'étiquette de l'image suit le contenu de son dossier : modifier le Dockerfile ou l'entrypoint la reconstruit.
tag=$(cat "$image_dir/Dockerfile" "$image_dir/entrypoint.sh" | cksum | cut -d' ' -f1)
image=heros-sos/make:$tag
if ! docker image inspect "$image" >/dev/null 2>&1; then
  echo "Première utilisation : construction de l'image make (une seule fois)…" >&2
  docker build --tag "$image" "$image_dir" >&2
fi

# Comme make natif, il travaille dans le dossier courant (../make depuis backend/ lance backend/Makefile).
workdir=$(pwd -P)
case $workdir in "$root" | "$root"/*) ;; *) workdir=$root ;; esac

# Terminal : make moderator et make reset posent une question. -t seulement si l'entrée ET la sortie en sont un.
if [ -t 0 ] && [ -t 1 ]; then
  tty_flags=-it
else
  tty_flags=-i
fi

# Make lit son environnement : `FORCE=1 ./make up` doit marcher comme `FORCE=1 make up`, donc ces variables (et
# seulement elles, pas tout l'environnement de la machine) sont transmises. -e NOM sans valeur : une valeur secrète
# (LOADTEST_PASSWORD) ne figure pas dans la liste des processus. À compléter si un Makefile lit une nouvelle variable.
env_flags=
for name in $(env | sed -n 's/^\([A-Z][A-Z0-9_]*\)=.*/\1/p'); do
  case $name in
    FORCE | V | RESET | EMAIL | FIRST | LAST | BIRTH | LOADTEST_PASSWORD | SCENARIOS | DISK_MIN_MB | DB_* | ELK_* | PERF_*)
      env_flags="$env_flags -e $name"
      ;;
  esac
done

# shellcheck disable=SC2086 # tty_flags et env_flags sont des listes d'options, à découper
exec docker run --rm --init $tty_flags \
  -v "$sock:/var/run/docker.sock" \
  -v "$root:$root" \
  -w "$workdir" \
  -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
  $env_flags \
  "$image" "$@"
