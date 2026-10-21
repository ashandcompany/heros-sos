// Démarre Swagger UI sur le contrat de l'API. Ce code est dans un fichier à part : la politique de
// sécurité de la page (CSP) interdit les scripts en ligne.
window.onload = function () {
  window.ui = SwaggerUIBundle({
    // Relatif à la page (/api/v1/docs) : le contrat est servi par la même API.
    url: "docs/openapi.yaml",
    dom_id: "#swagger-ui",
    deepLinking: true,
    // Par défaut, Swagger UI envoie l'adresse du contrat à validator.swagger.io. Aucun appel extérieur.
    validatorUrl: null,
    // Pas de barre d'adresse (preset « standalone ») : la page ne sert qu'à ce contrat.
    presets: [SwaggerUIBundle.presets.apis],
    layout: "BaseLayout",
  });
};
