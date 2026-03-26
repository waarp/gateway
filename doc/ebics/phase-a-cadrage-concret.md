# Cadrage concret Phase A EBICS

## 1. Objet

Ce document detaille la `Phase A` du premier squelette technique EBICS.

Il traduit le cadrage general en un plan de travail fichier par fichier, en
gardant une posture `production grade` des le depart:

- objets explicites;
- invariants d'exploitation fixes;
- erreurs et observabilite pensees tout de suite;
- migrations et administration anticipees;
- stubs complets plutot que fragments minimaux ambigus.

## 2. Position de travail

La `Phase A` ne cherche pas a implementer tout le protocole EBICS.

En revanche, elle ne doit pas poser un squelette fragile ou jetable.

Elle doit livrer un socle deja propre sur:

- la modelisation durable minimale;
- l'enregistrement du protocole;
- la validation des `ProtoConfig`;
- l'alignement avec `updateconf` et l'import/export;
- la lisibilite de l'exploitation future.

## 3. Perimetre de la Phase A

### 3.1 Inclus

- constantes `Protocol` et identifiants EBICS;
- `ProtoConfig` serveur / client / partenaire EBICS;
- module `protocols.Module` EBICS enregistrable;
- models de base necessaires a l'identite et a la securite protocolaire:
  - `EbicsHost`
  - `EbicsSubscriber`
  - `EbicsBankKey`;
- conventions de nommage `pkg/model`;
- bases des erreurs techniques EBICS cote Gateway;
- points d'accroche `updateconf` / import-export;
- exigences de logs, audit et correlation.

### 3.2 Exclus

- soumission de payloads `BTU/BTD/FUL/FDL`;
- `EbicsOperation`;
- `EbicsContractView` / `EbicsContractViewItem`;
- `EbicsPayloadProfile`;
- `EbicsKeyLifecycle`;
- `EbicsInitializationWorkflow`;
- `RTN`;
- handlers REST et commandes CLI completes.

Ces objets restent prevus, mais pas implantes dans cette phase.

## 4. Exigences `production grade`

La `Phase A` doit etre pensee pour l'exploitation, meme si tous les flux
fonctionnels ne sont pas encore actifs.

### 4.0 Multi-SGBD et ORM

Gateway etant multi-SGBD, la `Phase A` doit rester strictement compatible avec
l'ORM existant et ses conventions.

Cela implique:

- aucune hypothese implicite "PostgreSQL only" ou "MySQL only";
- aucun SQL specifique a un moteur dans les models;
- privilegier les types Go/XORM deja utilises dans l'existant;
- reserver les requetes SQL manuelles aux cas vraiment necessaires, avec une
  vigilance explicite sur leur portabilite;
- ne pas introduire de colonnes JSONB, ARRAY ou types vendor-specifiques comme
  prealable du design EBICS.

Regle de conception:

- si un besoin n'est pas naturellement portable via l'ORM existant, il faut
  revoir le modele avant de coder.

### 4.1 Invariants

- aucune donnee critique EBICS ne doit reposer uniquement sur un `ProtoConfig`;
- les objets de persistance doivent deja porter `Owner`, dates et references
  techniques essentielles;
- les validations doivent etre deterministes et explicites;
- les messages d'erreur doivent etre orientables en exploitation;
- les noms des tables et objets doivent etre stables.
- les champs doivent rester mappables proprement via XORM sur les SGBD deja
  supportes par Gateway.

### 4.2 Exploitation

- chaque objet doit avoir une cle fonctionnelle lisible en plus de son `ID`;
- chaque erreur de validation doit etre exploitable sans debug du code;
- chaque `ProtoConfig` doit pouvoir etre round-trippe proprement via
  `updateconf`;
- les champs secrets doivent rester dans les objets deja prevus a cet effet
  (`Credential`, `CryptoKey`, secret text), pas dans des champs libres.
- les structures persistantes doivent eviter les serialisations opaques qui
  compliqueraient les migrations et les UI sur plusieurs SGBD.

### 4.3 Observabilite

Les stubs de `server.go` et `client.go` doivent deja prevoir:

- un logger nomme `ebics`;
- une correlation par `hostID`, `partnerID`, `userID` quand disponible;
- des messages differenciant `config`, `startup`, `shutdown`, `validation`,
  `protocol bootstrap`.

## 5. Fichiers a cadrer concretement

## 5.1 `pkg/model/table_names.go`

Ajouts cibles:

- `TableEbicsHosts = "ebics_hosts"`
- `TableEbicsSubscribers = "ebics_subscribers"`
- `TableEbicsBankKeys = "ebics_bank_keys"`

Raison:

- verrouiller des maintenant les noms SQL de base;
- eviter les renommages tardifs couteux.

## 5.2 `pkg/model/display_names.go`

Ajouts cibles:

- `NameEbicsHost = "ebics host"`
- `NameEbicsSubscriber = "ebics subscriber"`
- `NameEbicsBankKey = "ebics bank key"`

Raison:

- garder les erreurs et messages admin homogenees avec l'existant.

## 5.3 `pkg/model/ebics_host.go`

Responsabilite:

- representer un `HostID` EBICS local ou gere par Gateway;
- porter la politique technique de base du host EBICS;
- servir de point d'ancrage a la relation `host -> subscribers -> bank keys`.

Champs cibles de `Phase A`:

- `ID int64`
- `Owner string`
- `Name string`
- `HostID string`
- `Description string`
- `Enabled bool`
- `ProtocolVersion string`
- `Transport string`
- `IsServer bool`
- `DefaultBankURL string`
- `CreatedAt time.Time`
- `UpdatedAt time.Time`

Methodes minimales:

- `TableName() string`
- `Appellation() string`
- `GetID() int64`
- `BeforeWrite(db database.Access) error`

Validations a poser des la `Phase A`:

- `HostID` obligatoire;
- unicite de `Name`;
- unicite de `HostID` par `Owner`;
- `Transport` borne a la cible initiale supportee;
- `ProtocolVersion` borne a `H004` / `H005` selon la ligne retenue.

Points d'attention:

- ne pas stocker ici les clefs privees;
- ne pas dupliquer des informations deja presentes dans `LocalAgent`.

## 5.4 `pkg/model/ebics_subscriber.go`

Responsabilite:

- representer l'identite protocolaire `PartnerID/UserID` rattachee a un host;
- servir de pivot futur pour les `EbicsOperation`, profils payload et contrat.

Champs cibles de `Phase A`:

- `ID int64`
- `Owner string`
- `EbicsHostID int64`
- `Name string`
- `PartnerID string`
- `UserID string`
- `SystemID string`
- `TransportURL string`
- `Enabled bool`
- `DefaultOrderDataEncoding string`
- `CreatedAt time.Time`
- `UpdatedAt time.Time`

Methodes minimales:

- `TableName() string`
- `Appellation() string`
- `GetID() int64`
- `BeforeWrite(db database.Access) error`

Validations a poser:

- `EbicsHostID` obligatoire;
- `PartnerID` obligatoire;
- `UserID` obligatoire;
- unicite `(ebics_host_id, partner_id, user_id)`;
- validation de l'existence du host parent.

Points d'attention:

- `SystemID` est optionnel en `Phase A`;
- pas de semantique de contrat ici;
- pas de semantique de signature ici.

## 5.5 `pkg/model/ebics_bank_key.go`

Responsabilite:

- stocker les clefs de banque de reference pour verification;
- separer clairement les usages `X002` / `E002` ou equivalents selon version;
- preparer les futures operations `HPB` et rotations cote banque.

Champs cibles de `Phase A`:

- `ID int64`
- `Owner string`
- `EbicsHostID int64`
- `KeyType string`
- `Version string`
- `PublicKey string`
- `PublicKeyHash string`
- `State string`
- `ValidFrom time.Time`
- `ValidTo time.Time`
- `CreatedAt time.Time`
- `UpdatedAt time.Time`

Methodes minimales:

- `TableName() string`
- `Appellation() string`
- `GetID() int64`
- `BeforeWrite(db database.Access) error`

Validations a poser:

- `EbicsHostID` obligatoire;
- `KeyType` obligatoire et borne;
- `PublicKey` ou reference equivalente obligatoire;
- `State` borne a un ensemble reduit des le depart:
  - `imported`
  - `validated`
  - `retired`;
- verification d'unicite logique du triplet `(ebics_host_id, key_type, version)`.

Points d'attention:

- la `Phase A` peut rester sur stockage de cle publique;
- l'extraction/verification de digest peut rester dans une aide technique
  dediee;
- ne pas melanger avec `Credential`, qui porte le materiel client/local.

## 5.6 `pkg/protocols/modules/ebics/constants.go`

Responsabilite:

- definir les constantes stables du module.

Contenu cible:

- nom du protocole, par exemple `const EBICS = "ebics"`;
- variantes si necessaire plus tard;
- constantes de version;
- constantes de transport cible `https`, `wss` pour RTN futur;
- categories d'erreurs internes.

Exigence:

- aucun litteral disperse dans les autres fichiers.

## 5.7 `pkg/protocols/modules/ebics/config.go`

Responsabilite:

- definir les `ServerConfig`, `ClientConfig`, `PartnerConfig`;
- implementer `ValidServer`, `ValidClient`, `ValidPartner`;
- garder un `ProtoConfig` lisible et limite au parametrage technique.

Alignement code:

- suivre le pattern de [config.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\modules\sftp\config.go).

Contenu cible:

- `type serverConfig struct { ... }`
- `type clientConfig struct { ... }`
- `type partnerConfig struct { ... }`

Champs minimaux a poser des la `Phase A`:

- `protocolVersion`
- `endpointURL`
- `requestTimeout`
- `maxSegmentSize`
- `allowRecovery`
- `tlsMinVersion`
- `verifyBankKeys`
- `defaultOrderDataEncoding`
- `profilePolicy`

Validations:

- `endpointURL` requis quand pertinent;
- `requestTimeout > 0`;
- `maxSegmentSize > 0`;
- `profilePolicy` borne a:
  - `profile-required`
  - `profile-preferred`
  - `free-input-allowed`;
- `tlsMinVersion` borne a la politique supportee.

## 5.8 `pkg/protocols/modules/ebics/module.go`

Responsabilite:

- implementer `protocols.Module`.

Alignement code:

- suivre l'interface exposee par [protocol.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\protocol.go).

Methodes a fournir:

- `NewServer`
- `NewClient`
- `MakeServerConfig`
- `MakeClientConfig`
- `MakePartnerConfig`

Exigence `production grade`:

- aucun panic;
- si le runtime n'est pas encore complet, renvoyer des stubs explicites et
  journalisables plutot que des comportements implicites.

## 5.9 `pkg/protocols/modules/ebics/server.go`

Responsabilite:

- poser le squelette du serveur EBICS Gateway.

Ce que le stub doit deja faire:

- accepter `db` et `LocalAgent`;
- charger/valider le `ServerConfig`;
- initialiser logger et contexte technique;
- exposer des erreurs nettes `not implemented yet` sur les points non livres.

Ce qu'il ne doit pas faire:

- parser des ordres EBICS partiellement a la main;
- lancer des routines opaques non traquees;
- cacher une config invalide.

## 5.10 `pkg/protocols/modules/ebics/client.go`

Responsabilite:

- poser le squelette du client EBICS Gateway.

Ce que le stub doit deja faire:

- accepter `db` et `Client`;
- charger/valider `ClientConfig`;
- preparer la resolution future vers `RemoteAgent` / `RemoteAccount` /
  `EbicsSubscriber`;
- journaliser les parametres utiles hors secrets.

## 5.11 `pkg/protocols/modules/ebics/errors.go`

Responsabilite:

- centraliser les erreurs du module EBICS cote Gateway.

Contenu cible:

- erreurs de configuration;
- erreurs de bootstrap;
- erreurs d'objet absent;
- erreurs de compatibilite protocolaire;
- erreurs de fonctionnalite non encore implementee.

Exigence:

- utiliser des erreurs nommees et stables;
- separer erreurs internes Gateway et futurs return codes EBICS.

## 5.12 `pkg/protocols/modules.go`

Ajout cible:

- enregistrement du protocole `ebics` dans la map globale.

Alignement code:

- suivre [modules.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\modules.go).

Exigence:

- l'ajout doit rendre `protocols.IsValid("ebics")` vrai;
- les checks `ConfigChecker` doivent fonctionner sans code special hors norme.

## 5.13 `pkg/tasks/updateconf.go` et `pkg/backup/*`

Responsabilite:

- garantir que les nouveaux `ProtoConfig` EBICS peuvent etre importes/exportes
  proprement.

Objectif de `Phase A`:

- round-trip sans perte des `ProtoConfig`;
- prise en compte du protocole `ebics` dans les chemins nominaux;
- documentation d'exemple JSON/YAML a fournir plus tard.

## 6. Migrations et ordre de pose

Ordre recommande:

1. `pkg/model/table_names.go` et `display_names.go`
2. `pkg/model/ebics_host.go`
3. `pkg/model/ebics_subscriber.go`
4. `pkg/model/ebics_bank_key.go`
5. `pkg/protocols/modules/ebics/constants.go`
6. `pkg/protocols/modules/ebics/config.go`
7. `pkg/protocols/modules/ebics/errors.go`
8. `pkg/protocols/modules/ebics/server.go`
9. `pkg/protocols/modules/ebics/client.go`
10. `pkg/protocols/modules/ebics/module.go`
11. `pkg/protocols/modules.go`
12. `pkg/tasks/updateconf.go` et `pkg/backup/*`

Raison:

- on fixe d'abord le socle de donnees;
- puis le contrat protocolaire;
- puis le branchement produit.

## 7. Definition de done de la Phase A

La `Phase A` est terminee si:

- `ebics` est reconnu comme protocole valide par Gateway;
- les trois `ProtoConfig` sont validables et round-trippables;
- les models `EbicsHost`, `EbicsSubscriber`, `EbicsBankKey` existent et
  valident leurs invariants de base;
- aucune donnee sensible EBICS n'est placee dans un champ inadapte;
- les erreurs de config et d'objet manquant sont explicites;
- le socle reste coherent avec `Credential` et `CryptoKey`;
- les stubs `server/client` sont deja exploitables pour bootstrap et debug.

## 8. Point de vigilance principal

La derive la plus dangereuse serait de produire un faux squelette "minimal"
qui:

- enregistre un protocole sans vrai contrat de config;
- cree des models sans validations;
- reporte l'observabilite a plus tard;
- et oblige ensuite a plusieurs passes de rework.

La bonne ligne est donc:

- peu de fichiers au debut;
- mais chacun deja pense pour l'exploitation, la maintenance et les futures
  extensions EBICS.

Le niveau de declaration Go exact des structs et signatures est detaille dans:

- `phase-a-structs-et-signatures.md`
