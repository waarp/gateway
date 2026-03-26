# Architecture logicielle cible

## 1. Principes d'architecture

L'architecture cible repose sur cinq principes:

- EBICS est un protocole natif supplementaire de Gateway;
- `AMQP 0.9.1` et `AMQP 1.0` peuvent etre introduits comme protocoles natifs
  supplementaires, independants d'EBICS;
- la librairie WAARP EBICS est consommee comme moteur protocolaire, pas comme
  application autonome;
- les services, la persistance, l'administration et l'observabilite restent
  uniformes avec le reste de Gateway;
- les traitements fichier EBICS convergent vers le pipeline Gateway.
- les decisions non protocolaires restent hors Gateway.

## 2. Vue d'ensemble

```text
Administrateur / API REST / CLI / UI
                |
                v
        Couche administration Gateway
                |
                v
        Modele Gateway + services coeur
                |
        +-------+-----------------------------+
        |                                     |
        v                                     v
 Module protocolaire EBICS       AMQP 0.9.1 / AMQP 1.0       Autres protocoles Gateway
        |
        +-------------------------------+-------------------+
        |                               |                   |
        v                               v                   v
 Service serveur EBICS           Service client EBICS   Vue contrat technique
        |                               |               banque / permissions
        |                               +-------------------+
        |                                                   |
        v                                                   v
 Workflow initialisation                         Ingestion RTN / Auto-pull
 et rotation de cles                                      |
        |                                                  |
        +---------------+---------------+
                        |
                        v
             Adaptateurs Gateway <-> lib EBICS
                        |
        +---------------+------------------------------+
        |               |              |               |
        v               v              v               v
   KeyStore       SubscriberStore    TxStore      NonceStore
                        |
                        v
                 Base de donnees Gateway
                        |
                        v
             Pipeline / Tasks / Historique
                        |
                        v
              Passe-plat vers application metier
           (FS / REST / CLI / AMQP 0.9.1 / AMQP 1.0)
```

## 3. Composants cibles

### 3.1 Module `pkg/protocols/modules/ebics`

Responsabilites:

- implementer l'interface `protocols.Module`;
- fournir les services serveur et client;
- definir les structures de `ProtoConfig`;
- enregistrer le protocole dans Gateway.

Sous-composants proposes:

- `module.go`
- `server.go`
- `client.go`
- `config.go`
- `stores/`
- `mapping/`
- `orders/`
- `observability/`

### 3.2 Service serveur EBICS

Responsabilites:

- charger la configuration du `LocalAgent`;
- construire le serveur EBICS via `provider/server`;
- exposer le handler HTTP EBICS sur l'adresse du serveur local;
- raccorder TLS, mTLS, XSD, stores et observer;
- transformer les ordres fichier en executions Gateway.

### 3.3 Service client EBICS

Responsabilites:

- charger la configuration du `Client`;
- construire un profil client EBICS durci;
- executer les ordres vers un `RemoteAgent` / `RemoteAccount`;
- synchroniser les resultats avec `Transfer` et `TransferInfo` quand pertinent.

### 3.4 Couche d'adaptation des stores

Responsabilites:

- implementer les interfaces de la librairie EBICS sur la base SQL Gateway;
- isoler le schema EBICS du reste du protocole;
- garantir la durabilite des transactions et nonces.

Interfaces a couvrir:

- `store.KeyStore`
- `store.SubscriberStore`
- `store.TxStore`
- `store.NonceStore`

### 3.5 Couche de mapping metier

Responsabilites:

- projeter les objets Gateway sur les identites EBICS;
- convertir un ordre EBICS en execution Gateway;
- faire le lien entre transaction EBICS et transfert Gateway;
- normaliser les logs et statuts.

Le terme "metier" ici doit etre compris comme "mapping de donnees et
orchestration technique", pas comme "prise de decision metier".

### 3.6 Workflow d'initialisation et de rotation

Responsabilites:

- orchestrer les phases techniques `INI` / `HIA` / `H3K` et `HPB`;
- generer les lettres EBICS a partir des donnees stockees;
- materialiser les ruptures d'automatisme et les validations operateur;
- piloter la rotation des cles et certificats.

### 3.7 Service d'ingestion RTN

Responsabilites:

- consommer un canal d'evenements temps reel distinct du coeur EBICS;
- parser et valider les evenements RTN;
- assurer idempotence, rejeu controle et observabilite;
- convertir les notifications en declenchements de pulls standards.

### 3.8 Couche de passe-plat vers l'application metier

Responsabilites:

- exposer les informations techniques EBICS a une application tierce;
- publier ou pousser les evenements techniques necessaires;
- recevoir en retour des instructions non protocolaires deja decidees;
- ne pas embarquer la logique de decision elle-meme.

Principe:

- la couche de passe-plat doit s'appuyer sur des connecteurs standards;
- lorsque ces connecteurs portent un vrai cycle de vie protocolaire, ils
  gagnent a etre implementes comme protocoles Gateway a part entiere.

Connecteurs / protocoles cibles:

- filesystem;
- REST;
- CLI;
- `AMQP 0.9.1`;
- `AMQP 1.0`.

### 3.9 Vue technique du contrat banque

Responsabilites:

- collecter `HPD`, `HKD`, `HTD`, `HAA`;
- construire une projection technique coherente des capacites et permissions;
- fournir un controle preventif des ordres et BTF;
- historiser la provenance et la date de rafraichissement.

## 4. Raccordement avec l'existant Gateway

### 4.1 Demarrage et arret

Le service principal `WG` demarre deja tous les serveurs et clients selon leur
protocole.

Impact cible:

- aucun mecanisme special n'est necessaire;
- un `LocalAgent` ou un `Client` avec protocole `ebics` doit etre demarre comme
  les autres;
- l'arret doit utiliser le meme cycle de vie que les protocoles existants.

### 4.2 Administration

L'administration doit reutiliser la filiere existante:

- modeles;
- routes REST;
- commandes CLI;
- ecrans UI.

Le principe est d'etendre les ressources existantes avant de creer de
nouvelles familles d'API.

### 4.3 Execution

Pour les flux fichier:

- EBICS initialise ou consomme une transaction protocolaire;
- Gateway convertit cette transaction en `Transfer`;
- le `Pipeline` orchestre stockage, tasks et historisation;
- la correlation structurelle avec l'operation EBICS doit etre portee d'abord
  par la persistance dediee, pas seulement par `TransferInfo`.

Pour les flux non fichier:

- l'execution reste dans le service EBICS;
- le resultat est historise dans une journalisation EBICS dediee;
- un lien vers `Transfer` n'est cree que si un fichier est reellement manipule.

L'architecture cible materialise cette separation via une notion dediee
`EbicsOperation` pour les ordres et traitements EBICS non payload.

### 4.4 Initialisation et lettre EBICS

L'architecture doit traiter l'initialisation comme un sous-processus hybride.

Decoupage:

1. sous-phase protocolaire automatisee:
   `INI`, `HIA`, `H3K`, `HPB`
2. sous-phase operateur/hors bande:
   generation de lettre, transmission, attente de validation banque
3. sous-phase d'activation:
   passage de l'abonne en etat productif

Le point cle est de ne pas melanger succes technique de l'ordre et activation
reelle de l'abonne.

### 4.5 Rotation de cles

La rotation doit etre exposee comme un workflow metier technique transverse.

Principes:

- preparation du futur materiel;
- emission des ordres de mise a jour;
- attente de validation externe si necessaire;
- bascule controlee;
- retrait de l'ancien materiel.

### 4.6 Notifications RTN

RTN doit etre place en bordure de l'architecture, comme source de
declenchement.

Principes:

- RTN n'est pas le protocole de transfert lui-meme;
- RTN alimente le client EBICS avec un plan d'actions;
- l'execution resultant des notifications reste tracee dans Gateway.
- la premiere implementation de transport RTN visee dans Gateway est un canal
  `WebSocket/WSS`, tout en gardant l'architecture RTN decouplee du transport.

## 5. Modelisation logique cible

### 5.1 Niveau Gateway

Objets conserves:

- `LocalAgent`
- `Client`
- `RemoteAgent`
- `LocalAccount`
- `RemoteAccount`
- `Credential`
- `Rule`
- `Transfer`

### 5.2 Niveau EBICS dedie

Objets ajoutes:

- `EbicsHost`
- `EbicsSubscriber`
- `EbicsBankKey`
- `EbicsTransaction`
- `EbicsTransactionSegment`
- `EbicsNonceEntry`
- `EbicsKeyLifecycle`
- `EbicsInitializationWorkflow`
- `EbicsRTNEvent`
- `EbicsContractView`
- eventuellement `EbicsOrderHistory`

Principe:

- le modele Gateway porte les objets d'exploitation communs;
- le modele EBICS porte les invariants protocolaires durables.

## 6. Flux cibles

### 6.1 Flux serveur EBICS entrant

1. Le service EBICS recoit une requete HTTP.
2. La librairie EBICS valide transport, syntaxe, XSD, signature et anti-rejeu.
3. Le routeur d'ordre determine le traitement.
4. Si l'ordre concerne un fichier, un `Transfer` Gateway est cree ou enrichi.
5. Le pipeline Gateway execute les traitements.
6. Les statuts et historiques sont synchronises.
7. La reponse EBICS est retournee.

### 6.2 Flux client EBICS sortant

1. Une demande REST/CLI ou une planification cree un contexte d'execution.
2. Le service client EBICS charge identites, credentials et politique TLS.
3. La librairie EBICS construit et execute l'ordre.
4. Le resultat est mappe:
   - vers `Transfer` si le flux manipule un fichier;
   - vers l'historique EBICS sinon.
5. Les post-traitements Gateway sont lances si applicable.

### 6.3 Flux d'initialisation avec lettre

1. L'operateur cree un workflow d'initialisation.
2. Gateway genere ou charge les cles/certificats.
3. Le client EBICS emet `INI` / `HIA` ou `H3K`.
4. Gateway genere la lettre EBICS a partir des donnees persistees.
5. L'operateur transmet la lettre hors bande et renseigne l'evidence.
6. Gateway attend le retour d'activation banque.
7. L'abonne passe en etat actif.

### 6.4 Flux de rotation de cles

1. Un workflow de rotation est ouvert.
2. Le nouveau materiel est cree et stocke en etat futur.
3. Les ordres de rotation sont emis.
4. Gateway attend la confirmation necessaire.
5. La bascule d'etat active la nouvelle cle.
6. L'ancien materiel est retire ou archive.

### 6.5 Flux RTN

1. Un composant d'ingestion recoit une notification RTN.
2. Le message est parse et valide.
3. Une cle d'idempotence est calculee et controlee.
4. Un plan de pulls est resolu.
5. Le client EBICS execute les ordres standards correspondants.
6. Les resultats sont historises et relies a l'evenement RTN.

## 7. Decisions d'architecture recommandees

### 7.1 Reutiliser `Transfer` uniquement pour les flux fichier

Raison:

- `Transfer` est oriente echange de fichier et pipeline de tasks;
- certains ordres EBICS sont purement administratifs;
- surcharger `Transfer` pour tous les ordres complexifierait inutilement le
  coeur Gateway.

### 7.7 Garder la decision metier hors Gateway

Raison:

- la demande cible recentre Gateway sur le protocole;
- les workflows de signature metier sortent clairement du bon perimetre;
- Gateway doit etre un automate et un passe-plat fiable, pas un moteur de
  decision bancaire.

Le document
[frontiere-protocole-metier.md](c:\MonProjet\Waarp-Gateway\doc\ebics\frontiere-protocole-metier.md)
doit etre considere comme contrainte d'architecture.

### 7.2 Introduire des stores EBICS SQL dedies

Raison:

- la librairie EBICS exige une durabilite explicite pour `TxStore` et
  `NonceStore`;
- ces concepts ne correspondent pas proprement aux tables actuelles.

### 7.5 Introduire un workflow operateur pour la lettre EBICS

Raison:

- la librairie genere la lettre mais ne l'envoie pas;
- l'activation banque est exterieure a la session HTTP EBICS;
- Gateway doit donc assumer explicitement cette rupture d'automatisme.

### 7.6 Introduire RTN comme capacite transverse

Raison:

- RTN n'existe pas dans Gateway aujourd'hui;
- RTN releve plus d'une architecture d'evenements que d'un simple protocole de
  transfert;
- il faut un composant dedie pour intake, idempotence et auto-pull.

### 7.8 Stocker une vue contractuelle technique cote Gateway

Raison:

- `HPD`, `HKD`, `HTD` et `HAA` publient des informations directement utiles a
  l'execution securisee;
- laisser cette connaissance uniquement au metier expose a des erreurs
  d'exploitation evitables;
- cette vue reste protocolaire tant qu'elle est limitee aux capacites,
  permissions et BTF visibles via EBICS.

### 7.9 Introduire l'integration messaging comme capacite transverse

Raison:

- filesystem, REST et CLI ne couvrent pas tous les modeles d'integration
  d'entreprise;
- une messagerie AMQP augmente la valeur de Gateway comme point d'integration;
- le decouplage temporel est particulierement pertinent pour EBICS, RTN,
  reporting et passe-plat metier;
- cette capacite depasse EBICS et peut servir l'ensemble de la Gateway.

Decision complementaire:

- `AMQP 0.9.1` et `AMQP 1.0` doivent etre envisages non comme simples plugins
  EBICS mais comme deux protocoles Gateway autonomes.

Consequence:

- leur introduction peut constituer un prealable architectural pertinent avant
  l'implementation d'EBICS.

### 7.3 Garder la configuration dans `ProtoConfig` et les details durables en tables dediees

Raison:

- `ProtoConfig` est adapte au parametrage technique;
- il n'est pas adapte a l'etat transactionnel, aux nonces ni aux segments.

### 7.4 Rester compatible avec les canaux d'administration existants

Raison:

- l'operateur doit piloter EBICS comme le reste de Gateway;
- cela limite les couts d'exploitation et de formation.

## 8. Contraintes de securite

- TLS obligatoire en cible de production;
- mTLS activable par profil;
- stockage chiffre des secrets via les mecanismes existants;
- validations operateur explicites sur les etapes hors bande;
- rotation gouvernee par etat et evidence;
- `NonceStore` persistant si anti-rejeu;
- `TxStore` persistant si segmentation et recovery;
- event store RTN resistant aux doublons et aux rejeux;
- validation stricte H005 et `OrderData` par defaut.

## 9. Contraintes de performance

- accepter le streaming des charges plutot que le buffering massif;
- borner les tailles de payload, le nombre de segments et les timeouts;
- indexer les tables de transactions EBICS pour les recherches par `tx_id`,
  `host_id`, `partner_id`, `user_id`;
- prevoir des mecanismes de purge pour transactions et nonces.

## 10. Plan d'implementation recommande

### Lot 0. Protocoles d'integration metier

- module `amqp091`;
- module `amqp10`;
- contrats de message et outbox mutualisee;
- administration REST/CLI minimale;
- preuves de livraison, retry et observabilite.

### Lot 1. Socle protocolaire

- module `ebics`;
- configs;
- service serveur minimal;
- stores SQL;
- workflow d'initialisation et generation de lettre;
- tests d'integration de base.

### Lot 2. Flux fichier

- `FUL`, `FDL`, `BTU`, `BTD`;
- mapping vers `Transfer` et `Pipeline`;
- rotation de cles;
- historique et supervision.

### Lot 3. Administration et onboarding

- ressources REST/CLI;
- gestion des cles et abonnes;
- execution des ordres d'initialisation.
- pilotage operateur de la lettre et de l'activation.

### Lot 4. Ordres avances

- RTN, auto-pull et event store;
- ordres administratifs et reporting;
- ecrans UI dedies;
- observabilite avancee et runbooks.

## 11. Risques principaux

- sous-modeliser EBICS en essayant de tout loger dans `TransferInfo`;
- dupliquer la logique de la librairie EBICS dans Gateway;
- accepter des stores memoire au-dela du bootstrap;
- automatiser a tort l'initialisation jusqu'a l'activation banque;
- confondre rotation de cles et simple mise a jour de credential;
- integrer RTN comme un detail du client au lieu d'une capacite d'evenements;
- melanger politique metier banque et logique protocolaire;
- degrader la lisibilite d'exploitation par manque de correlation entre ordres,
  transactions et transferts.

## 12. Ligne directrice retenue

La bonne architecture consiste a:

- ajouter EBICS comme protocole natif de Gateway;
- confier le protocole pur a la librairie WAARP EBICS;
- confier a Gateway la persistance, l'administration, l'observabilite et
  l'orchestration;
- faire converger les flux fichier EBICS vers le pipeline standard de Gateway;
- traiter les ordres purement administratifs dans une couche EBICS dediee mais
  administree par les memes canaux que le reste de la plateforme.
