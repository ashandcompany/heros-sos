// Package openapi embarque le contrat OpenAPI de l'API. Le fichier openapi.yaml reste la référence
// (voir le README) ; ce paquet le rend disponible au code, pour le servir dans la documentation
// interactive.
package openapi

import _ "embed"

// Spec contient le contenu de openapi.yaml.
//
//go:embed openapi.yaml
var Spec []byte
