#!/bin/bash
# Initialisation de la pile ELK de démonstration : cycle de vie et modèles d'index (Elasticsearch), puis vues de
# données et dashboard (Kibana). Idempotent : on peut le relancer sans dégâts (`make elk`).
set -u

ES="${ELASTICSEARCH_URL:-http://elasticsearch:9200}"
KIBANA="${KIBANA_URL:-http://kibana:5601}"
DIR=/elk
MAX_TRIES=150 # 150 × 2 s : Elasticsearch et Kibana mettent une à deux minutes à démarrer

wait_for() { # $1 nom, $2 URL, $3 motif attendu dans la réponse
  local tries=0
  until curl -fsS "$2" 2>/dev/null | grep -q "$3"; do
    tries=$((tries + 1))
    if [ "$tries" -gt "$MAX_TRIES" ]; then
      echo "Délai dépassé en attendant $1 ($2)." >&2
      exit 1
    fi
    sleep 2
  done
}

put() { # $1 URL, $2 fichier JSON
  curl -fsS -X PUT "$1" -H 'Content-Type: application/json' --data-binary "@$2" >/dev/null || {
    echo "Échec : PUT $1" >&2
    exit 1
  }
}

data_view() { # $1 identifiant, $2 motif d'index
  curl -fsS -X POST "$KIBANA/api/data_views/data_view" -H 'kbn-xsrf: true' -H 'Content-Type: application/json' \
    -d "{\"override\":true,\"data_view\":{\"id\":\"$1\",\"title\":\"$2\",\"timeFieldName\":\"@timestamp\"}}" >/dev/null || {
    echo "Échec : vue de données $2" >&2
    exit 1
  }
}

wait_for Elasticsearch "$ES/_cluster/health" '"status":"\(green\|yellow\)"'
put "$ES/_ilm/policy/herossos-7d" "$DIR/elasticsearch/ilm-policy.json"
put "$ES/_index_template/herossos-logs" "$DIR/elasticsearch/template-logs.json"
put "$ES/_index_template/herossos-events" "$DIR/elasticsearch/template-events.json"
echo "Elasticsearch : cycle de vie (7 jours) et modèles d'index en place."

wait_for Kibana "$KIBANA/api/status" '"level":"available"'
data_view herossos-logs 'herossos-logs-*'
data_view herossos-events 'herossos-events-*'
echo "Kibana : vues de données créées."

# Le dashboard est écrit à la main (voir dashboards.ndjson) : son import n'est pas bloquant. S'il échoue, les vues de
# données restent utilisables dans Discover, et les graphiques se recréent dans Lens (voir stack/performance-et-synchronisation.md).
if curl -fsS -X POST "$KIBANA/api/saved_objects/_import?overwrite=true" -H 'kbn-xsrf: true' \
  --form "file=@$DIR/kibana/dashboards.ndjson" -o /tmp/import.json && grep -q '"success":true' /tmp/import.json; then
  echo "Kibana : dashboard « Heros-SOS : performance et synchronisation » importé."
else
  echo "Kibana : l'import du dashboard a échoué (non bloquant) :" >&2
  cat /tmp/import.json >&2 2>/dev/null
fi
echo "Prêt : http://localhost:${KIBANA_PORT:-5601}"
