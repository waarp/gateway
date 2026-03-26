# Cadrage concret Phase E EBICS

## 1. Objet

Ce document detaille la `Phase E` du squelette technique EBICS.

La `Phase E` couvre:

- `EbicsRTNEvent`;
- les providers RTN;
- l'ingestion `WebSocket/WSS` de phase 1;
- l'idempotence, la quarantaine et le retry;
- l'auto-pull declenche par evenement.

## 2. Position de travail

La `Phase E` doit rester alignee sur la decision deja prise:

- RTN est une capacite transverse de Gateway;
- le transport cible de phase 1 est `WebSocket/WSS`;
- le coeur RTN doit rester decouple du transport;
- RTN ne doit pas etre absorbé par `Transfer` ni par un simple cron client.

## 3. Perimetre de la Phase E

### 3.1 Inclus

- model `EbicsRTNEvent`;
- model ou configuration de provider RTN;
- service d'ingestion RTN;
- provider `WebSocket/WSS` de phase 1;
- calcul de cle d'idempotence;
- retry / quarantine des evenements;
- resolution `event -> auto-pull`;
- correlation vers `EbicsOperation`.

### 3.2 Exclus

- support d'autres transports RTN que `WebSocket/WSS` en phase 1;
- orchestration metier de la consommation des reports;
- moteur de regles complexes d'evenements;
- supervision avancee hors besoins minimaux d'exploitation.

## 4. Exigences `production grade`

### 4.1 Idempotence

- tout evenement RTN doit avoir une cle d'idempotence stable;
- les doublons doivent etre detectes avant tout auto-pull;
- l'idempotence doit etre durable en base.

### 4.2 Retry et quarantaine

- un evenement RTN ne doit pas etre rejoue aveuglement;
- les erreurs transitoires doivent etre retryables selon une policy explicite;
- les cas ambigus ou mal formes doivent pouvoir etre quarantenes;
- la quarantaine doit rester visible et exploitable.

### 4.3 Decouplage transport

- les models et DTO ne doivent pas exposer de details `WebSocket` comme coeur
  du domaine;
- le `WebSocket/WSS` reste un adapter/provider;
- les evenements persistants doivent rester normalises.

### 4.4 Multi-SGBD

- stockage portable via XORM;
- pas de schema dependant d'un moteur particulier;
- pas de file/event store specifique base de donnees comme prerequis.

## 5. Fichiers a cadrer concretement

## 5.1 `pkg/model/table_names.go`

Ajout cible:

- `TableEbicsRTNEvents = "ebics_rtn_events"`

Si un objet provider persistant est retenu:

- `TableEbicsRTNProviders = "ebics_rtn_providers"`

## 5.2 `pkg/model/display_names.go`

Ajouts cibles:

- `NameEbicsRTNEvent = "ebics RTN event"`
- `NameEbicsRTNProvider = "ebics RTN provider"`

## 5.3 `pkg/model/ebics_rtn_event.go`

Responsabilite:

- porter l'evenement RTN normalise;
- conserver la cle d'idempotence;
- porter l'etat de traitement;
- fournir un pivot pour retry/quarantaine/auto-pull.

Invariants:

- `Source` obligatoire;
- `EventID` optionnel selon fournisseur, mais `IdempotenceKey` obligatoire;
- `Status` obligatoire;
- `ReceivedAt` obligatoire;
- `ProfileID` optionnel;
- `EbicsHostID` / `EbicsSubscriberID` optionnels mais fortement recommandes si
  deja resolus;
- unicite `(owner, idempotence_key)`.

## 5.4 `pkg/model/ebics_rtn_provider.go`

Responsabilite:

- porter la configuration d'un provider RTN;
- garder le coeur agnostique du transport tout en cadrant `WSS` en phase 1;
- permettre l'administration standard REST/CLI/UI.

Invariants:

- `Name` obligatoire;
- `Transport` obligatoire et borne a `wss` en phase 1;
- `Enabled` explicite;
- subscriber cible obligatoire si le provider est mono-abonne;
- configuration de connexion structuree et portable.

## 5.5 `pkg/protocols/modules/ebics/runtime/rtn_ingestion.go`

Responsabilite:

- normaliser un evenement brut venant du provider;
- calculer/verifier la cle d'idempotence;
- inserer ou retrouver l'evenement;
- declencher le plan d'action.

## 5.6 `pkg/protocols/modules/ebics/runtime/rtn_autopull.go`

Responsabilite:

- convertir un evenement RTN valide en demande(s) de pull EBICS standard;
- choisir le profil payload ou la strategie associee;
- creer les `EbicsOperation` correspondantes si necessaire.

## 5.7 `pkg/protocols/modules/ebics/rtn/provider.go`

Responsabilite:

- definir l'interface commune des providers RTN;
- isoler le coeur RTN du transport.

## 5.8 `pkg/protocols/modules/ebics/rtn/wss_provider.go`

Responsabilite:

- implementer le provider `WebSocket/WSS`;
- gerer connexion, reconnexion, heartbeat, fermeture propre;
- remettre au runtime des messages normalises.

## 5.9 `pkg/admin/rest/api/ebics_rtn.go`

Responsabilite:

- DTO `In/Out` des evenements RTN;
- DTO `In/Out` des providers RTN;
- actions `retry`, `quarantine`, `resume` si retenu.

## 5.10 `pkg/cmd/client/ebics_rtn.go`

Responsabilite:

- commandes d'exploitation RTN;
- vues lisibles `event` et `provider`;
- garde-fous sur retry/quarantine.

## 6. Axes de modelisation prioritaires

## 6.1 Evenement normalise

RTN doit etre modelise d'abord comme evenement persistant normalise, pas comme
message brut de transport.

## 6.2 Cle d'idempotence

La cle doit etre:

- stable;
- reproductible;
- calculable a partir du message et/ou des identifiants utiles;
- exploitable pour deduplication et support.

## 6.3 Auto-pull

L'auto-pull doit rester:

- explicite;
- gouverne par policy;
- traçable;
- et toujours relie a l'evenement source.

## 7. Ordre de pose

Ordre recommande:

1. `table_names.go` et `display_names.go`
2. `ebics_rtn_event.go`
3. `ebics_rtn_provider.go`
4. `rtn/provider.go`
5. `runtime/rtn_ingestion.go`
6. `runtime/rtn_autopull.go`
7. `rtn/wss_provider.go`
8. DTO `api`
9. CLI RTN

## 8. Definition de done de la Phase E

La `Phase E` est terminee si:

- un provider `WSS` peut etre administre et demarre proprement;
- un evenement RTN peut etre persiste avec deduplication durable;
- les statuts `received / processed / retry / quarantined` sont explicites;
- un evenement peut etre relie a une ou plusieurs operations EBICS;
- l'auto-pull cree des executions standardes et tracees;
- le transport reste bien un detail de provider, pas du coeur RTN.

## 9. Point de vigilance principal

Le risque central de la `Phase E` est de coder RTN comme une suite de callbacks
non traces ou comme une simple “option du client EBICS”.

La bonne ligne est:

- provider de transport isole;
- event store durable;
- policy d'idempotence et de retry explicite;
- et transformation RTN -> auto-pull pleinement observable.
