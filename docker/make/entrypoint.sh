#!/bin/sh
# Lance make avec l'identité de l'utilisateur de la machine (HOST_UID, HOST_GID, transmis par ./make).
#
# Le conteneur démarre en root, le temps de redescendre d'un cran, pour deux raisons :
#   - les fichiers que make crée dans le dépôt (backend/.env, backend/secrets/…) doivent appartenir à
#     l'utilisateur, pas à root : il ne pourrait plus les lire ni les modifier sous Linux ;
#   - le socket Docker n'est lisible que par son groupe (root:docker sous Linux, root:root sous Docker Desktop) :
#     l'utilisateur y est ajouté à l'exécution, quel que soit ce groupe sur la machine.
set -eu

: "${HOST_UID:?HOST_UID manquant : lancez make avec ./make}" "${HOST_GID:?HOST_GID manquant : lancez make avec ./make}"

sock=/var/run/docker.sock
groups=$HOST_GID
if [ -S "$sock" ]; then
  groups="$groups,$(stat -c %g "$sock")"
fi

# L'uid de la machine n'a pas de compte dans l'image : HOME pointe un dossier où il peut écrire (config de Docker).
export HOME=/tmp
exec setpriv --reuid "$HOST_UID" --regid "$HOST_GID" --groups "$groups" make "$@"
