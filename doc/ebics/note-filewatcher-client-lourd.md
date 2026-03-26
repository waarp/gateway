# Note - Filewatcher, client lourd et positionnement Gateway pour EBICS

## 1. Objet

Cette note clarifie un point de positionnement important:

- Gateway est d'abord une plateforme de rupture protocolaire et
  d'orchestration asynchrone;
- elle n'est pas, par nature, un client lourd de poste de travail;
- elle n'embarque pas nativement un filewatcher integre au coeur comparable a
  une logique d'agent local resident.

La question est donc:

- faut-il ajouter un filewatcher integre ou un client lourd pour servir EBICS ?

## 2. Constat sur l'existant

L'existant montre:

- un positionnement majoritairement asynchrone du moteur Gateway;
- l'existence d'utilitaires satellites comme `get-remote`;
- la presence historique d'une configuration `filewatcher` dans la chaine
  `updateconf`, ce qui confirme l'existence d'un outillage annexe plutot que
  d'un sous-systeme coeur fusionne.

Il existe aussi un mode `synchronous` dans certaines taches de transfert
Gateway.

Cela ne change toutefois pas la nature du produit:

- ce mode sert a enchainer des traitements dans un contexte d'execution;
- il ne transforme pas Gateway en client lourd ni en moteur principal de
  surveillance locale de repertoires.

## 3. Lecture produit recommandee

Pour EBICS, il ne faut pas melanger:

- la collecte protocolaire bancaire;
- le declenchement local par surveillance de repertoire;
- l'ergonomie poste utilisateur d'un client lourd.

La bonne lecture est:

- EBICS dans Gateway apporte la couche protocolaire, la securite, la
  tracabilite et le passe-plat;
- la surveillance locale de fichiers et les usages poste de travail relevent
  plutot d'outils adjacents ou d'integrations metier.

## 4. Decision recommandee

Recommendation de phase 1:

- ne pas ajouter de filewatcher integre au coeur Gateway pour justifier EBICS;
- ne pas ajouter de client lourd dans le perimetre EBICS/Gateway;
- capitaliser d'abord sur:
  - REST
  - CLI
  - filesystem passif
  - `AMQP 0.9.1`
  - `AMQP 1.0`

Motifs:

- cela reste coherent avec le positionnement asynchrone de Gateway;
- cela evite de diluer le produit vers un autre metier;
- cela reduit fortement le cout UI/UX, packaging et support;
- cela concentre l'effort sur la valeur la plus forte: protocole et
  intermediation fiable.

## 5. Quand un filewatcher deviendrait utile

Un filewatcher pourrait devenir pertinent plus tard si:

- un nombre significatif de clients ont un besoin de depot local automatique
  sans middleware intermediaire;
- le couple `filesystem + watcher` apporte une vraie valeur commerciale
  recurrente;
- on veut completer l'offre d'integration pour des SI peu equipes en API ou en
  messaging.

Dans ce cas, la bonne approche serait:

- un composant additionnel;
- ou un utilitaire compagnon;
- pas un glissement non maitrise du coeur Gateway.

## 6. Quand un client lourd deviendrait utile

Un client lourd ne deviendrait pertinent que pour un autre segment de besoin:

- interactions humaines locales;
- ergonomie poste utilisateur;
- pilotage manuel de fichiers et de signatures;
- usage de proximite hors logique serveur MFT.

Ce besoin ne correspond pas au centre de gravite retenu ici.

## 7. Conclusion

Pour l'integration EBICS dans Gateway:

- `non` a un filewatcher integre comme prerequis;
- `non` a un client lourd dans le perimetre;
- `oui` a des connecteurs asynchrones solides et a un passe-plat bien pense;
- `oui` eventuellement plus tard a des outils compagnons si un vrai besoin
  marche le justifie.

La bonne priorite reste donc:

- protocole EBICS;
- `EbicsOperation`;
- socle AMQP;
- administration propre;
- correlation et observabilite.
