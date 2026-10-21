# OMDb pour les films

[OMDb](https://omdbapi.com/) remplace « l'API IMDb » du cahier, qui est payante. Il couvre :

- `?s=titre` pour chercher,
- `?i=tt…` pour le détail par identifiant IMDb,
- `?t=titre` pour un titre exact,
- une réponse en JSON.

Le quota gratuit par jour n'est pas indiqué sur le site : à regarder au moment de demander la clé, qui arrive par email. Autres points :

- **Licence :** le contenu est sous CC BY-NC 4.0 (non commercial), ce qui est acceptable pour un projet pédagogique.
- **Posters :** l'API d'images dédiée est réservée aux patrons.
- **Clé :** elle reste côté backend, dans `.env`, jamais envoyée au front.
- **Cache :** le film est stocké en base à l'ajout, ce qui économise le quota. De toute façon un modérateur doit valider chaque film (§5.8.4).

OMDb ne remplace pas l'encyclopédie Marvel du §6.1.4, qui reste à trancher.
