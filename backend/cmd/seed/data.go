package main

// Données de démonstration. Les clés (champ Key, codes de type, identifiants de film) relient les
// enregistrements entre eux ; elles ne sont pas stockées telles quelles.

const emailDomain = "heros-sos.local"

type address struct {
	Street, PostalCode, City, Country string
}

type person struct {
	Key       string // partie locale de l'adresse email
	FirstName string
	LastName  string
	Civility  string
	BirthDate string
	Phone     string
	Address   address
}

type moderatorSeed struct {
	person
}

type heroSeed struct {
	person
	Alias          string
	Bio            string
	RequestMessage string
	Status         string // approved | pending | rejected
	Rejection      string
	Films          []string
	PendingFilms   []string // proposées par le héros, en attente de modération
}

type citizenSeed struct {
	person
	Status     string // active (défaut) | pending_activation | deactivated
	Newsletter bool
}

type incidentTypeSeed struct {
	Code          string
	Label         string
	EndangersLife bool
}

type villainSeed struct {
	Alias       string
	FirstName   string
	LastName    string
	Sector      string
	Description string
	Threat      string // faible | modere | eleve | critique
	Status      string // en-liberte | emprisonne
	Specialties []string
	Films       []string
}

type filmSeed struct {
	IMDbID string
	Title  string
	Year   int16
	Plot   string
}

type articleSeed struct {
	Title     string
	Slug      string
	Excerpt   string
	Content   string
	Published bool
	DaysAgo   int
}

var moderators = []moderatorSeed{
	{person{Key: "admin", FirstName: "Nick", LastName: "Fury", Civility: "monsieur", BirthDate: "1950-12-04",
		Phone: "+12125550100", Address: address{"1 Triskelion Way", "20001", "Washington", "US"}}},
	{person{Key: "maria.hill", FirstName: "Maria", LastName: "Hill", Civility: "madame", BirthDate: "1980-04-05",
		Phone: "+12125550101"}},
	{person{Key: "phil.coulson", FirstName: "Phil", LastName: "Coulson", Civility: "monsieur", BirthDate: "1964-07-08",
		Phone: "+12125550102"}},
}

var incidentTypes = []incidentTypeSeed{
	{"attaque-surhumaine", "Attaque surhumaine", true},
	{"prise-otages", "Prise d'otages", true},
	{"braquage", "Braquage", false},
	{"incendie", "Incendie", true},
	{"catastrophe", "Catastrophe naturelle", true},
	{"menace-cosmique", "Menace extraterrestre", true},
	{"cyberattaque", "Cyberattaque", false},
	{"accident", "Accident grave", true},
	{"enlevement", "Enlèvement", true},
	{"sabotage", "Sabotage industriel", false},
	{"anomalie-mystique", "Anomalie mystique", true},
	{"trafic-armes", "Trafic d'armes", false},
}

var heroes = []heroSeed{
	{
		person: person{Key: "tony.stark", FirstName: "Tony", LastName: "Stark", Civility: "monsieur", BirthDate: "1970-05-29",
			Phone: "+13105550142", Address: address{"10880 Malibu Point", "90265", "Malibu", "US"}},
		Alias:  "Iron Man",
		Bio:    "Génie, milliardaire, philanthrope. Pilote d'armures de combat conçues dans son propre atelier, spécialiste des interventions aériennes et technologiques.",
		Status: "approved", Films: []string{"tt0371746", "tt1228705", "tt1300854", "tt0848228", "tt4154796"},
	},
	{
		person: person{Key: "steve.rogers", FirstName: "Steve", LastName: "Rogers", Civility: "monsieur", BirthDate: "1918-07-04",
			Address: address{"569 Leaman Place", "11201", "Brooklyn", "US"}},
		Alias:  "Captain America",
		Bio:    "Super-soldat sorti de la Seconde Guerre mondiale. Tacticien hors pair, il coordonne les équipes sur le terrain, bouclier au bras.",
		Status: "approved", Films: []string{"tt0458339", "tt1843866", "tt3498820", "tt0848228"},
	},
	{
		person: person{Key: "thor.odinson", FirstName: "Thor", LastName: "Odinson", Civility: "monsieur", BirthDate: "1983-03-21",
			Address: address{"Nedre Langgate 1", "3126", "Tønsberg", "NO"}},
		Alias:  "Thor",
		Bio:    "Dieu du tonnerre et roi de New Asgard. Intervient sur les menaces cosmiques et les catastrophes climatiques, Stormbreaker à la main.",
		Status: "approved", Films: []string{"tt0800369", "tt1981115", "tt3501632"},
	},
	{
		person: person{Key: "natasha.romanoff", FirstName: "Natasha", LastName: "Romanoff", Civility: "madame", BirthDate: "1984-11-22"},
		Alias:  "Black Widow",
		Bio:    "Ancienne espionne formée dans la Chambre Rouge. Infiltration, négociation de prises d'otages et extractions discrètes.",
		Status: "approved", Films: []string{"tt3480822", "tt1843866"},
	},
	{
		person: person{Key: "bruce.banner", FirstName: "Bruce", LastName: "Banner", Civility: "monsieur", BirthDate: "1969-12-18",
			Address: address{"Culver University", "22901", "Charlottesville", "US"}},
		Alias:  "Hulk",
		Bio:    "Physicien spécialiste des rayons gamma. Réservé aux situations où il faut soulever un immeuble ou arrêter quelque chose de très gros.",
		Status: "approved", Films: []string{"tt0800080", "tt3501632"},
	},
	{
		person: person{Key: "clint.barton", FirstName: "Clint", LastName: "Barton", Civility: "monsieur", BirthDate: "1971-01-07",
			Address: address{"Barton Farm Road", "63401", "Hannibal", "US"}},
		Alias:  "Hawkeye",
		Bio:    "Archer d'élite, ne rate jamais sa cible. Couverture en hauteur et neutralisation à distance.",
		Status: "approved", Films: []string{"tt10160804", "tt0848228"},
	},
	{
		person: person{Key: "peter.parker", FirstName: "Peter", LastName: "Parker", Civility: "monsieur", BirthDate: "2001-08-10",
			Address: address{"20 Ingram Street", "11375", "Forest Hills", "US"}},
		Alias:  "Spider-Man",
		Bio:    "Votre sympathique voisin. Toile, réflexes surhumains et « sens d'araignée » : idéal pour les sauvetages en milieu urbain.",
		Status: "approved", Films: []string{"tt2250912", "tt6320628", "tt10872600"},
	},
	{
		person: person{Key: "stephen.strange", FirstName: "Stephen", LastName: "Strange", Civility: "monsieur", BirthDate: "1976-11-18",
			Address: address{"177A Bleecker Street", "10012", "New York", "US"}},
		Alias:  "Doctor Strange",
		Bio:    "Ancien neurochirurgien devenu Sorcier suprême. Toute anomalie mystique ou incursion dimensionnelle relève de son sanctuaire.",
		Status: "approved", Films: []string{"tt1211837", "tt9419884"},
	},
	{
		person: person{Key: "tchalla", FirstName: "T'Challa", LastName: "Udaku", Civility: "monsieur", BirthDate: "1977-11-29"},
		Alias:  "Black Panther",
		Bio:    "Roi du Wakanda, protégé par une armure en vibranium. Intervient en appui des opérations internationales.",
		Status: "approved", Films: []string{"tt1825683", "tt3498820"},
	},
	{
		person: person{Key: "carol.danvers", FirstName: "Carol", LastName: "Danvers", Civility: "madame", BirthDate: "1965-04-24"},
		Alias:  "Captain Marvel",
		Bio:    "Ancienne pilote de l'US Air Force aux pouvoirs cosmiques. Première appelée quand la menace vient de l'espace.",
		Status: "approved", Films: []string{"tt4154664", "tt4154796"},
	},
	{
		person: person{Key: "wanda.maximoff", FirstName: "Wanda", LastName: "Maximoff", Civility: "madame", BirthDate: "1989-02-10",
			Address: address{"2800 Sherwood Drive", "07524", "Westview", "US"}},
		Alias:  "Scarlet Witch",
		Bio:    "Maîtrise du chaos et de la télékinésie. Ses interventions sont spectaculaires, ses rapports un peu moins détaillés.",
		Status: "approved", Films: []string{"tt9140560", "tt2395427"},
	},
	{
		person: person{Key: "scott.lang", FirstName: "Scott", LastName: "Lang", Civility: "monsieur", BirthDate: "1969-04-06",
			Address: address{"1501 Mission Street", "94103", "San Francisco", "US"}},
		Alias:  "Ant-Man",
		Bio:    "Ancien cambrioleur repenti, rétrécit à volonté grâce aux particules Pym. Parfait pour les infiltrations… et les serrures.",
		Status: "approved", Films: []string{"tt0478970", "tt5095030", "tt10954600"},
	},
	{
		person: person{Key: "hope.vandyne", FirstName: "Hope", LastName: "van Dyne", Civility: "madame", BirthDate: "1979-09-01",
			Address: address{"1500 Pine Street", "94109", "San Francisco", "US"}},
		Alias:  "The Wasp",
		Bio:    "Dirigeante de Pym Technologies. Vol, rétrécissement et décharges bioélectriques.",
		Status: "approved", Films: []string{"tt5095030", "tt10954600"},
	},
	{
		person: person{Key: "sam.wilson", FirstName: "Sam", LastName: "Wilson", Civility: "monsieur", BirthDate: "1978-09-23",
			Address: address{"14 Dumaine Street", "70116", "La Nouvelle-Orléans", "US"}},
		Alias:  "Falcon",
		Bio:    "Ancien parachutiste, ailes en carbone et drone Redwing. Sauvetages aériens et évacuations.",
		Status: "approved", Films: []string{"tt9208876", "tt1843866"},
	},
	{
		person: person{Key: "bucky.barnes", FirstName: "James", LastName: "Barnes", Civility: "monsieur", BirthDate: "1917-03-10"},
		Alias:  "Winter Soldier",
		Bio:    "Ancien sergent de la 107e, bras en vibranium. En reconversion, il préfère désormais les missions de protection.",
		Status: "approved", Films: []string{"tt1843866", "tt9208876"},
	},
	{
		person: person{Key: "matt.murdock", FirstName: "Matt", LastName: "Murdock", Civility: "monsieur", BirthDate: "1982-12-13",
			Address: address{"West 44th Street", "10036", "New York", "US"}},
		Alias:  "Daredevil",
		Bio:    "Avocat le jour, justicier aveugle la nuit à Hell's Kitchen. Ses sens surdéveloppés compensent largement.",
		Status: "approved", Films: []string{"tt3322312"},
	},
	{
		person: person{Key: "logan", FirstName: "James", LastName: "Howlett", Civility: "monsieur", BirthDate: "1882-01-01"},
		Alias:  "Wolverine",
		Bio:    "Griffes en adamantium et facteur guérisseur. Le meilleur dans sa partie, et sa partie n'est pas très agréable.",
		Status: "approved", Films: []string{"tt0120903", "tt1877832", "tt3315342"},
	},
	{
		person: person{Key: "ororo.munroe", FirstName: "Ororo", LastName: "Munroe", Civility: "madame", BirthDate: "1975-10-25",
			Address: address{"1407 Graymalkin Lane", "10560", "North Salem", "US"}},
		Alias:  "Storm",
		Bio:    "Contrôle les éléments : incendies maîtrisés par la pluie, inondations détournées par le vent.",
		Status: "approved", Films: []string{"tt0120903", "tt1877832"},
	},
	{
		person: person{Key: "kamala.khan", FirstName: "Kamala", LastName: "Khan", Civility: "madame", BirthDate: "2006-09-15",
			Address: address{"Sip Avenue", "07306", "Jersey City", "US"}},
		Alias:  "Miss Marvel",
		Bio:    "Lycéenne de Jersey City aux pouvoirs de lumière solide. Grande fan des Avengers, encore plus depuis qu'elle travaille avec eux.",
		Status: "approved", Films: []string{"tt10857164"},
	},
	{
		person: person{Key: "shang.chi", FirstName: "Shang-Chi", LastName: "Xu", Civility: "monsieur", BirthDate: "1996-04-07",
			Address: address{"Clement Street", "94118", "San Francisco", "US"}},
		Alias:  "Shang-Chi",
		Bio:    "Maître des arts martiaux et porteur des Dix Anneaux. Ancien voiturier, il conduit très bien.",
		Status: "approved", Films: []string{"tt9376612"},
	},
	{
		person: person{Key: "marc.spector", FirstName: "Marc", LastName: "Spector", Civility: "monsieur", BirthDate: "1985-03-09",
			Address: address{"Great Russell Street", "WC1B 3DG", "Londres", "GB"}},
		Alias:  "Moon Knight",
		Bio:    "Avatar de Khonshou. Intervient la nuit ; contactez Steven Grant pour les questions d'égyptologie.",
		Status: "approved", Films: []string{"tt10234724"},
	},
	{
		person: person{Key: "peter.quill", FirstName: "Peter", LastName: "Quill", Civility: "monsieur", BirthDate: "1980-06-15"},
		Alias:  "Star-Lord",
		Bio:    "Chef autoproclamé des Gardiens de la Galaxie. Disponible sur Terre entre deux contrats, cassette audio fournie.",
		Status: "approved", Films: []string{"tt2015381"},
		PendingFilms: []string{"tt4154756"},
	},
	{
		person: person{Key: "jennifer.walters", FirstName: "Jennifer", LastName: "Walters", Civility: "madame", BirthDate: "1985-07-12",
			Address: address{"Wilshire Boulevard", "90010", "Los Angeles", "US"}},
		Alias:          "She-Hulk",
		Bio:            "Avocate spécialisée dans le droit des surhumains, cousine de Bruce Banner.",
		RequestMessage: "Je plaide le jour et je soulève des camions le soir. Je connais la procédure mieux que quiconque : merci de valider ma demande dans les délais légaux.",
		Status:         "pending",
	},
	{
		person:         person{Key: "doreen.green", FirstName: "Doreen", LastName: "Green", Civility: "madame", BirthDate: "2002-01-20"},
		Alias:          "Squirrel Girl",
		Bio:            "Étudiante en informatique, force et agilité d'écureuil. Invaincue (si, vraiment).",
		RequestMessage: "J'ai déjà battu Doctor Doom, Thanos et Galactus. Mes écureuils peuvent témoigner.",
		Status:         "pending",
	},
	{
		person: person{Key: "riri.williams", FirstName: "Riri", LastName: "Williams", Civility: "madame", BirthDate: "2007-05-03",
			Address: address{"77 Massachusetts Avenue", "02139", "Cambridge", "US"}},
		Alias:          "Ironheart",
		Bio:            "Étudiante au MIT, a construit son armure dans sa chambre d'internat.",
		RequestMessage: "Armure opérationnelle, tests de vol validés. Tony Stark aurait approuvé.",
		Status:         "pending",
	},
	{
		person:         person{Key: "wade.wilson", FirstName: "Wade", LastName: "Wilson", Civility: "monsieur", BirthDate: "1973-11-22"},
		Alias:          "Deadpool",
		Bio:            "Mercenaire à grande bouche. Régénération, katanas et quatrième mur en option.",
		RequestMessage: "Salut les modos ! Je suis techniquement un héros. Enfin, un anti-héros. Enfin, je suis gentil le mardi.",
		Status:         "rejected",
		Rejection:      "Méthodes incompatibles avec la charte d'intervention (usage excessif de la force et commentaires adressés au public).",
	},
}

var citizens = []citizenSeed{
	{person: person{Key: "mary.jane.watson", FirstName: "Mary Jane", LastName: "Watson", Civility: "madame", BirthDate: "2001-04-12",
		Phone: "0612345678", Address: address{"18 rue des Martyrs", "75009", "Paris", "FR"}}, Newsletter: true},
	{person: person{Key: "may.parker", FirstName: "May", LastName: "Parker", Civility: "madame", BirthDate: "1965-03-03",
		Phone: "0623456789", Address: address{"4 place Bellecour", "69002", "Lyon", "FR"}}, Newsletter: true},
	{person: person{Key: "jonah.jameson", FirstName: "John Jonah", LastName: "Jameson", Civility: "monsieur", BirthDate: "1960-01-11",
		Phone: "0634567890", Address: address{"100 rue Réaumur", "75002", "Paris", "FR"}}},
	{person: person{Key: "pepper.potts", FirstName: "Virginia", LastName: "Potts", Civility: "madame", BirthDate: "1974-08-27",
		Address: address{"12 cours Mirabeau", "13100", "Aix-en-Provence", "FR"}}, Newsletter: true},
	{person: person{Key: "happy.hogan", FirstName: "Harold", LastName: "Hogan", Civility: "monsieur", BirthDate: "1966-10-19",
		Address: address{"35 quai des Chartrons", "33000", "Bordeaux", "FR"}}},
	{person: person{Key: "jane.foster", FirstName: "Jane", LastName: "Foster", Civility: "madame", BirthDate: "1983-06-29",
		Address: address{"118 route de Narbonne", "31062", "Toulouse", "FR"}}, Newsletter: true},
	{person: person{Key: "darcy.lewis", FirstName: "Darcy", LastName: "Lewis", Civility: "madame", BirthDate: "1990-02-14",
		Address: address{"7 rue de la Monnaie", "59000", "Lille", "FR"}}},
	{person: person{Key: "foggy.nelson", FirstName: "Franklin", LastName: "Nelson", Civility: "monsieur", BirthDate: "1982-05-09",
		Address: address{"22 rue Crébillon", "44000", "Nantes", "FR"}}},
	{person: person{Key: "karen.page", FirstName: "Karen", LastName: "Page", Civility: "madame", BirthDate: "1987-09-30",
		Address: address{"3 place Kléber", "67000", "Strasbourg", "FR"}}, Newsletter: true},
	{person: person{Key: "ned.leeds", FirstName: "Ned", LastName: "Leeds", Civility: "monsieur", BirthDate: "2001-11-05",
		Address: address{"52 promenade des Anglais", "06000", "Nice", "FR"}}},
	{person: person{Key: "betty.ross", FirstName: "Betty", LastName: "Ross", Civility: "madame", BirthDate: "1975-12-01",
		Address: address{"9 la Canebière", "13001", "Marseille", "FR"}}},
	{person: person{Key: "flash.thompson", FirstName: "Eugene", LastName: "Thompson", Civility: "monsieur", BirthDate: "2001-07-21"},
		Status: "pending_activation"},
	{person: person{Key: "justin.hammer", FirstName: "Justin", LastName: "Hammer", Civility: "monsieur", BirthDate: "1972-02-02"},
		Status: "deactivated"},
}

var villains = []villainSeed{
	{Alias: "Thanos", Sector: "Univers connu", Threat: "critique", Status: "en-liberte",
		Description: "Titan fou obsédé par l'équilibre de l'univers. A déjà rayé la moitié de toute vie d'un claquement de doigts.",
		Specialties: []string{"menace-cosmique", "attaque-surhumaine"}, Films: []string{"tt4154756", "tt4154796"}},
	{Alias: "Loki", FirstName: "Loki", LastName: "Laufeyson", Sector: "Asgard, New York", Threat: "eleve", Status: "en-liberte",
		Description: "Dieu de la malice, illusionniste et métamorphe. Ses alliances changent plus vite que ses costumes.",
		Specialties: []string{"prise-otages", "anomalie-mystique"}, Films: []string{"tt0848228", "tt0800369", "tt9140554"}},
	{Alias: "Red Skull", FirstName: "Johann", LastName: "Schmidt", Sector: "Europe", Threat: "eleve", Status: "en-liberte",
		Description: "Ancien chef de l'HYDRA, défiguré par un sérum expérimental. Réapparaît régulièrement là où on ne l'attend pas.",
		Specialties: []string{"sabotage", "trafic-armes"}, Films: []string{"tt0458339"}},
	{Alias: "Ultron", Sector: "Réseaux informatiques mondiaux", Threat: "critique", Status: "en-liberte",
		Description: "Intelligence artificielle conçue pour protéger la paix, arrivée à la conclusion que l'humanité était le problème.",
		Specialties: []string{"cyberattaque", "attaque-surhumaine"}, Films: []string{"tt2395427"}},
	{Alias: "Green Goblin", FirstName: "Norman", LastName: "Osborn", Sector: "New York", Threat: "eleve", Status: "emprisonne",
		Description: "Industriel d'Oscorp rendu instable par sa propre formule. Planeur, citrouilles explosives et rire caractéristique.",
		Specialties: []string{"attaque-surhumaine", "sabotage"}, Films: []string{"tt0145487", "tt10872600"}},
	{Alias: "Doctor Octopus", FirstName: "Otto", LastName: "Octavius", Sector: "New York", Threat: "eleve", Status: "emprisonne",
		Description: "Physicien nucléaire greffé à quatre bras mécaniques autonomes. Spécialiste des attaques de banques et de trains.",
		Specialties: []string{"braquage", "sabotage"}, Films: []string{"tt0316654", "tt10872600"}},
	{Alias: "Venom", FirstName: "Eddie", LastName: "Brock", Sector: "San Francisco", Threat: "modere", Status: "en-liberte",
		Description: "Journaliste lié à un symbiote extraterrestre. Se dit « protecteur létal ». La définition de « protecteur » reste à préciser.",
		Specialties: []string{"attaque-surhumaine"}, Films: []string{"tt1270797"}},
	{Alias: "Magneto", FirstName: "Max", LastName: "Eisenhardt", Sector: "Genosha", Threat: "critique", Status: "en-liberte",
		Description: "Maître du magnétisme, militant radical de la cause mutante. Capable de déplacer un pont entier.",
		Specialties: []string{"attaque-surhumaine", "sabotage"}, Films: []string{"tt0120903", "tt1877832"}},
	{Alias: "Doctor Doom", FirstName: "Victor", LastName: "von Doom", Sector: "Latvérie", Threat: "critique", Status: "en-liberte",
		Description: "Monarque de Latvérie, génie scientifique et sorcier. Bénéficie de l'immunité diplomatique, ce qui complique tout.",
		Specialties: []string{"cyberattaque", "anomalie-mystique"}, Films: []string{"tt0120667"}},
	{Alias: "Kingpin", FirstName: "Wilson", LastName: "Fisk", Sector: "Hell's Kitchen", Threat: "eleve", Status: "emprisonne",
		Description: "Caïd du crime organisé new-yorkais. Contrôle trafics, racket et une partie de la police.",
		Specialties: []string{"trafic-armes", "braquage", "enlevement"}, Films: []string{"tt3322312", "tt10160804"}},
	{Alias: "Hela", Sector: "Asgard", Threat: "critique", Status: "en-liberte",
		Description: "Déesse de la mort, aînée d'Odin. Invoque des armes à volonté et des armées de morts-vivants.",
		Specialties: []string{"anomalie-mystique", "attaque-surhumaine"}, Films: []string{"tt3501632"}},
	{Alias: "Killmonger", FirstName: "Erik", LastName: "Stevens", Sector: "Oakland, Wakanda", Threat: "eleve", Status: "emprisonne",
		Description: "Ancien soldat d'élite, prétendant au trône du Wakanda. Prêt à armer le monde entier avec du vibranium.",
		Specialties: []string{"trafic-armes", "braquage"}, Films: []string{"tt1825683"}},
	{Alias: "Vulture", FirstName: "Adrian", LastName: "Toomes", Sector: "Queens", Threat: "modere", Status: "emprisonne",
		Description: "Ferrailleur recyclant de la technologie chitauri. Vend des armes extraterrestres au marché noir.",
		Specialties: []string{"trafic-armes", "braquage"}, Films: []string{"tt2250912"}},
	{Alias: "Mysterio", FirstName: "Quentin", LastName: "Beck", Sector: "Europe", Threat: "modere", Status: "en-liberte",
		Description: "Ancien ingénieur de Stark Industries, maître des illusions par drones. Ne croyez rien de ce que vous voyez.",
		Specialties: []string{"cyberattaque", "sabotage"}, Films: []string{"tt6320628"}},
	{Alias: "Kang le Conquérant", FirstName: "Nathaniel", LastName: "Richards", Sector: "Royaume quantique", Threat: "critique", Status: "en-liberte",
		Description: "Conquérant venu du XXXIe siècle. Ses variants sont innombrables ; en arrêter un ne suffit jamais.",
		Specialties: []string{"anomalie-mystique", "menace-cosmique"}, Films: []string{"tt10954600", "tt9140554"}},
	{Alias: "Abomination", FirstName: "Emil", LastName: "Blonsky", Sector: "Harlem", Threat: "modere", Status: "emprisonne",
		Description: "Ancien commando transformé par une dose de sérum et de rayons gamma. Plus gros, plus colérique que Hulk.",
		Specialties: []string{"attaque-surhumaine"}, Films: []string{"tt0800080"}},
	{Alias: "Baron Zemo", FirstName: "Helmut", LastName: "Zemo", Sector: "Sokovie", Threat: "modere", Status: "en-liberte",
		Description: "Aristocrate sokovien, stratège patient. Préfère diviser les héros plutôt que les affronter.",
		Specialties: []string{"sabotage", "enlevement"}, Films: []string{"tt3498820", "tt9208876"}},
	{Alias: "Taskmaster", FirstName: "Antonia", LastName: "Dreykov", Sector: "Europe de l'Est", Threat: "modere", Status: "en-liberte",
		Description: "Copie instantanément le style de combat de ses adversaires. Ancienne arme de la Chambre Rouge.",
		Specialties: []string{"enlevement", "attaque-surhumaine"}, Films: []string{"tt3480822"}},
	{Alias: "Electro", FirstName: "Max", LastName: "Dillon", Sector: "New York", Threat: "eleve", Status: "emprisonne",
		Description: "Électricien devenu générateur vivant. Un réseau électrique entier peut tomber quand il est de mauvaise humeur.",
		Specialties: []string{"sabotage", "incendie"}, Films: []string{"tt10872600"}},
	{Alias: "Sandman", FirstName: "Flint", LastName: "Marko", Sector: "New York", Threat: "modere", Status: "emprisonne",
		Description: "Petit voleur dont le corps est fait de sable. Passe sous les portes des coffres-forts.",
		Specialties: []string{"braquage", "catastrophe"}, Films: []string{"tt10872600"}},
	{Alias: "Dormammu", Sector: "Dimension noire", Threat: "critique", Status: "en-liberte",
		Description: "Entité cosmique qui règne sur la Dimension noire. Négocie mal, mais finit par céder face aux boucles temporelles.",
		Specialties: []string{"anomalie-mystique", "menace-cosmique"}, Films: []string{"tt1211837"}},
	{Alias: "Whiplash", FirstName: "Ivan", LastName: "Vanko", Sector: "Monaco, Moscou", Threat: "modere", Status: "emprisonne",
		Description: "Physicien russe armé de fouets électrifiés alimentés par un réacteur ARC de sa conception.",
		Specialties: []string{"sabotage", "incendie"}, Films: []string{"tt1228705"}},
	{Alias: "Malekith", Sector: "Svartalfheim", Threat: "eleve", Status: "en-liberte",
		Description: "Seigneur des Elfes noirs, veut replonger l'univers dans les ténèbres grâce à l'Éther.",
		Specialties: []string{"menace-cosmique"}, Films: []string{"tt1981115"}},
	{Alias: "Ronan l'Accusateur", Sector: "Empire kree", Threat: "critique", Status: "emprisonne",
		Description: "Fanatique kree armé du Marteau universel. Ses vaisseaux sont aussi grands que ses colères.",
		Specialties: []string{"menace-cosmique", "attaque-surhumaine"}, Films: []string{"tt2015381", "tt4154664"}},
}

var films = []filmSeed{
	{"tt0371746", "Iron Man", 2008, "Captif d'un groupe terroriste, l'industriel Tony Stark construit une armure pour s'échapper et décide de mettre fin au commerce d'armes de sa société."},
	{"tt0800080", "L'Incroyable Hulk", 2008, "En fuite, Bruce Banner cherche un remède à sa condition tandis que l'armée lance à ses trousses un soldat devenu monstre."},
	{"tt1228705", "Iron Man 2", 2010, "Tony Stark refuse de céder son armure au gouvernement, pendant qu'un physicien russe prépare sa vengeance."},
	{"tt0800369", "Thor", 2011, "Banni d'Asgard par son père, le dieu du tonnerre doit prouver qu'il est digne de son marteau, sur Terre."},
	{"tt0458339", "Captain America : First Avenger", 2011, "1942 : un jeune homme chétif devient le premier super-soldat et affronte Red Skull, chef de l'HYDRA."},
	{"tt0848228", "Avengers", 2012, "Nick Fury réunit une équipe de héros pour contrer Loki et son armée chitauri au-dessus de New York."},
	{"tt1300854", "Iron Man 3", 2013, "Hanté par la bataille de New York, Tony Stark affronte le Mandarin, qui détruit sa vie pièce par pièce."},
	{"tt1981115", "Thor : Le Monde des ténèbres", 2013, "Thor s'allie à Loki pour empêcher Malekith de plonger les Neuf Royaumes dans l'obscurité."},
	{"tt1843866", "Captain America : Le Soldat de l'hiver", 2014, "Le S.H.I.E.L.D. est infiltré par l'HYDRA ; Steve Rogers affronte un mystérieux assassin venu de son passé."},
	{"tt2015381", "Les Gardiens de la Galaxie", 2014, "Un aventurier spatial et une bande de marginaux doivent empêcher Ronan de détruire la planète Xandar."},
	{"tt2395427", "Avengers : L'Ère d'Ultron", 2015, "Une intelligence artificielle créée par Tony Stark décide d'éradiquer l'humanité pour sauver le monde."},
	{"tt0478970", "Ant-Man", 2015, "Un cambrioleur reçoit une combinaison capable de le réduire à la taille d'une fourmi, et un casse à réaliser."},
	{"tt3498820", "Captain America : Civil War", 2016, "Les accords de Sokovie divisent les Avengers en deux camps, manipulés par le baron Zemo."},
	{"tt1211837", "Doctor Strange", 2016, "Un neurochirurgien brillant mais arrogant découvre les arts mystiques après un accident."},
	{"tt2250912", "Spider-Man : Homecoming", 2017, "Peter Parker tente de concilier lycée et carrière de super-héros face au Vautour."},
	{"tt3501632", "Thor : Ragnarok", 2017, "Prisonnier sur une planète de gladiateurs, Thor doit rejoindre Asgard avant que Hela ne la détruise."},
	{"tt1825683", "Black Panther", 2018, "Devenu roi du Wakanda, T'Challa voit son trône contesté par un ennemi venu de l'étranger."},
	{"tt4154756", "Avengers : Infinity War", 2018, "Thanos cherche à réunir les six Pierres d'infinité pour anéantir la moitié de l'univers."},
	{"tt5095030", "Ant-Man et la Guêpe", 2018, "Scott Lang et Hope van Dyne s'aventurent dans le royaume quantique pour y retrouver Janet."},
	{"tt4154664", "Captain Marvel", 2019, "Une guerrière kree découvre son passé terrien en pleine guerre entre Krees et Skrulls."},
	{"tt4154796", "Avengers : Endgame", 2019, "Les Avengers survivants tentent l'impossible pour annuler le claquement de doigts de Thanos."},
	{"tt6320628", "Spider-Man : Far From Home", 2019, "En voyage scolaire en Europe, Peter Parker croise la route d'un nouveau héros nommé Mysterio."},
	{"tt3480822", "Black Widow", 2021, "Natasha Romanoff affronte son passé et la Chambre Rouge, traquée par Taskmaster."},
	{"tt9376612", "Shang-Chi et la Légende des Dix Anneaux", 2021, "Un expert en arts martiaux doit affronter l'organisation de son propre père."},
	{"tt10872600", "Spider-Man : No Way Home", 2021, "Un sort raté de Doctor Strange ouvre le multivers et fait venir des ennemis d'autres réalités."},
	{"tt9419884", "Doctor Strange in the Multiverse of Madness", 2022, "Strange protège une jeune fille capable de voyager entre les univers, convoitée par la Sorcière rouge."},
	{"tt10954600", "Ant-Man et la Guêpe : Quantumania", 2023, "La famille Lang-Pym se retrouve piégée dans le royaume quantique face à Kang le Conquérant."},
	{"tt0120903", "X-Men", 2000, "Deux jeunes mutants rejoignent l'école du professeur Xavier, en guerre contre la Confrérie de Magneto."},
	{"tt1877832", "X-Men : Days of Future Past", 2014, "Wolverine remonte le temps pour empêcher l'événement qui condamnera mutants et humains."},
	{"tt3315342", "Logan", 2017, "En 2029, un Logan vieillissant protège une jeune mutante traquée par une organisation."},
	{"tt0145487", "Spider-Man", 2002, "Mordu par une araignée génétiquement modifiée, Peter Parker affronte le Bouffon vert."},
	{"tt0316654", "Spider-Man 2", 2004, "Peter Parker doute de sa double vie au moment où Doctor Octopus menace New York."},
	{"tt1270797", "Venom", 2018, "Le journaliste Eddie Brock fusionne avec un symbiote extraterrestre à l'appétit inquiétant."},
	{"tt0120667", "Les 4 Fantastiques", 2005, "Quatre scientifiques gagnent des pouvoirs après une tempête cosmique, comme leur rival Victor von Doom."},
	{"tt3322312", "Daredevil (série)", 2015, "Avocat aveugle le jour, Matt Murdock protège Hell's Kitchen la nuit face à Wilson Fisk."},
	{"tt9140560", "WandaVision (série)", 2021, "Wanda et Vision vivent une vie de banlieue idyllique… qui n'a rien de réel."},
	{"tt9208876", "Falcon et le Soldat de l'hiver (série)", 2021, "Sam Wilson et Bucky Barnes font équipe face à une organisation qui veut un monde sans frontières."},
	{"tt9140554", "Loki (série)", 2021, "Loki est arrêté par le Tribunal des variations anachroniques et doit réparer la chronologie."},
	{"tt10160804", "Hawkeye (série)", 2021, "Clint Barton et sa jeune admiratrice Kate Bishop affrontent la mafia new-yorkaise avant Noël."},
	{"tt10234724", "Moon Knight (série)", 2022, "Un employé de musée découvre qu'il partage son corps avec un mercenaire au service d'un dieu égyptien."},
	{"tt10857164", "Miss Marvel (série)", 2022, "Une lycéenne de Jersey City découvre que le bracelet de sa grand-mère lui donne des pouvoirs."},
}

var articles = []articleSeed{
	{
		Title: "Bienvenue sur Héros SOS", Slug: "bienvenue-sur-heros-sos", Published: true, DaysAgo: 60,
		Excerpt: "Une plateforme pour signaler un incident et mobiliser les super-héros les plus proches.",
		Content: `<p>Héros SOS met en relation les citoyens et les super-héros agréés. Un incident ? Déclarez-le en quelques secondes : les héros disponibles sont prévenus et s'organisent pour intervenir.</p>
<h2>Comment ça marche</h2>
<ol><li>Vous déclarez l'incident (type, lieu, description).</li><li>Un héros lance une intervention et constitue son équipe.</li><li>Une fois l'intervention terminée, vous pouvez noter les héros.</li></ol>
<p>Chaque héros est vérifié par nos modérateurs avant d'apparaître sur la plateforme.</p>`,
	},
	{
		Title: "Bilan de la bataille de la Défense", Slug: "bilan-bataille-de-la-defense", Published: true, DaysAgo: 34,
		Excerpt: "Ultron a tenté de prendre le contrôle des tours de la Défense. Retour sur une intervention exemplaire.",
		Content: `<p>Le 17 septembre, une copie d'Ultron a infiltré les systèmes de gestion de plusieurs tours du quartier d'affaires. <strong>Iron Man</strong>, <strong>Captain Marvel</strong> et <strong>Spider-Man</strong> ont coordonné l'évacuation de plus de 300 personnes.</p>
<blockquote>« Sans l'alerte lancée par une habitante, on serait arrivés dix minutes trop tard. » – Tony Stark</blockquote>
<p>Merci à tous les citoyens qui signalent les incidents rapidement : c'est votre réactivité qui sauve des vies.</p>`,
	},
	{
		Title: "Cinq réflexes face à une attaque surhumaine", Slug: "cinq-reflexes-attaque-surhumaine", Published: true, DaysAgo: 21,
		Excerpt: "Vous assistez à un affrontement entre un vilain et des héros ? Voici comment vous protéger.",
		Content: `<h2>Les bons réflexes</h2>
<ul><li><strong>Éloignez-vous</strong> : les débris volent plus loin qu'on ne le croit.</li><li><strong>Signalez l'incident</strong> sur Héros SOS, avec l'adresse la plus précise possible.</li><li>Ne filmez pas : un héros distrait est un héros en danger.</li><li>Suivez les consignes des héros et des secours.</li><li>Ne tentez jamais de raisonner un vilain, même s'il a l'air sympathique (Loki, on parle de toi).</li></ul>`,
	},
	{
		Title: "Nouveau : la vidéothèque des héros", Slug: "nouveau-videotheque-des-heros", Published: true, DaysAgo: 9,
		Excerpt: "Les héros peuvent désormais associer à leur profil les films qui racontent leurs exploits.",
		Content: `<p>Chaque profil de héros affiche maintenant ses films, validés par nos modérateurs. Les fiches des vilains listent aussi leurs apparitions.</p>
<p>Une idée d'amélioration ? Écrivez-nous via le <a href="https://heros-sos.local/contact">formulaire de contact</a>.</p>`,
	},
	{
		Title: "Le Bureau recrute des modérateurs", Slug: "le-bureau-recrute-des-moderateurs", Published: true, DaysAgo: 3,
		Excerpt: "Vous avez l'œil pour repérer un faux héros ? Rejoignez l'équipe de modération.",
		Content: `<p>Avec plus de vingt héros actifs et des dizaines d'incidents chaque mois, l'équipe du Bureau s'agrandit.</p>
<p>Profil recherché : sens du détail, discrétion, et capacité à dire non à Deadpool.</p>`,
	},
	{
		Title: "Brouillon : retour sur l'invasion kree", Slug: "brouillon-retour-invasion-kree", Published: false, DaysAgo: 1,
		Excerpt: "Article en cours de rédaction.",
		Content: `<p>À compléter avec le rapport de Captain Marvel.</p>`,
	},
}

// Avis laissés par les citoyens, choisis selon l'issue de l'intervention.
var successComments = []string{
	"Intervention rapide et efficace, merci infiniment !",
	"Arrivé en moins de cinq minutes. Impressionnant.",
	"Très professionnel, a pris le temps de rassurer les enfants.",
	"Je n'oublierai jamais. Merci d'avoir sauvé mon immeuble.",
	"Efficace, même si le parking a un peu souffert.",
	"Un vrai héros, humble et à l'écoute.",
	"",
}

var failureComments = []string{
	"Le vilain s'est échappé, mais l'équipe a fait de son mieux.",
	"Arrivé trop tard, dommage.",
	"Beaucoup de dégâts pour un résultat décevant.",
	"",
}

var heroReplies = []string{
	"Merci pour votre message, c'est pour ça qu'on fait ce métier.",
	"Content que tout le monde aille bien. Restez prudents !",
	"Merci ! Et désolé pour le parking.",
}
