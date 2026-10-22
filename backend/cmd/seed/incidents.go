package main

// Issue d'un incident de démonstration : elle décide de l'intervention créée avec lui.
const (
	outcomeReported   = "reported"    // déclaré, aucune intervention
	outcomeInProgress = "in_progress" // intervention en cours, héros actifs
	outcomeProposed   = "proposed"    // intervention en cours, issue proposée par le lead, en attente de validation
	outcomeSucceeded  = "succeeded"   // intervention clôturée avec succès, avis des citoyens
	outcomeFailed     = "failed"      // intervention clôturée en échec
)

type incidentSeed struct {
	Title       string
	Type        string
	Priority    string
	Description string
	Address     address
	Lat, Lon    float64
	Reporter    string   // clé d'un citoyen
	Villains    []string // alias des vilains impliqués
	Outcome     string
	Heroes      []string // clés des héros ; le premier mène l'intervention
	Invited     []string // héros invités qui n'ont pas encore répondu
	Rescued     int16
	DaysAgo     int
	Notes       string
}

// Un héros n'a qu'une intervention active à la fois : les incidents en cours n'en partagent aucun.
var incidents = []incidentSeed{
	// --- En cours -----------------------------------------------------------------------------
	{
		Title: "Vaisseau kree au-dessus du Capitole", Type: "menace-cosmique", Priority: "high",
		Description: "Un vaisseau gigantesque est apparu au-dessus de la place du Capitole. Des soldats en armure bleue descendent par des faisceaux lumineux.",
		Address:     address{"Place du Capitole", "31000", "Toulouse", "FR"}, Lat: 43.604468, Lon: 1.443360,
		Reporter: "jane.foster", Villains: []string{"Ronan l'Accusateur"}, Outcome: outcomeInProgress,
		Heroes: []string{"thor.odinson", "carol.danvers"}, DaysAgo: 0,
	},
	{
		Title: "Homme-sable dans le métro", Type: "braquage", Priority: "high",
		Description: "Un homme fait de sable a envahi la station Châtelet et vide les distributeurs de billets. Les voyageurs sont bloqués sur le quai.",
		Address:     address{"Place du Châtelet", "75001", "Paris", "FR"}, Lat: 48.858370, Lon: 2.347000,
		Reporter: "mary.jane.watson", Villains: []string{"Sandman"}, Outcome: outcomeInProgress,
		Heroes: []string{"peter.parker", "matt.murdock"}, Invited: []string{"kamala.khan"}, DaysAgo: 0,
	},
	{
		Title: "Portail dimensionnel au Vieux-Lyon", Type: "anomalie-mystique", Priority: "high",
		Description: "Un cercle de flammes orange tourne au milieu de la rue Saint-Jean. Des créatures en sortent par intermittence.",
		Address:     address{"Rue Saint-Jean", "69005", "Lyon", "FR"}, Lat: 45.762500, Lon: 4.827300,
		Reporter: "may.parker", Villains: []string{"Dormammu"}, Outcome: outcomeInProgress,
		Heroes: []string{"stephen.strange", "wanda.maximoff"}, DaysAgo: 1,
	},
	{
		Title: "Otages à l'ambassade", Type: "prise-otages", Priority: "high",
		Description: "Un groupe armé retient une quinzaine de personnes dans une ambassade. Une femme masquée semble imiter les techniques de combat des agents de sécurité.",
		Address:     address{"Avenue Gabriel", "75008", "Paris", "FR"}, Lat: 48.867700, Lon: 2.317200,
		Reporter: "jonah.jameson", Villains: []string{"Taskmaster"}, Outcome: outcomeInProgress,
		Heroes: []string{"natasha.romanoff", "clint.barton"}, DaysAgo: 1,
	},
	{
		Title: "Inondation du Vieux-Port", Type: "catastrophe", Priority: "high",
		Description: "Une vague anormale a submergé les quais. Plusieurs bateaux ont été projetés sur la chaussée, des personnes sont coincées dans les restaurants.",
		Address:     address{"Quai du Port", "13002", "Marseille", "FR"}, Lat: 43.296500, Lon: 5.369800,
		Reporter: "betty.ross", Outcome: outcomeProposed,
		Heroes: []string{"ororo.munroe", "sam.wilson"}, Rescued: 23, DaysAgo: 2,
		Notes: "Toutes les personnes évacuées, la vague a été repoussée vers le large.",
	},
	{
		Title: "Fourmis géantes au laboratoire", Type: "accident", Priority: "normal",
		Description: "Des fourmis de la taille d'un chien se sont échappées d'un laboratoire de l'université. Elles semblent dociles mais bloquent le campus.",
		Address:     address{"351 cours de la Libération", "33400", "Talence", "FR"}, Lat: 44.806700, Lon: -0.605400,
		Reporter: "happy.hogan", Outcome: outcomeInProgress,
		Heroes: []string{"scott.lang", "hope.vandyne"}, DaysAgo: 2,
	},

	// --- Déclarés, en attente d'un héros -------------------------------------------------------
	{
		Title: "Drones hostiles sur la Défense", Type: "cyberattaque", Priority: "high",
		Description: "Des dizaines de drones survolent le parvis et projettent des images géantes. Les écrans publicitaires affichent un message de menace.",
		Address:     address{"Parvis de la Défense", "92400", "Courbevoie", "FR"}, Lat: 48.891900, Lon: 2.238500,
		Reporter: "jonah.jameson", Villains: []string{"Mysterio"}, Outcome: outcomeReported, DaysAgo: 0,
	},
	{
		Title: "Panne électrique géante", Type: "sabotage", Priority: "high",
		Description: "Tout le quartier est privé d'électricité. Des témoins ont vu un homme lumineux absorber le courant du transformateur.",
		Address:     address{"Boulevard de la Liberté", "59000", "Lille", "FR"}, Lat: 50.633300, Lon: 3.058600,
		Reporter: "darcy.lewis", Villains: []string{"Electro"}, Outcome: outcomeReported, DaysAgo: 0,
	},
	{
		Title: "Chat coincé en haut de la cathédrale", Type: "accident", Priority: "low",
		Description: "Un chat est bloqué sur la flèche de la cathédrale depuis ce matin. Les pompiers n'ont pas d'échelle assez haute.",
		Address:     address{"Place de la Cathédrale", "67000", "Strasbourg", "FR"}, Lat: 48.581800, Lon: 7.751000,
		Reporter: "karen.page", Outcome: outcomeReported, DaysAgo: 0,
	},
	{
		Title: "Armes extraterrestres en vente sur le port", Type: "trafic-armes", Priority: "normal",
		Description: "Des caisses de matériel lumineux sont échangées la nuit sur le quai. Un homme portait des ailes mécaniques.",
		Address:     address{"Quai de la Fosse", "44000", "Nantes", "FR"}, Lat: 47.211900, Lon: -1.567000,
		Reporter: "foggy.nelson", Villains: []string{"Vulture"}, Outcome: outcomeReported, DaysAgo: 1,
	},
	{
		Title: "Feu de forêt d'origine inconnue", Type: "incendie", Priority: "high",
		Description: "Un incendie se propage très vite dans le massif. Les flammes ont une couleur bleutée inhabituelle.",
		Address:     address{"Route des Crêtes", "13260", "Cassis", "FR"}, Lat: 43.208000, Lon: 5.555500,
		Reporter: "pepper.potts", Outcome: outcomeReported, DaysAgo: 1,
	},
	{
		Title: "Enlèvement d'un chercheur", Type: "enlevement", Priority: "high",
		Description: "Un physicien du CNRS a été enlevé devant son laboratoire par des hommes en uniforme vert. Un véhicule sans plaque est parti vers l'autoroute.",
		Address:     address{"Avenue de la Terrasse", "91190", "Gif-sur-Yvette", "FR"}, Lat: 48.708700, Lon: 2.147500,
		Reporter: "jane.foster", Villains: []string{"Baron Zemo"}, Outcome: outcomeReported, DaysAgo: 2,
	},
	{
		Title: "Symbiote dans les égouts", Type: "attaque-surhumaine", Priority: "normal",
		Description: "Une substance noire vivante a été aperçue dans les égouts. Deux agents d'entretien ont été blessés.",
		Address:     address{"Rue Sainte-Catherine", "33000", "Bordeaux", "FR"}, Lat: 44.840600, Lon: -0.573900,
		Reporter: "happy.hogan", Villains: []string{"Venom"}, Outcome: outcomeReported, DaysAgo: 3,
	},
	{
		Title: "Distorsion temporelle au musée", Type: "anomalie-mystique", Priority: "normal",
		Description: "Les visiteurs vieillissent et rajeunissent en traversant la galerie égyptienne. Un homme en armure violette y a été vu.",
		Address:     address{"Rue de Rivoli", "75001", "Paris", "FR"}, Lat: 48.860600, Lon: 2.337600,
		Reporter: "mary.jane.watson", Villains: []string{"Kang le Conquérant"}, Outcome: outcomeReported, DaysAgo: 3,
	},
	{
		Title: "Vol de la collection de vibranium", Type: "braquage", Priority: "normal",
		Description: "La vitrine d'objets africains du musée a été forcée. Le voleur a laissé une griffe en métal noir.",
		Address:     address{"Quai Branly", "75007", "Paris", "FR"}, Lat: 48.860900, Lon: 2.297800,
		Reporter: "karen.page", Villains: []string{"Killmonger"}, Outcome: outcomeReported, DaysAgo: 4,
	},
	{
		Title: "Brouillard toxique sur la promenade", Type: "accident", Priority: "low",
		Description: "Un brouillard verdâtre stagne au-dessus de la plage. Pas de blessé, mais une odeur de soufre très forte.",
		Address:     address{"Promenade des Anglais", "06000", "Nice", "FR"}, Lat: 43.695000, Lon: 7.265000,
		Reporter: "ned.leeds", Outcome: outcomeReported, DaysAgo: 5,
	},
	{
		Title: "Robots détraqués à Times Square", Type: "cyberattaque", Priority: "high",
		Description: "Des robots de sécurité attaquent les passants. Leurs yeux brillent en rouge et ils répètent « Il n'y a pas de cordes sur moi ».",
		Address:     address{"Times Square", "10036", "New York", "US"}, Lat: 40.758000, Lon: -73.985500,
		Reporter: "mary.jane.watson", Villains: []string{"Ultron"}, Outcome: outcomeReported, DaysAgo: 6,
	},

	// --- Réussis ------------------------------------------------------------------------------
	{
		Title: "Invasion chitauri à Manhattan", Type: "menace-cosmique", Priority: "high",
		Description: "Un portail s'est ouvert au-dessus de la Stark Tower. Des créatures volantes attaquent les immeubles alentour.",
		Address:     address{"200 Park Avenue", "10166", "New York", "US"}, Lat: 40.753200, Lon: -73.976300,
		Reporter: "pepper.potts", Villains: []string{"Loki"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"tony.stark", "steve.rogers", "thor.odinson", "bruce.banner", "natasha.romanoff", "clint.barton"},
		Rescued: 312, DaysAgo: 58, Notes: "Portail refermé, Loki remis aux autorités asgardiennes.",
	},
	{
		Title: "Ultron prend le contrôle des tours", Type: "cyberattaque", Priority: "high",
		Description: "Les ascenseurs et les portes de plusieurs tours sont bloqués. Une voix métallique parle dans les haut-parleurs.",
		Address:     address{"1 parvis de la Défense", "92800", "Puteaux", "FR"}, Lat: 48.892600, Lon: 2.236000,
		Reporter: "jonah.jameson", Villains: []string{"Ultron"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"tony.stark", "carol.danvers", "peter.parker"},
		Rescued: 304, DaysAgo: 34, Notes: "Copie d'Ultron isolée et effacée, évacuation complète.",
	},
	{
		Title: "Braquage du train Paris-Lyon", Type: "braquage", Priority: "high",
		Description: "Un homme avec quatre bras mécaniques a arrêté le TGV en pleine voie et vide le wagon postal.",
		Address:     address{"Gare de Lyon-Part-Dieu", "69003", "Lyon", "FR"}, Lat: 45.760500, Lon: 4.859500,
		Reporter: "may.parker", Villains: []string{"Doctor Octopus"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"peter.parker", "kamala.khan"},
		Rescued: 187, DaysAgo: 28, Notes: "Train arrêté avant le viaduc, Octavius neutralisé.",
	},
	{
		Title: "Attaque du Bouffon vert au marché de Noël", Type: "attaque-surhumaine", Priority: "high",
		Description: "Un homme sur un planeur lance des bombes en forme de citrouille sur les chalets.",
		Address:     address{"Place Broglie", "67000", "Strasbourg", "FR"}, Lat: 48.584000, Lon: 7.749600,
		Reporter: "karen.page", Villains: []string{"Green Goblin"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"peter.parker", "matt.murdock"},
		Rescued: 41, DaysAgo: 45, Notes: "Osborn arrêté, planeur saisi.",
	},
	{
		Title: "Incendie de l'usine pétrochimique", Type: "incendie", Priority: "high",
		Description: "Explosion dans une raffinerie, le feu menace les cuves voisines. Des ouvriers sont bloqués dans la salle de contrôle.",
		Address:     address{"Route de la Raffinerie", "13117", "Lavéra", "FR"}, Lat: 43.388900, Lon: 5.013900,
		Reporter: "betty.ross", Villains: []string{"Whiplash"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"ororo.munroe", "tony.stark"},
		Rescued: 18, DaysAgo: 40, Notes: "Incendie maîtrisé par la pluie de Storm, Vanko interpellé.",
	},
	{
		Title: "Glissement de terrain à Chamonix", Type: "catastrophe", Priority: "high",
		Description: "Une coulée de boue a emporté une partie du hameau. Plusieurs chalets sont ensevelis.",
		Address:     address{"Route des Praz", "74400", "Chamonix-Mont-Blanc", "FR"}, Lat: 45.937500, Lon: 6.880000,
		Reporter: "darcy.lewis", Outcome: outcomeSucceeded,
		Heroes:  []string{"bruce.banner", "sam.wilson", "ororo.munroe"},
		Rescued: 27, DaysAgo: 52, Notes: "Tous les habitants retrouvés vivants.",
	},
	{
		Title: "Prise d'otages à la banque centrale", Type: "prise-otages", Priority: "high",
		Description: "Des hommes armés retiennent les employés de la succursale. Leur chef se présente comme le « Caïd ».",
		Address:     address{"31 rue Croix-des-Petits-Champs", "75001", "Paris", "FR"}, Lat: 48.864400, Lon: 2.339700,
		Reporter: "foggy.nelson", Villains: []string{"Kingpin"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"matt.murdock", "natasha.romanoff"},
		Rescued: 22, DaysAgo: 31, Notes: "Otages libérés sans blessé, Fisk arrêté.",
	},
	{
		Title: "Hela à la cathédrale de Reims", Type: "anomalie-mystique", Priority: "high",
		Description: "Une femme coiffée de bois de cerf fait surgir des épées noires du sol. Des soldats squelettes l'accompagnent.",
		Address:     address{"Place du Cardinal Luçon", "51100", "Reims", "FR"}, Lat: 49.253900, Lon: 4.034100,
		Reporter: "jane.foster", Villains: []string{"Hela"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"thor.odinson", "stephen.strange"},
		Rescued: 64, DaysAgo: 25, Notes: "Hela renvoyée à Hel, la cathédrale a souffert.",
	},
	{
		Title: "Abomination dans le stade", Type: "attaque-surhumaine", Priority: "high",
		Description: "Une créature reptilienne de trois mètres a envahi la pelouse en plein match.",
		Address:     address{"Avenue de Saint-Exupéry", "13008", "Marseille", "FR"}, Lat: 43.269800, Lon: 5.395900,
		Reporter: "betty.ross", Villains: []string{"Abomination"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"bruce.banner", "shang.chi"},
		Rescued: 120, DaysAgo: 20, Notes: "Blonsky maîtrisé, aucun spectateur blessé grièvement.",
	},
	{
		Title: "Trafic d'armes chitauri à Queens", Type: "trafic-armes", Priority: "normal",
		Description: "Des adolescents utilisent des armes extraterrestres pour braquer des épiceries.",
		Address:     address{"Queens Boulevard", "11375", "Forest Hills", "US"}, Lat: 40.721700, Lon: -73.844600,
		Reporter: "ned.leeds", Villains: []string{"Vulture"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"peter.parker"},
		Rescued: 4, DaysAgo: 48, Notes: "Toomes arrêté, armes remises au Département du contrôle des dommages.",
	},
	{
		Title: "Mutants en danger au tribunal", Type: "enlevement", Priority: "high",
		Description: "Des sentinelles ont enlevé trois adolescents mutants à la sortie du tribunal.",
		Address:     address{"Place de la Bourse", "33000", "Bordeaux", "FR"}, Lat: 44.841400, Lon: -0.569700,
		Reporter: "happy.hogan", Villains: []string{"Magneto"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"logan", "ororo.munroe"},
		Rescued: 3, DaysAgo: 37, Notes: "Adolescents retrouvés sains et saufs.",
	},
	{
		Title: "Accident de bus sur le pont", Type: "accident", Priority: "high",
		Description: "Un bus scolaire est suspendu au-dessus du fleuve après une collision.",
		Address:     address{"Pont de Normandie", "76600", "Le Havre", "FR"}, Lat: 49.432200, Lon: 0.274400,
		Reporter: "may.parker", Outcome: outcomeSucceeded,
		Heroes:  []string{"peter.parker", "sam.wilson"},
		Rescued: 32, DaysAgo: 15, Notes: "Tous les enfants récupérés.",
	},
	{
		Title: "Vol de l'Éther au Muséum", Type: "menace-cosmique", Priority: "high",
		Description: "Une substance rouge s'est échappée d'une vitrine et absorbe la lumière de toute la galerie.",
		Address:     address{"57 rue Cuvier", "75005", "Paris", "FR"}, Lat: 48.843900, Lon: 2.359800,
		Reporter: "jane.foster", Villains: []string{"Malekith"}, Outcome: outcomeSucceeded,
		Heroes:  []string{"thor.odinson", "carol.danvers"},
		Rescued: 9, DaysAgo: 12, Notes: "Éther confiné et remis au Collectionneur.",
	},
	{
		Title: "Rixe surnaturelle au British Museum", Type: "anomalie-mystique", Priority: "normal",
		Description: "Un homme en bandages blancs combat des chacals géants dans la salle égyptienne.",
		Address:     address{"Great Russell Street", "WC1B 3DG", "Londres", "GB"}, Lat: 51.519400, Lon: -0.127000,
		Reporter: "darcy.lewis", Outcome: outcomeSucceeded,
		Heroes:  []string{"marc.spector", "stephen.strange"},
		Rescued: 6, DaysAgo: 10,
	},
	{
		Title: "Fuite de particules Pym", Type: "accident", Priority: "normal",
		Description: "Des voitures rétrécissent et grandissent aléatoirement sur le parking du centre commercial.",
		Address:     address{"Avenue de l'Europe", "44800", "Saint-Herblain", "FR"}, Lat: 47.223400, Lon: -1.627700,
		Reporter: "foggy.nelson", Outcome: outcomeSucceeded,
		Heroes:  []string{"scott.lang", "hope.vandyne"},
		Rescued: 11, DaysAgo: 8,
	},
	{
		Title: "Les Dix Anneaux à Chinatown", Type: "attaque-surhumaine", Priority: "normal",
		Description: "Des hommes masqués ont attaqué un restaurant. L'un d'eux portait une machette qui se transforme.",
		Address:     address{"Avenue d'Ivry", "75013", "Paris", "FR"}, Lat: 48.823000, Lon: 2.365000,
		Reporter: "mary.jane.watson", Outcome: outcomeSucceeded,
		Heroes:  []string{"shang.chi", "steve.rogers"},
		Rescued: 14, DaysAgo: 5,
	},

	// --- Échoués ------------------------------------------------------------------------------
	{
		Title: "Thanos aperçu à Vormir… euh, à Annecy", Type: "menace-cosmique", Priority: "high",
		Description: "Un géant violet portant un gant doré a traversé la vieille ville. Plusieurs personnes ont disparu en poussière.",
		Address:     address{"Quai de l'Île", "74000", "Annecy", "FR"}, Lat: 45.898700, Lon: 6.126000,
		Reporter: "pepper.potts", Villains: []string{"Thanos"}, Outcome: outcomeFailed,
		Heroes:  []string{"tony.stark", "steve.rogers", "tchalla", "wanda.maximoff"},
		Rescued: 50, DaysAgo: 55, Notes: "Thanos s'est téléporté avec le gant. Recherches en cours.",
	},
	{
		Title: "Loki s'évade du convoi", Type: "prise-otages", Priority: "high",
		Description: "Le convoi blindé transportant Loki a été attaqué. Il a pris le chauffeur en otage.",
		Address:     address{"Autoroute A7", "26200", "Montélimar", "FR"}, Lat: 44.558000, Lon: 4.750000,
		Reporter: "jonah.jameson", Villains: []string{"Loki"}, Outcome: outcomeFailed,
		Heroes:  []string{"thor.odinson", "bucky.barnes"},
		Rescued: 1, DaysAgo: 30, Notes: "Le chauffeur est sain et sauf, mais Loki s'est volatilisé (illusion).",
	},
	{
		Title: "Doom pirate le réseau bancaire", Type: "cyberattaque", Priority: "normal",
		Description: "Tous les distributeurs de la ville affichent un blason vert et un masque de métal.",
		Address:     address{"Rue de la République", "69002", "Lyon", "FR"}, Lat: 45.763000, Lon: 4.835700,
		Reporter: "may.parker", Villains: []string{"Doctor Doom"}, Outcome: outcomeFailed,
		Heroes:  []string{"tony.stark", "kamala.khan"},
		DaysAgo: 22, Notes: "Réseau restauré mais l'attaquant reste introuvable (immunité diplomatique).",
	},
	{
		Title: "Le Crâne rouge vend des plans secrets", Type: "trafic-armes", Priority: "normal",
		Description: "Un homme au visage rouge a été vu négociant des documents classifiés près de l'arsenal.",
		Address:     address{"Rue de Siam", "29200", "Brest", "FR"}, Lat: 48.387100, Lon: -4.488100,
		Reporter: "karen.page", Villains: []string{"Red Skull"}, Outcome: outcomeFailed,
		Heroes:  []string{"steve.rogers", "bucky.barnes"},
		DaysAgo: 18, Notes: "Schmidt a disparu par le port.",
	},
	{
		Title: "Wolverine contre un inconnu", Type: "attaque-surhumaine", Priority: "low",
		Description: "Bagarre dans un bar entre un homme aux griffes de métal et un mercenaire en rouge qui refuse de mourir.",
		Address:     address{"Rue Masséna", "06000", "Nice", "FR"}, Lat: 43.697500, Lon: 7.268700,
		Reporter: "ned.leeds", Outcome: outcomeFailed,
		Heroes:  []string{"logan"},
		DaysAgo: 7, Notes: "Le mercenaire s'est enfui en faisant des blagues.",
	},
}
