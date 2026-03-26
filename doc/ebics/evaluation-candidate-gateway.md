# Evaluation de Gateway comme candidat pour EBICS

## 1. Objet

Ce document evalue si Waarp Gateway est un bon candidat pour une
implementation EBICS, a partir:

- de l'existant Gateway;
- du perimetre cible retenu;
- d'une lecture de la specification officielle EBICS 3.0.2.

La question n'est pas seulement "est-ce possible ?", mais aussi:

- le decouplage protocole / metier est-il coherent avec la norme;
- ce decouplage rendra-t-il effectivement service aux futurs clients.

## 2. Points normatifs utiles a la decision

### 2.1 Ce qui releve clairement du moteur protocolaire

La specification EBICS 3.0.2 formalise tres clairement des responsabilites
techniques qui cadrent bien avec Gateway:

- gestion des transactions EBICS, de leurs phases et de leur reprise
  (`chapitre 5`, en particulier pages 80 a 81 et 118 a 139);
- protection anti-rejeu par `Nonce` / `Timestamp`, avec stockage local cote
  serveur (`chapitre 11.4`, pages 234 a 236);
- gestion des ordres administratifs et de leurs retours normalises;
- gestion du customer acknowledgement `HAC`, explicitement oriente vers
  l'evaluation automatique cote client (`chapitre 10.1`, page 215).

Conclusion:

- ces sujets sont pleinement du ressort d'une plateforme protocolaire comme
  Gateway.

### 2.2 Ce qui releve du cycle de vie cryptographique

La specification distingue bien:

- l'initialisation des abonnes, avec lettres `INI` / `HIA`
  (`chapitre 4.4` et `chapitre 11.5`, pages 237 a 238);
- les changements de cles abonnes `PUB`, `HCA`, `HCS`, qui ne necessitent pas
  de lettres d'initialisation (`chapitre 4.6.1`, page 70);
- la mise a jour des cles banque, dont la duree de validite et la politique de
  rotation ne font pas partie de l'interface EBICS elle-meme
  (`chapitre 4.6.2`, page 76).

La specification dit aussi explicitement qu'en cas de changement de cles via
`PUB` / `HCA` / `HCS`, la tracabilite documentaire reste a la charge de
l'etablissement bancaire (`page 70`).

Conclusion:

- l'automatisation de rotation de cles est pertinente dans Gateway;
- il faut en revanche distinguer soigneusement le protocole EBICS de la
  gouvernance PKI et des decisions de conformite.

### 2.3 Ce que dit la norme sur les signatures distribuees

La partie `EDS` est centrale pour ta decision.

La specification indique que:

- le support des ordres administratifs EDS (`HVU`, `HVZ`, `HVD`, `HVT`, `HVE`,
  `HVS`) est obligatoire pour une implementation bancaire conforme
  (`chapitre 8`, page 148);
- le processus EDS permet a plusieurs abonnes, potentiellement de clients
  differents, de signer de maniere distribuee, dans le temps et dans l'espace
  (`page 148`);
- une partie des informations necessaires a la signature peut etre obtenue en
  dehors d'EBICS, par un canal alternatif de communication (`pages 149 a 151`);
- l'autorisation finale depend de regles locales de signatures et de classes de
  signatures, deposees cote banque (`pages 148 a 150`, `chapitre 11.2.3`,
  page 230).

Conclusion:

- la norme n'impose pas qu'un client EBICS porte lui-meme tout le workflow
  humain de signature;
- elle impose surtout la capacite protocolaire de:
  - voir l'etat,
  - recuperer les informations utiles,
  - ajouter une signature,
  - annuler un ordre dans l'EDS.

Le decouplage propose est donc coherent:

- Gateway porte le protocole de signature distribuee;
- l'application metier peut porter l'orchestration humaine et les decisions.

### 2.4 Ce que dit la norme sur la frontiere protocole / traitement metier

La specification est utile ici car elle separe explicitement plusieurs couches.

Au `chapitre 5.1.3.1` (`pages 80 a 81`), elle indique que:

- le pre-traitement, le stockage intermediaire et la gestion des ordres en
  attente dependent de l'implementation de la banque;
- ces mecanismes ne sont pas eux-memes une composante EBICS;
- seuls certains ordres administratifs doivent etre traites completement dans
  la transaction EBICS.

Conclusion:

- la norme ne demande pas qu'un produit EBICS embarque l'integralite du
  workflow metier bancaire;
- elle laisse clairement de la place a une architecture en couches.

### 2.4.1 Cas particulier du "contrat technique" publie par la banque

La specification montre qu'une partie du contrat entre banque et client est
projetee dans le protocole lui-meme:

- `HPD` publie les capacites et options supportees (`pages 193 a 194`, `240 a
  241`);
- `HKD` / `HTD` publient les permissions visibles cote EBICS, y compris
  `AdminOrderType`, `Service`, `AuthorisationLevel`, `AccountID`,
  `MaxAmount` (`pages 199 a 212`);
- `HAA` publie les services / BTF telechargeables (`pages 193`, `240`);
- le chapitre sur l'interpretation des BTF indique explicitement que les
  autorisations dependent du contrat banque mais qu'elles ont une projection
  exploitable dans EBICS (`pages 35 a 36`).

Conclusion:

- Gateway doit stocker cette projection technique du contrat;
- l'application metier peut stocker en plus sa propre politique fonctionnelle;
- la source d'autorite juridique ou commerciale reste hors Gateway.

### 2.5 Ce que cela implique pour RTN

Le document principal EBICS 3.0.2 ne traite pas RTN comme chemin coeur
request/response.

Dans le materiel WAARP associe a la librairie, RTN est traite comme:

- une extension optionnelle;
- un canal de signalisation hors bande;
- un mecanisme declenchant ensuite des ordres EBICS standards de type pull.

Conclusion:

- architecturer RTN comme une capacite transverse additionnelle dans Gateway
  est coherent;
- il n'y a pas de raison de melanger RTN avec le coeur transactionnel EBICS.

### 2.6 Autres points de frontiere identifies

La lecture de la specification fait ressortir quelques sujets proches de cette
frontiere:

- `PreValidation`:
  Gateway doit connaitre si la banque le supporte et peut l'utiliser, mais les
  controles de compte, de limite et d'autorisation restent fondamentalement du
  ressort banque / metier (`pages 27`, `83 a 84`, `241`);
- `technical subscriber / SystemID`:
  c'est clairement un concept protocolaire et non metier (`pages 27 a 28`);
- `NumSigRequired`, `AuthorisationLevel`, `MaxAmount`:
  ce sont des faits techniques publies par EBICS, utiles pour prevenir et
  expliquer, mais ils ne remplacent pas un workflow metier interne
  (`pages 208 a 212`);
- interpretation des payloads:
  rien dans la norme n'impose de la faire dans la couche protocolaire.

## 3. Pertinence du decouplage propose

## 3.1 Decouplage recommande

Le decouplage suivant est pertinent:

- `Gateway = moteur protocolaire EBICS + automate technique`
- `Application metier = moteur de decision et d'orchestration humaine`

Cela couvre bien le besoin cible:

- rotation des cles relevant du protocole;
- recuperation automatisee des reports;
- notifications RTN et auto-pull;
- verification et collecte des signatures;
- passe-plat vers le SI metier pour les decisions non protocolaires.

## 3.2 Pourquoi ce decouplage rendra service

Si les futurs clients disposent deja d'applications metier consommant les
payloads bancaires, ce decouplage a plusieurs avantages:

- il evite de dupliquer dans Gateway des workflows metier deja existants;
- il reduit le risque de construire une "demi-application bancaire" peu
  adaptable;
- il garde Gateway dans son point fort: robustesse d'exploitation,
  standardisation protocolaire, tracabilite et securite;
- il permet a des clients differents de brancher des logiques metier
  differentes sans forker l'implementation EBICS.

## 3.3 Point d'attention

Ce decouplage est pertinent seulement si Gateway fournit quand meme une vraie
valeur technique.

Cela suppose au minimum:

- un historique technique exploitable;
- une API claire pour les etats EDS, HAC, RTN et les rotations;
- une outbox ou publication d'evenements fiable vers le metier;
- des outils operateur minimaux pour diagnostiquer et rejouer.

Sans cela, Gateway ne serait qu'un simple proxy EBICS et la valeur du
decouplage serait trop faible.

### 3.4 Effet structurant d'AMQP

L'ajout de `AMQP 0.9.1` et `AMQP 1.0` comme protocoles Gateway a part entiere
renforce fortement la credibilite de l'option Gateway.

Pourquoi:

- cela donne a Gateway un role d'integration plus large que le seul transport
  fichier;
- cela rend le passe-plat metier plus robuste et plus industriel;
- cela prepare un socle de decouplage utile avant meme EBICS;
- cela montre que Gateway n'est pas seulement "un adaptateur EBICS", mais une
  plateforme d'echanges et d'orchestration.

Conclusion:

- oui, AMQP peut etre considere comme un prealable pertinent a EBICS;
- meme si ce n'est pas un prealable strictement obligatoire, c'est un
  prealable d'architecture produit tres defendable.

## 4. Grille de decision

### 4.1 Adequation au protocole

Verdict: `forte`

Pourquoi:

- Gateway sait deja heberger plusieurs protocoles;
- la norme EBICS demande beaucoup de discipline transactionnelle, de
  persistance technique et de tracabilite;
- ce sont des sujets naturels pour Gateway.

### 4.2 Adequation a la persistance

Verdict: `moyenne a forte`

Pourquoi:

- il faut ajouter plusieurs objets durables specifiques EBICS;
- mais ils restent majoritairement techniques:
  `transactions`, `nonces`, `bank keys`, `subscribers`, `rtn events`,
  `event outbox`.

Le point de vigilance principal reste d'eviter de tordre `Transfer`.

### 4.3 Adequation a l'administration

Verdict: `forte`

Pourquoi:

- l'existant Gateway REST/CLI/UI donne deja une base solide;
- EBICS a besoin d'administration et d'exploitation continue;
- la valeur de Gateway est reelle sur cet aspect.

### 4.4 Risque d'intrusion metier

Verdict: `maitrisable sous condition`

Condition:

- ne pas reimplementer le workflow metier de signature;
- ne pas reimplementer les circuits de validation bancaire;
- ne pas transformer l'initialisation en outil de pilotage humain complet.

### 4.5 Valeur reelle de Gateway sur ce perimetre

Verdict: `forte`

Parce que Gateway apporte:

- une execution protocolisee stable;
- un cadre d'exploitation;
- une persistance durable;
- un point d'integration unique vers le SI metier.

## 5. Decision provisoire

Avec le perimetre recentre que tu proposes, Gateway est un `bon candidat`
pour une implementation EBICS.

La conclusion la plus defendable a ce stade est:

- `GO Gateway`, sous reserve de tenir strictement la frontiere
  protocole / metier.

Le `NO-GO` redeviendrait plausible seulement si l'un des besoins suivants
devenait central:

- pilotage humain complet du workflow de signatures;
- logique bancaire metier embarquee dans Gateway;
- orchestration metier lourde autour des decisions de validation;
- forte variabilite fonctionnelle par client qui depasse le cadre protocolaire.

## 6. Recommandation pratique

La bonne strategie n'est donc ni un `from scratch` immediat, ni un codage trop
rapide dans Gateway.

La bonne suite est:

1. verrouiller cette frontiere protocole / metier;
2. valider les lots 1 et 2 contre cette frontiere;
3. ne produire un squelette de code que si la preuve reste propre.
