# Etat activable EBICS cote client et cote serveur

## 1. Objet

Ce document cadre, d'un point de vue fonctionnel et ergonomique, la notion
d'objet EBICS:

- cree;
- partiellement exploitable;
- activable.

L'objectif est d'aider a concevoir une UI/CLI qui ne se limite pas a creer des
objets valides en base, mais qui indique aussi clairement si le perimetre est
reellement pret pour l'execution.

## 2. Principe general

Un objet EBICS ne doit pas etre considere comme "pret" au simple motif qu'il
existe en base. Il faut distinguer:

- `cree`: l'objet passe la validation CRUD;
- `exploitable partiel`: certaines actions de consultation ou de refresh sont
  possibles;
- `activable`: le runtime Gateway peut reellement executer les flux attendus.

Cette distinction vaut:

- cote client EBICS;
- cote serveur EBICS;
- cote RTN.

## 3. Reference canonique de selection du client

### 3.1 Regle cible

La reference canonique qui prevaut deja dans Gateway doit rester `ClientID`.

Cela implique:

- pour les transferts Gateway, la reference canonique reste `Transfer.ClientID`;
- pour les actions EBICS hors transfert (`admin`, `reporting`,
  `initialisation`, `gestion/rotation de cles`), la selection du client doit
  converger vers une reference explicite stable de type `ClientID`;
- pour `RTN`, la selection du client doit converger vers la meme reference,
  sans reposer sur l'hypothese "un seul client EBICS actif".

Regle d'entree recommandee:

- le contrat fonctionnel cible doit exposer `ClientID` pour les actions
  non payload;
- `clientName` peut exister comme alias ergonomique en CLI/UI, mais il doit
  etre resolu en `ClientID` avant execution;
- `EbicsSubscriberID` designe l'identite EBICS cible, pas le client Gateway a
  lui seul.

### 3.2 Ce qu'il faut eviter

L'implementation ne doit pas supposer:

- qu'il n'existe qu'un seul client `protocol=ebics`;
- qu'un `clientName` optionnel suffit comme mecanisme principal de
  selection;
- qu'un singleton global est acceptable pour les chemins non payload.

## 4. Cloisonnement fonctionnel attendu

Le multi-client EBICS n'est pas un cas marginal. Il doit etre considere comme
un besoin normal.

Exemples de cloisonnement legitimes:

- une banque A et une banque B;
- plusieurs partenaires EBICS pour une institution financiere;
- separation entre flux `payload`, `admin`, `reporting`, `RTN`;
- separation de politiques TLS, certificats, cles et contrats.

En consequence, l'UI/CLI doit permettre de comprendre et d'exprimer
explicitement:

- quel `ClientID` est utilise;
- quel `RemoteAccount` est utilise;
- quel `EbicsSubscriber` est utilise;
- quel perimetre est activable ou non.

### 4.1 Environnements

La notion de multi-environnement a l'interieur d'une meme instance Gateway
n'est pas retenue ici.

Hypothese de travail:

- une instance Gateway correspond a un environnement logique donne;
- si l'on veut separer `recette`, `preproduction`, `production`, on deploie
  plusieurs instances Gateway;
- le cloisonnement traite dans ce document est donc un cloisonnement
  `multi-banques / multi-partenaires / multi-perimetres fonctionnels`,
  pas un multi-environnement interne a une meme instance.

Cette position est alignee avec le fonctionnement actuel de Gateway et evite
d'introduire dans le chantier EBICS une notion transverse plus large que le
protocole lui-meme.

## 5. Etat activable cote client EBICS

### 5.1 Niveau 1 - Cree

Le minimum pour creer les objets est:

- un `EbicsHost`;
- un `EbicsSubscriber` rattache a ce host.

Ce niveau n'implique pas que les actions runtime soient possibles.

### 5.2 Niveau 2 - Pret pour refresh contractuel

Pour pouvoir recuperer `HEV`, `HPD`, `HKD`, `HTD`, `HAA`, il faut au minimum:

- un `Client` Gateway EBICS explicite;
- un `EbicsHost` avec:
  - `hostID`
  - `protocolVersion`
  - `transport=https`
  - une URL effective via `subscriber.transportURL`, `host.defaultBankURL` ou
    `client.protoConfig.endpointURL`
- un `EbicsSubscriber` cote client avec:
  - `partnerID`
  - `userID`
  - `remoteAccountID`
- un `RemoteAccount` et son `RemoteAgent`;
- les credentials TLS necessaires:
  - confiance TLS cote `RemoteAgent`
  - certificat client TLS cote `RemoteAccount` si mTLS requis
- les cles EBICS runtime actives necessaires au subscriber
  (`AUTHENTICATION`, `ENCRYPTION`, `SIGNATURE`) selon les ordres executes.

Ce niveau permet la recuperation d'informations contractuelles et techniques
complementaires pour completer l'activation.

### 5.3 Niveau 3 - Pret pour payload

Pour les flux payload `BTU/BTD/FUL/FDL`, il faut en plus:

- une `Rule` Gateway coherente;
- un `ClientID` explicite pour le transfert;
- un `RemoteAccountID` explicite pour le transfert;
- si necessaire, un `EbicsPayloadProfile`;
- si la verification banque est active, les cles banque validees cote host;
- les references contractuelles / BTF necessaires au flux vise.

### 5.4 Niveau 4 - Pret pour RTN

Pour le RTN entrant et l'auto-pull:

- tout le niveau "pret pour payload" doit etre disponible;
- un `EbicsRTNProvider` doit etre configure avec:
  - `transport=WSS`
  - `enabled=true`
  - `ebicsSubscriberID`
  - `configuration.endpoint`
  - une `autoPullPolicy`
- la resolution du client doit etre explicite et non ambigue;
- les profils payload et les regles necessaires au pull doivent etre connus.

### 5.5 Visibilite operateur attendue sur RTN

Pour qu'un provider RTN soit exploitable sans ambiguite, les surfaces REST/CLI
doivent rendre visibles au minimum:

- le `clientID` selectionne;
- le `clientName` resolu;
- un `activationStatus` lisible;
- un `activationReason` explicite quand le perimetre est bloque.

Exemples d'etats utiles:

- `READY_MANUAL`
- `READY_AUTO`
- `READY_AUTO_FILTERED`
- `BLOCKED`
- `DISABLED`
- `ERROR`

## 6. Etat activable cote serveur EBICS

### 6.1 Niveau 1 - Serveur payload activable

Le minimum serveur actuellement activable correspond au serveur payload.

Il faut:

- un `LocalAgent` EBICS actif avec `ProtoConfig` serveur valide;
- une adresse d'ecoute;
- les credentials TLS serveur attendus par Gateway;
- un `EbicsHost` cote banque, coherent et actif;
- un ou plusieurs `EbicsSubscriber` cote serveur, lies a un `LocalAccount`;
- les cles banque / provider necessaires au runtime serveur.

Ce niveau permet d'activer le serveur EBICS Gateway sur le perimetre payload
deja supporte.

### 6.2 Niveau 2 - Serveur bancaire complet

Le vrai mode "banque sur Waarp Gateway" va au-dela du payload. Il doit couvrir:

- ordres contractuels `HPD`, `HKD`, `HTD`, `HAA`;
- ordres d'initialisation et de gestion de cles;
- ordres de rotation de cles;
- ordres de reporting et de signature;
- RTN sortant vers les partenaires.

Ce niveau n'est pas encore implemente. Il correspond au chantier `P5`.

## 7. Donnees minimales a demander en UI/CLI

### 7.1 Client EBICS

Le parcours de configuration devrait permettre d'exprimer au minimum:

- le `Client` Gateway a utiliser;
- le `Host` EBICS;
- le `Subscriber` EBICS;
- le `RemoteAccount`;
- les credentials TLS;
- l'etat des cles EBICS runtime;
- l'etat des cles banque validees;
- l'etat RTN si RTN active.

### 7.2 Serveur EBICS

Le parcours de configuration devrait permettre d'exprimer au minimum:

- le `LocalAgent` serveur EBICS;
- le `Host` banque;
- les `Subscribers` serveur;
- les `LocalAccount` associes;
- les credentials TLS serveur;
- le niveau de capacite:
  - `payload ready`
  - `bank admin ready`
  - `outbound RTN ready`

## 8. Messages ergonomiques recommandes

L'interface ne devrait pas seulement afficher "valide / invalide". Elle devrait
afficher des etats de preparation utiles.

Exemples cote client:

- `Configuration creee`
- `Pret pour refresh contractuel`
- `Pret pour payload`
- `Pret pour RTN auto-pull`
- `Bloque: aucun ClientID explicite`
- `Bloque: aucun client RTN explicite configure`
- `Bloque: remote account manquant`
- `Bloque: credentials TLS incomplets`
- `Bloque: lifecycle EBICS actif manquant`
- `Bloque: cle banque validee manquante`

Exemples cote serveur:

- `Serveur payload activable`
- `Serveur payload actif`
- `Serveur bancaire complet non disponible`
- `RTN sortant non configure`
- `RTN sortant pret`

## 9. Conclusion

Le point important pour l'ergonomie est le suivant:

- on ne doit pas confondre "objet valide en base" et "perimetre activable";
- le multi-client EBICS doit etre traite comme un besoin normal;
- la selection du client doit converger vers la reference canonique Gateway
  `ClientID`;
- l'UI/CLI doit guider l'utilisateur jusqu'au bon niveau d'activation,
  different selon qu'il vise:
  - refresh contractuel;
  - payload;
  - RTN;
  - serveur payload;
  - serveur bancaire complet.
