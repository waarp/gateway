# Gateway en role banque EBICS

## 1. Objet

Ce document cadre le perimetre cible quand Waarp Gateway joue le role de
serveur bancaire EBICS, interface avec une application metier interne.

Objectif:

- ne plus limiter le serveur EBICS Gateway aux seuls ordres payload
  `BTU` / `BTD`;
- expliciter les familles d'ordres non payload a servir cote banque;
- cadrer la source des donnees retournees;
- cadrer le RTN sortant vers les partenaires;
- rester aligne sur la philosophie Waarp Gateway:
  un coeur protocolaire borne, des modeles explicites, une observabilite
  forte, et aucun raccourci metier cache dans le runtime protocolaire.

## 2. Positionnement architectural

Quand Gateway est utilisee cote banque:

- le protocole EBICS reste une frontiere technique;
- les donnees metier internes ne doivent pas etre construites ad hoc dans les
  handlers EBICS;
- les reponses serveur doivent etre alimentees par des modeles internes
  explicites, persistants, administrables et observables;
- le RTN sortant doit etre traite comme un composant de notification
  protocolaire, pas comme un bus metier generique.

Consequence:

- les ordres serveur non payload doivent etre relies a des projections internes
  stables;
- les ordres payload `BTU` / `BTD` continuent de s'appuyer sur le moteur
  Gateway standard des transferts;
- les workflows sensibles `initialisation`, `key management`, `reporting`,
  `signature` et `VEU` ne doivent pas etre confondus avec des transferts.

## 3. Priorisation fonctionnelle

### Priorite 1

- `HPD`
- `HKD`
- `HTD`
- `HAA`
- RTN sortant minimal vers les partenaires

But:

- rendre Gateway credible comme serveur bancaire minimum exploitable;
- publier les capacites, permissions, vues contractuelles et services/BTF
  disponibles;
- notifier qu'un document ou ordre recuperable est disponible.

### Priorite 2

- ordres d'initialisation et de gestion de cles serveur
- ordres de rotation de cles serveur

But:

- permettre le cycle de vie complet des partenaires cote banque;
- permettre a Gateway de tenir un vrai role d'hebergeur EBICS.

### Priorite 3

- ordres de reporting serveur
- ordres de signature serveur

But:

- completer le role banque sur les consultations et interactions sensibles;
- preparer ensuite le workflow VEU metier.

## 4. Perimetre fonctionnel par famille

### 4.1 Ordres contractuels

Ordres:

- `HPD`
- `HKD`
- `HTD`
- `HAA`

Attendus cote serveur:

- `HPD` publie les capacites protocolaire et options supportees par le host;
- `HKD` publie la vue des permissions et informations de cles visibles par le
  partenaire;
- `HTD` publie la vue des permissions et capacites de transfert visibles par
  le partenaire;
- `HAA` publie les ordres / BTF / services recuperables cote partenaire.

Bornes fonctionnelles minimales:

- `HPD` et `HAA` doivent rester des projections portees par le host bancaire;
- `HKD` et `HTD` doivent rester des projections bornees par subscriber
  `partnerID/userID`;
- `HKD` ne doit pas servir par defaut les memes elements que `HTD`:
  meme si `lib-ebics` partage une interface provider proche,
  Gateway doit conserver une distinction fonctionnelle nette entre
  permissions/cles visibles cote partenaire et capacites de transfert
  visibles cote utilisateur;
- les projections doivent rester explicites et requetables hors runtime
  HTTP, pour que l'operateur puisse controler ce qui sera servi.

Source de donnees attendue:

- projection interne explicite, pas calcul opportuniste au fil de l'eau;
- versionnee, observable, historisable si necessaire;
- alignee avec les modeles de contrats, catalogues et policies internes.

### 4.2 Initialisation et gestion de cles

Ordres cibles minimaux:

- `INI`
- `HIA`
- `H3K`
- `HPB`

Attendus cote serveur:

- accepter et verifier les demandes des partenaires;
- persister les etats et preuves associes;
- permettre une validation operateur ou metier lorsque la spec ou la politique
  interne l'impose;
- separer strictement:
  reception protocolaire,
  decision interne,
  activation effective.

### 4.3 Rotation de cles

Ordres cibles:

- `PUB`
- `HCA`
- `HCS`
- `HSA`
- `SPR`

Attendus cote serveur:

- lifecycle borne des demandes;
- non-regression forte sur les etats de cles actives / pending / retirees;
- evidence operateur et technique exploitable;
- aucune suppression destructive silencieuse.

### 4.4 Reporting et signature

Ordres cibles:

- `HAC`
- `HVD`
- `HVU`
- `HVZ`
- `HVT`
- `HVE`
- `HVS`

Attendus cote serveur:

- servir les vues de reporting a partir d'un historique interne stable;
- distinguer les reponses purement techniques des decisions metier;
- ne pas annoncer un workflow VEU complet tant qu'il n'existe pas.

## 5. RTN sortant cote banque

Le RTN sortant doit permettre a Gateway, en role banque, de notifier qu'un:

- document est disponible au pull;
- ordre ou action attend un traitement partenaire;
- changement d'etat important est survenu.

Le composant devra couvrir:

- provider(s) de notification sortante administres;
- correlation `partner / subscriber / ordre / payload / event`;
- retries bornes;
- quarantaine / reemission;
- observabilite REST/CLI;
- distinction claire entre notification technique et decision metier.

Le RTN sortant ne remplace pas:

- le passe-plat metier;
- AMQP;
- le futur workflow VEU.

## 6. Regles de conception a respecter

- ne pas surcharger `TransferInfo`;
- ne pas contourner le moteur Gateway quand un vrai `Transfer` existe;
- ne pas inventer un second moteur d'historique si un historique dedie existe
  deja pour la famille concernee;
- ne pas melanger dans un meme objet:
  donnees metier internes,
  projection protocolaire EBICS,
  etat runtime d'execution;
- privilegier des tables/projections explicites plutot que du JSON libre.

## 7. Decoupage recommande du chantier

1. `P5B`
Serveur contractuel `HPD/HKD/HTD/HAA`
Etat:
- ferme au 2026-04-03
- implementation branchee sur les handlers `lib-ebics` natifs
- projection serveur initiale issue de `EbicsHost`, `EbicsSubscriber`,
  `EbicsContractViewItem`

2. `P5C`
Projection interne des contrats, services et permissions
Etat:
- ferme au 2026-04-03
- projection serveur dediee stockee dans
  `EbicsServerContractSet` / `EbicsServerContractItem`
- bornage fonctionnel explicite:
  `HPD/HAA` scopes host, `HKD/HTD` scopes subscriber
- observabilite REST/CLI minimale disponible via
  `ebics server-contract-set list/get`

3. `P5D`
Ordres serveur non payload hors contrats

4. `P5E`
RTN sortant

5. `P5F`
Observabilite, securite, non-regression

## 8. Hors perimetre immediat

Ne pas considerer comme inclus dans `P5A`:

- le passe-plat metier asynchrone;
- `AMQP 0.9.1` / `AMQP 1.0`;
- le workflow VEU complet;
- le multi-environnement dans une meme instance Gateway.

Ces sujets restent relies, mais doivent etre portes par leurs chantiers
dedies.
