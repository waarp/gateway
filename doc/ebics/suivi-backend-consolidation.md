# Suivi de consolidation backend EBICS

## 1. Usage

Cette checklist sert a piloter la fermeture du backend EBICS avant le frontend.

Regles:

- cocher `[x]` quand l'item est termine, relu et valide techniquement;
- laisser `[ ]` si l'item n'est pas encore ferme;
- utiliser `[-]` pour un report explicite;
- documenter toute divergence structurelle.

## 2. Gate de sortie backend

- [x] Plus aucun `ErrNotImplemented` sur le chemin nominal EBICS
- [ ] Plus aucun endpoint/commande EBICS expose sans logique runtime suffisante
- [x] Plus aucun `replace` local vers `lib-ebics`
- [x] Import/export/updateconf complets pour les objets EBICS administres
- [x] Catalogue BTF standard disponible pour les validations pre-contractuelles
- [ ] Politique d'exploitation documentee et relue
- [ ] Backend declare "pret frontend"

## 3. Lot B1 - Execution cliente reelle

- [x] Remplacer le stub `InitTransfer` dans `pkg/protocols/modules/ebics/client.go`
- [x] Definir le mapping `Transfer -> ordre client EBICS`
- [x] Creer la creation d'`EbicsOperation` cote client
- [x] Creer la creation d'`EbicsTransaction` cote client quand necessaire
- [x] Brancher `BTU/BTD` cote client
- [x] Confirmer que `FUL/FDL` restent des alias de compatibilite normalises vers `BTU/BTD` en cible `EBICS 3.0.2`
- [x] Brancher `HEV` et le refresh contractuel `HPD` / `HKD` / `HTD` / `HAA` cote client
- [x] Brancher le reste du reporting / des ordres admin cote client
- [x] Brancher initialisation / key management cote client
- [x] Garantir la correlation `operation / transaction / transfer`
- [x] Verifier l'exploitation des return codes `technical/business`

## 4. Lot B2 - Couverture backend complete

- [x] Revoir chaque famille REST EBICS et confirmer l'absence de logique partielle
- [x] Revoir chaque famille CLI EBICS et confirmer l'absence de logique partielle
- [x] Verifier que `payloads` est bien exploitable de bout en bout
- [x] Verifier que `operations` est bien exploitable de bout en bout
- [x] Verifier que `transactions` est bien exploitable de bout en bout
- [x] Verifier que `contract views` est bien exploitable de bout en bout
- [x] Verifier que `payload profiles` est bien exploitable de bout en bout
- [x] Verifier que `initializations` est bien exploitable de bout en bout
- [x] Verifier que `key lifecycles` est bien exploitable de bout en bout
- [x] Verifier que `RTN` est bien exploitable de bout en bout

## 5. Lot B3 - Import / export / updateconf

- [x] Etendre `pkg/backup/export.go`
- [x] Etendre `pkg/backup/import.go`
- [x] Ajouter les helpers `*_export.go`
- [x] Ajouter les helpers `*_import.go`
- [x] Cadrer les jeux JSON/YAML de reference
- [x] Verifier le round-trip complet des `ProtoConfig`
- [x] Verifier le round-trip complet des objets EBICS administres

## 6. Lot B4 - Durcissement exploitation

## 5 bis. Lot B3.5 - Catalogue BTF standard

- [x] Cadrer l'objet de persistance dedie
- [x] Arreter la source initiale `GLB`
- [x] Arreter la source initiale `FR`, `DE`, `AT`, `CH`
- [x] Figer la regle `specific > country > GLB`
- [x] Cadrer la strategie de seed versionne
- [x] Cadrer les surfaces REST/CLI minimales
- [x] Cadrer le fallback runtime dans `contract_validation`

## 6. Lot B4 - Durcissement exploitation

- [ ] Revoir l'execution serveur EBICS reelle sur le chemin nominal `BTU/BTD`
- [ ] Revoir la couverture normative des ordres serveur non payload exposes
- [ ] Revoir la segmentation / reprise / recovery cote serveur
- [ ] Revoir la journalisation des flux serveur EBICS
- [ ] Revoir la journalisation des flux client EBICS
- [ ] Revoir les messages d'erreur REST EBICS
- [ ] Revoir les messages CLI EBICS
- [ ] Revoir les statuts operateur visibles
- [ ] Revoir la purge / retention des nonces
- [ ] Revoir la purge / retention des transactions
- [ ] Revoir la purge / retention des evenements RTN
- [ ] Revoir la coherences des reprises / recovery
- [ ] Revoir la discipline multi-SGBD / XORM
- [ ] Revoir les protections de suppression / mutation sur objets sensibles

## 7. Lot B5 - Verification de sortie

- [ ] Rejouer une passe `rg ErrNotImplemented|not implemented` et solder tous les cas EBICS
- [ ] Rejouer une passe linter sur le perimetre backend EBICS
- [ ] Rejouer une passe compilation/test ciblee sur le perimetre backend EBICS
- [ ] Revoir les documents de suivi
- [ ] Declarer la gate backend "GO frontend"

## 8. Notes

- Date de creation: 2026-03-27
- Cible: backend EBICS complet avant chantier frontend
- 2026-03-31: revue de situation B1 -> B5 rejouee sur le depot.
  `B1`, `B2`, `B3` et `B3.5` restent consideres fermes;
  `B4` et `B5` restent ouverts.
  La dependance `code.waarp.fr/lib/ebics` est bien referencee sans `replace`
  local actif dans `go.mod`.
  Le principal reste a faire avant frontend est maintenant concentre sur
  l'exploitation serveur/provider et sur la passe finale de verification.
- 2026-03-31: `Lot 1A` est maintenant ferme.
  Une premiere vague de tests a ete ajoutee dans
  `pkg/protocols/modules/ebics/server_test.go` pour couvrir le cycle de vie du
  service EBICS: `Start/Stop`, double demarrage, arret hors etat running,
  erreur de configuration, erreur d'ecoute TLS et resolution du repertoire XSD.
  Verification rejouee: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model`.
- 2026-03-31: `Lot 1B` est maintenant ferme pour sa premiere vague de tests
  unitaires de routage payload serveur. Le fichier
  `pkg/protocols/modules/ebics/order_router_test.go` couvre a present:
  selection de profil la plus specifique, rejet des profils ambigus,
  validation contractuelle et mapping de return codes, derive de nom de fichier
  entrant, correlation runtime, enrichissement des metadonnees `Transfer`,
  resolution `host / subscriber / local account`, et rejet d'un profil payload
  lie a une regle Gateway de direction incompatible.
  Cette fermeture ne solde pas encore le scenario nominal complet
  `Upload/Download` avec creation verifiee de `EbicsOperation /
  EbicsTransaction / Transfer`; ce point reste porte par `Lot 1D`.
  Verification rejouee: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model`.
- 2026-03-31: `Lot 1C` est maintenant ferme sur la persistance provider et les
  stores effectivement utilises par le runtime EBICS. Le fichier
  `pkg/protocols/modules/ebics/provider_store_test.go` couvre a present:
  lifecycle transaction (`Create/Get/Update/Purge`), persistance des segments,
  lecture/ecriture du recovery, et lifecycle des nonces (`Seen/Store/Purge`).
  La revue a confirme que `pkg/protocols/modules/ebics/stores/tx_store.go` et
  `pkg/protocols/modules/ebics/stores/operation_store.go` ne portent ici que
  des interfaces de contrat, sans logique de persistance supplementaire a
  tester separement a ce stade.
  Verification rejouee: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`.
- 2026-03-31: `Lot 1D` est maintenant ferme avec
  `pkg/protocols/modules/ebics/server_integration_test.go`.
  La couverture ajoute un demarrage reel du service EBICS Gateway avec stores
  reels, un upload `BTU` nominal jusqu'au fichier final et a l'historique, puis
  un download `BTD` nominal jusqu'au payload retourne et a l'archivage du
  transfert. La persistance transactionnelle `TxStore/NonceStore/segments`
  reste couverte en profondeur par `Lot 1C`; ce lot ferme le chemin serveur
  payload `operation + transfert + pipeline + historique`.
  Verification rejouee: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model`.
- 2026-03-31: `Lot 1E` est maintenant ferme apres correction de deux defauts
  reels reveles par `Lot 1D`:
  preservation de la correlation apres archivage du transfert cote serveur
  payload (ID archive deplace dans les metadonnees d'operation avec fallback
  REST), et suppression d'une course `Start/Stop` dans `server.go` ou la
  goroutine `Serve` pouvait dereferencer `s.httpServer` apres remise a `nil`.
  Verification rejouee: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/admin/rest/... ./pkg/admin/rest/api`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/admin/rest ./pkg/admin/rest/api`.
- 2026-03-31: `Lot 2A` est maintenant ferme avec enrichissement de
  `pkg/protocols/modules/ebics/provider_store_test.go` et ajout de tests
  modeles `pkg/model/ebics_transaction_test.go` et
  `pkg/model/ebics_transaction_segment_test.go`.
  La couverture verrouille la reprise/segmentation serveur sur:
  statuts `RUNNING/RECOVERING`, lecture recovery apres rechargement DB, creation
  sur `UpdateTransaction`, absence de faux positifs sur transaction absente,
  duplication de segments, et monotonie `currentSegment/segmentCount`.
  Un defaut runtime reel a ete corrige dans `provider_store.go`:
  `AddSegment` ne peut plus retrograder `segmentCount` sous `currentSegment`
  lors d'une reprise ou d'un upsert de segment.
  Verification rejouee: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`.
- 2026-03-31: `Lot 2B` est maintenant ferme avec enrichissement de
  `pkg/protocols/modules/ebics/provider_store_test.go` et ajout de
  `pkg/model/ebics_nonce_test.go`.
  La couverture verrouille l'anti-rejeu sur: trimming coherent entre
  `StoreNonce` et `SeenNonce`, portee du nonce par subscriber, rejet des
  doublons pour un meme subscriber, acceptation du meme nonce pour des
  subscribers distincts, et comportement de purge a la borne exacte
  d'expiration.
  Cote modele, `EbicsNonce` est maintenant couvert sur la contrainte
  `ExpiresAt > Timestamp` et sur l'unicite par subscriber.
  Verification rejouee: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`.
- 2026-03-31: `Lot 2C` est maintenant ferme avec durcissement de
  `pkg/protocols/modules/ebics/provider_store.go` et enrichissement
  complementaire de `pkg/protocols/modules/ebics/provider_store_test.go`.
  Trois defauts runtime reels ont ete corriges sur le chemin serveur de
  reprise: `UpdateTransaction` preserve desormais `segmentCount` quand
  `lib-ebics` n'envoie plus `SegmentCnt` sur les segments suivants,
  `UpdateRecovery` fait passer la transaction en statut `RECOVERING`, et
  `AddSegment` remet explicitement la transaction en `RUNNING` en mettant aussi
  a jour son horodatage.
  La couverture ajoutee verrouille ces transitions d'etat et la conservation du
  contexte de reprise.
  Verification rejouee: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`.
- 2026-03-31: `Lot 3A` est maintenant ferme avec ajout de
  `pkg/admin/rest/ebics_handlers_test.go`.
  La premiere vague de couverture REST verrouille les handlers `payloads /
  operations / transactions` sur les points operateur les plus sensibles:
  filtrage effectif des `payloads` sur les ordres `BTU/BTD`, action `recover`
  qui reinitialise correctement l'etat exploitable d'une operation payload,
  detail d'operation avec fallback sur `archivedTransferID` et remontée des
  segments tries, et lecture detaillee d'une transaction avec segments ordonnes.
  Verification rejouee: `golangci-lint run ./pkg/admin/rest/... ./pkg/admin/rest/api`
  puis `go test ./pkg/admin/rest ./pkg/admin/rest/api`.
- 2026-03-31: `Lot 3B` est maintenant ferme avec extension de
  `pkg/admin/rest/ebics_handlers_test.go`.
  La couverture REST verrouille a present le detail `contract views` avec items
  ordonnes, une action locale `key lifecycle` avec transition de statut et
  evidence persistees, et une action locale `initialization` avec annulation
  coherente du workflow.
  Verification rejouee: `golangci-lint run ./pkg/admin/rest/... ./pkg/admin/rest/api`
  puis `go test ./pkg/admin/rest ./pkg/admin/rest/api`.
- 2026-03-27: `Lot B1` est entame et couvre maintenant le chemin nominal payload client
  `BTU/BTD` avec creation `EbicsOperation` / `EbicsTransaction`, contrat actif,
  TLS, recovery et correlation `transfer`.
  La cible `EBICS 3.0.2` est maintenant figee: `BTU/BTD` sont canoniques,
  `FUL/FDL` restent de simples alias de compatibilite normalises.
  Reste a fermer dans `B1`: les familles client
  reporting/admin/initialisation/key-management.
- 2026-03-27: le client hors payload couvre maintenant une execution reelle des
  actions d'initialisation `INI` / `HIA` / `H3K` et de la synchronisation banque
  `HPB`, avec creation d'`EbicsOperation`, persistance des references dans
  `EbicsInitializationWorkflow`, generation de la lettre `H3K` et persistance des
  cles banque.
- 2026-03-27: le client hors payload couvre maintenant aussi `HEV` et le refresh
  contractuel `HPD` / `HKD` / `HTD` / `HAA`, avec persistance de snapshots dans
  `EbicsContractView` / `EbicsContractViewItem`, exposition REST/CLI de l'action
  de refresh, et correction du modele `EbicsOperation` pour accepter les ordres
  non payload.
- 2026-03-27: le client hors payload couvre maintenant aussi le reporting
  `HVD` / `HVU` / `HVZ` / `HVT` / `HAC` et les signatures protocolaires
  `HVE` / `HVS`, avec actions REST/CLI dediees sur `ebics operation`,
  execution reelle via `lib-ebics`, persistance d'`EbicsOperation` et
  metadonnees d'exploitation.
- 2026-03-27: le key management de rotation client couvre maintenant une
  orchestration coordonnee `PUB` / `HCA` / `HCS` / `HSA` / `SPR`, avec
  coexistence `ACTIVATED + pending`, `coordinationID`, actions REST/CLI
  `prepare/send/confirm/cancel/reject/revoke`, et activation qui retire
  explicitement l'ancien lifecycle au profit du nouveau.
  `Lot B1` est maintenant ferme sur le perimetre d'execution cliente reelle.
- 2026-03-27: `Lot B2` est demarre avec fermeture des premiers ecarts
  objectifs de couverture CLI par rapport au backend REST:
  ajout de `ebics transaction list/get`, ajout de `ebics payload profile delete`
  et ajout de `ebics rtn provider delete`.
  Le lot reste ouvert tant que la revue complete REST/CLI et l'exploitation
  bout en bout de chaque famille EBICS n'ont pas ete bouclees.
- 2026-03-27: une passe `rg ErrNotImplemented|not implemented` sur
  `pkg/protocols/modules/ebics`, `pkg/admin/rest` et `pkg/cmd/client`
  ne remonte plus aucun cas EBICS actif.
- 2026-03-27: la famille `transactions` couvre maintenant aussi la lecture
  detaillee des segments en REST et en CLI:
  `GET /api/ebics/transactions/{transaction}/segments`,
  `GET /api/ebics/transactions/{transaction}/segments/{segment}`,
  `waarp-gateway ebics transaction segments` et
  `waarp-gateway ebics transaction segment`.
- 2026-03-27: la famille `operations` est maintenant explicitee comme une
  famille d'observabilite et d'actions specialisees:
  detail enrichi avec transaction, segments, identites et liens techniques;
  maintien des actions `reporting` et `signature`;
  abandon assume d'une facade generique `retry/cancel/confirm`, ces actions
  restant portees par les familles specialisees.
- 2026-03-27: les familles `contract views`, `payload profiles`,
  `initializations`, `key lifecycles` et `RTN` ont ete relues cote REST et
  CLI. Leur couverture est consideree suffisante pour `B2`, avec enrichissement
  des sorties operateur (items contractuels, metadonnees de profil, horodatages
  et references d'operations/workflows). Le point restant principal du lot est
  maintenant la verification bout en bout de `payloads`, puis la cloture
  synthese des familles REST/CLI EBICS.
- 2026-03-27: la famille `payloads` est maintenant consideree exploitable de
  bout en bout pour `B2`, avec soumission REST/CLI, controle contractuel,
  creation/correlation d'`EbicsOperation`, typage d'operation `PAYLOAD`
  explicite cote client et cote serveur, et actions operateur `retry/recover`.
  Le `Lot B2` est considere ferme.
- 2026-03-27: le plan de consolidation est precise pour rendre explicite le
  chantier serveur. Le durcissement de l'execution serveur reelle EBICS
  (payload nominal, ordres non payload, segmentation/reprise, observabilite)
  est maintenant porte explicitement par le `Lot B4`.
- 2026-03-27: `Lot B3` est demarre. Le backup EBICS couvre maintenant les
  objets administres `hosts`, `subscribers`, `bank keys`, `payload profiles`
  et `RTN providers`, avec orchestration d'import/export dans `pkg/backup`.
  Les objets purement operationnels (`operations`, `transactions`,
  `initializations`, `key lifecycles`, `RTN events`, `contract views`)
  restent volontairement hors sauvegarde de configuration. Les round-trips
  complets JSON/YAML et `updateconf` restent a verifier.
- 2026-03-27: `Lot B3` est maintenant ferme. Le perimetre EBICS administre
  (`hosts`, `subscribers`, `bank keys`, `payload profiles`, `RTN providers`)
  est couvert en import/export, avec jeux de reference `JSON/YAML`,
  test de round-trip dedie dans `pkg/backup`, verification `updateconf`,
  et migration `0.16.0` pour garantir la presence des tables EBICS sur les
  bases creees ou migrees.
- 2026-03-27: `Lot B3.5` est maintenant ferme. Gateway porte un catalogue BTF
  standard distinct des contrats banques, avec tables dediees,
  seed versionne `GLB/FR/DE/AT/CH`, bootstrap sur base fraiche,
  migration de rattrapage sur base existante, import/export/updateconf,
  et fallback runtime strict `specific > country > GLB`.
  Si un contrat specifique actif existe et qu'un tuple n'y est pas trouve,
  l'echange est rejete sans fallback vers le catalogue standard.

## 9. Priorisation des ecarts restants

- 2026-04-01: ouverture du chantier `P0` RTN reel.
  Un nouveau service de fond `EBICS RTN` est maintenant branche dans
  `gatewayd` et expose dans `services.Core`.
  Il charge les providers RTN actives, demarre le transport `WSS`, consomme
  `Events/Errors`, persiste l'ingestion idempotente, met a jour l'etat runtime
  des providers, puis derive un auto-pull en programmant un vrai `Transfer`
  Gateway immediat relie a une `EbicsOperation` `AUTO_TRIGGERED`.
  Le point important est architectural: le service RTN ne contourne pas le
  runtime client EBICS, il s'appuie sur le mecanisme natif des `Transfer`
  planifies pour declencher l'execution existante.
  La premiere vague de tests dedies couvre ce service dans
  `pkg/protocols/modules/ebics/rtn_service_test.go`.
  Verification rejouee:
  `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`
  puis
  `go test ./pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`.
  Limite residuelle explicite: le `P0` ferme la programmation du pull, mais
  pas encore la consommation terminale `BTD` jusqu'au payload final dans un
  scenario RTN de bout en bout.

### Bloquants frontend

- Valider l'execution serveur EBICS reelle sur le chemin nominal `BTU/BTD`
- Valider la couverture normative des ordres serveur non payload exposes
- Valider la segmentation / reprise / recovery cote serveur
- Solder la gate "plus aucun endpoint/commande EBICS expose sans logique runtime suffisante"
- Rejouer la passe de sortie backend `B5` avant de prononcer la gate frontend
- Rejouer la lecture de sortie backend au regard des specs fonctionnelles,
  techniques et d'architecture, pas seulement du code courant et des suivis
- Fermer le dernier ecart RTN entre creation d'operation auto-triggered et
  execution effective du `BTD` jusqu'au payload final

### Importants

- Revoir la journalisation des flux serveur EBICS
- Revoir la journalisation des flux client EBICS
- Revoir les messages d'erreur REST EBICS
- Revoir les messages CLI EBICS
- Revoir les statuts operateur visibles
- Revoir la coherence des reprises / recovery
- Revoir la discipline multi-SGBD / XORM
- Revoir les protections de suppression / mutation sur objets sensibles
- Repositionner explicitement les connecteurs de passe-plat metier
  (`filesystem`, `REST`, `CLI`, `AMQP 0.9.1`, `AMQP 1.0`) dans la lecture
  d'avancement EBICS, car ils font partie des specs et de l'architecture cible

### Confort exploitation

- Revoir la purge / retention des nonces
- Revoir la purge / retention des transactions
- Revoir la purge / retention des evenements RTN

### Discipline qualite

- Avant tout changement code EBICS, executer `golangci-lint` sur le perimetre
  cible avant compilation ou tests Go
- A chaque changement code EBICS, executer les tests unitaires cibles du
  perimetre touche puis une passe de non-regression backend EBICS
- Ne pas fermer `B4` ni `B5` sans enrichissement mesurable de la couverture de
  tests EBICS sur le client, le serveur, les handlers REST et la CLI

## 10. Backlog executable B4 / B5

### Etape 1. Consolider le serveur payload nominal

Objectif:

- verifier et durcir le chemin serveur `BTU/BTD` de bout en bout;
- fermer l'angle mort principal identifie avant frontend.

Fichiers cibles:

- `pkg/protocols/modules/ebics/server.go`
- `pkg/protocols/modules/ebics/order_router.go`
- `pkg/protocols/modules/ebics/provider_store.go`
- `pkg/protocols/modules/ebics/stores/tx_store.go`
- `pkg/protocols/modules/ebics/stores/operation_store.go`

Tests a ajouter ou enrichir en priorite:

- tests unitaires/integres sur le routage serveur payload
- tests de correlation `operation / transaction / transfer`
- tests de statuts et retours serveur en succes / echec

Commande qualite minimale:

- `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
- `go test ./pkg/protocols/modules/ebics/... ./pkg/model`

Sous-lots cochables:

- [x] Lot 1A - Poser les tests de cycle de vie serveur
  Fichier principal: `pkg/protocols/modules/ebics/server_test.go`
  Attendus: demarrage nominal, echec de configuration, echec de listener TLS,
  arret propre, double `Start` / double `Stop`, activation du profil XSD strict
  si disponible, comportement degrade explicite si le repertoire XSD est absent
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model`

- [x] Lot 1B - Poser les tests de routage payload serveur
  Fichier principal: `pkg/protocols/modules/ebics/order_router_test.go`
  Attendus: `BTU` nominal cree `EbicsOperation`, `EbicsTransaction` et le lien
  `Transfer`; `BTD` nominal cree les correlations attendues; rejet contractuel
  sans fallback interdit; mapping correct des return codes `technical` et
  `business`; absence de creation parasite d'objets sur un chemin rejete
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model`

- [x] Lot 1C - Couvrir les stores et la persistance provider
  Fichiers principaux: `pkg/protocols/modules/ebics/provider_store_test.go`,
  `pkg/protocols/modules/ebics/stores/tx_store_test.go`,
  `pkg/protocols/modules/ebics/stores/operation_store_test.go`
  Attendus: lecture / ecriture transaction, persistance des segments si
  presents, coherence `owner / host / partner / user`, reprise correcte depuis
  la base
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`

- [x] Lot 1D - Ajouter un test d'integration serveur minimal
  Fichier principal: `pkg/protocols/modules/ebics/server_integration_test.go`
  ou `pkg/protocols/modules/ebics/server_test.go`
  Attendus: demarrage du serveur Gateway EBICS avec stores reels, execution d'un
  scenario payload nominal, verification de `operation / transaction / transfer`,
  verification du statut final exploitable
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model`

- [x] Lot 1E - Corriger le code apres la premiere vague de tests
  Fichiers principaux: `pkg/protocols/modules/ebics/server.go`,
  `pkg/protocols/modules/ebics/order_router.go`,
  `pkg/protocols/modules/ebics/provider_store.go`,
  `pkg/protocols/modules/ebics/stores/tx_store.go`,
  `pkg/protocols/modules/ebics/stores/operation_store.go`
  Attendus: corriger les ecarts trouves sans perdre le filet de securite
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model`

Ordre d'execution recommande:

1. [x] Lot 1A
2. [x] Lot 1B
3. [x] Lot 1C
4. [x] Faire passer la premiere vague de tests
5. [x] Lot 1E
6. [x] Lot 1D
7. [x] Rejouer linter + tests

### Etape 2. Cadrer la segmentation, reprise et recovery serveur

Objectif:

- verifier la persistance durable et les comportements de reprise;
- confirmer l'alignement avec les specs techniques et d'architecture.

Fichiers cibles:

- `pkg/protocols/modules/ebics/server.go`
- `pkg/protocols/modules/ebics/provider_store.go`
- `pkg/protocols/modules/ebics/stores/tx_store.go`
- `pkg/model/ebics_transaction.go`
- `pkg/model/ebics_transaction_segment.go`
- `pkg/model/ebics_nonce.go`

Tests a ajouter ou enrichir en priorite:

- tests de reprise sur transaction segmentee
- tests de persistance des segments
- tests de fenetre anti-rejeu / nonce

Commande qualite minimale:

- `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
- `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`

Sous-lots cochables:

- [x] Lot 2A - Poser les tests de segmentation serveur
  Fichiers principaux: `pkg/protocols/modules/ebics/server.go`,
  `pkg/protocols/modules/ebics/stores/tx_store.go`,
  `pkg/model/ebics_transaction.go`,
  `pkg/model/ebics_transaction_segment.go`
  Attendus: tests de reprise sur transaction segmentee, persistance des
  segments, verification du comptage et de l'etat de transaction
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`

- [x] Lot 2B - Poser les tests anti-rejeu / nonce
  Fichiers principaux: `pkg/protocols/modules/ebics/provider_store.go`,
  `pkg/model/ebics_nonce.go`
  Attendus: tests de fenetre anti-rejeu, persistance et rejet des doublons
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`

- [x] Lot 2C - Corriger le runtime serveur sur la reprise / recovery
  Fichiers principaux: `pkg/protocols/modules/ebics/server.go`,
  `pkg/protocols/modules/ebics/provider_store.go`,
  `pkg/protocols/modules/ebics/stores/tx_store.go`
  Attendus: comportement de recovery explicite, persistance durable, alignement
  avec les specs techniques et d'architecture
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/database/migrations`

Ordre d'execution recommande:

1. [x] Lot 2A
2. [x] Lot 2B
3. [x] Faire passer la vague de tests segmentation / nonce
4. [x] Lot 2C
5. [x] Rejouer linter + tests

### Etape 3. Fermer la couverture REST EBICS

Objectif:

- verifier que les handlers REST EBICS exposes ont une logique exploitable,
  des erreurs lisibles et des statuts operateur coherents.

Fichiers cibles:

- `pkg/admin/rest/ebics_payloads.go`
- `pkg/admin/rest/ebics_operations.go`
- `pkg/admin/rest/ebics_transactions.go`
- `pkg/admin/rest/ebics_contract_views.go`
- `pkg/admin/rest/ebics_key_lifecycles.go`
- `pkg/admin/rest/ebics_initializations.go`
- `pkg/admin/rest/ebics_rtn.go`
- `pkg/admin/rest/ebics_utils.go`

Tests a ajouter ou enrichir en priorite:

- tests REST dedies par famille EBICS
- tests de validation d'entrees et erreurs operateur
- tests de mapping DTO <-> model pour les sorties critiques

Commande qualite minimale:

- `golangci-lint run ./pkg/admin/rest/... ./pkg/admin/rest/api`
- `go test ./pkg/admin/rest ./pkg/admin/rest/api`

Sous-lots cochables:

- [x] Lot 3A - Couvrir les handlers REST payloads / operations / transactions
  Fichiers principaux: `pkg/admin/rest/ebics_payloads.go`,
  `pkg/admin/rest/ebics_operations.go`,
  `pkg/admin/rest/ebics_transactions.go`
  Attendus: tests REST dedies, validation d'entrees, erreurs operateur,
  mapping DTO <-> model sur les sorties critiques
  Validation: `golangci-lint run ./pkg/admin/rest/... ./pkg/admin/rest/api`
  puis `go test ./pkg/admin/rest ./pkg/admin/rest/api`

- [x] Lot 3B - Couvrir les handlers REST contract views / key lifecycles / initializations
  Fichiers principaux: `pkg/admin/rest/ebics_contract_views.go`,
  `pkg/admin/rest/ebics_key_lifecycles.go`,
  `pkg/admin/rest/ebics_initializations.go`,
  `pkg/admin/rest/ebics_utils.go`
  Attendus: tests REST dedies, sorties operateur lisibles, validations de
  references et d'etats
  Validation: `golangci-lint run ./pkg/admin/rest/... ./pkg/admin/rest/api`
  puis `go test ./pkg/admin/rest ./pkg/admin/rest/api`

- [x] Lot 3C - Couvrir les handlers REST RTN et stabiliser les erreurs EBICS
  Fichier principal: `pkg/admin/rest/ebics_rtn.go`
  Attendus: tests REST RTN dedies, statuts et erreurs coherents, absence de
  logique partielle exposee
  Validation: `golangci-lint run ./pkg/admin/rest/... ./pkg/admin/rest/api`
  puis `go test ./pkg/admin/rest ./pkg/admin/rest/api`

  2026-03-31: Lot 3C est maintenant ferme. Une premiere vague de tests couvre
  les transitions operateur `RETRY`, le rejet des actions RTN non supportees,
  la validation des providers sans configuration et la preservation des champs
  runtime (`LastConnectionAt`, `LastError`) lors d'un `update`.

Ordre d'execution recommande:

1. [x] Lot 3A
2. [x] Lot 3B
3. [x] Lot 3C
4. [x] Rejouer linter + tests REST complets

### Etape 4. Fermer la couverture CLI EBICS

Objectif:

- verifier que les commandes EBICS exposent correctement les actions backend et
  les messages operateur.

Fichiers cibles:

- `pkg/cmd/client/ebics_operations.go`
- `pkg/cmd/client/ebics_payload.go`
- `pkg/cmd/client/ebics_payload_profiles.go`
- `pkg/cmd/client/ebics_contract_views.go`
- `pkg/cmd/client/ebics_key_lifecycles.go`
- `pkg/cmd/client/ebics_initializations.go`
- `pkg/cmd/client/ebics_rtn.go`
- `cmd/waarp-gateway/main.go`

Tests a ajouter ou enrichir en priorite:

- tests CLI dedies par commande EBICS
- tests des sorties utilisateur et des erreurs
- tests des actions specialisees `reporting`, `signature`, `retry`, `recover`

Commande qualite minimale:

- `golangci-lint run ./pkg/cmd/client ./cmd/waarp-gateway`
- `go test ./pkg/cmd/client ./cmd/waarp-gateway`

Sous-lots cochables:

- [x] Lot 4A - Couvrir les commandes CLI payloads / operations / transactions
  Fichiers principaux: `pkg/cmd/client/ebics_payload.go`,
  `pkg/cmd/client/ebics_operations.go`,
  `pkg/cmd/client/ebics_transactions.go`,
  `cmd/waarp-gateway/main.go`
  Attendus: tests CLI dedies, sorties utilisateur lisibles, erreurs coherentes
  Validation: `golangci-lint run ./pkg/cmd/client ./cmd/waarp-gateway`
  puis `go test ./pkg/cmd/client ./cmd/waarp-gateway`

  2026-03-31: Lot 4A est maintenant ferme. Une premiere vague de tests couvre
  la construction des requetes payload `upload` / `download`, l'action
  `recover`, l'affichage detaille d'une operation, la gestion d'erreur sur une
  transaction incomplete et le cas vide sur les segments. Un defaut reel de
  serialisation JSON a ete corrige sur les commandes `payload retry/recover`
  pour ne plus envoyer les arguments positionnels au backend.

- [x] Lot 4B - Couvrir les commandes CLI contract views / key lifecycles / initializations
  Fichiers principaux: `pkg/cmd/client/ebics_contract_views.go`,
  `pkg/cmd/client/ebics_key_lifecycles.go`,
  `pkg/cmd/client/ebics_initializations.go`
  Attendus: tests des actions et sorties operateur, coherence avec le backend
  REST expose
  Validation: `golangci-lint run ./pkg/cmd/client ./cmd/waarp-gateway`
  puis `go test ./pkg/cmd/client ./cmd/waarp-gateway`

  2026-04-01: Lot 4B est maintenant ferme. Une premiere vague de tests couvre
  l'affichage detaille d'une contract view, la commande `refresh`, les actions
  `key lifecycle` et `initialization`, ainsi que la preparation d'une rotation
  de cles. Un defaut reel de contrat JSON a ete corrige sur `IncludeHEV`
  pour aligner la CLI avec le backend et les specs.

- [x] Lot 4C - Couvrir les commandes CLI RTN / actions specialisees
  Fichiers principaux: `pkg/cmd/client/ebics_rtn.go`,
  `pkg/cmd/client/ebics_payload_profiles.go`
  Attendus: tests des actions specialisees `reporting`, `signature`, `retry`,
  `recover` et messages utilisateur associes
  Validation: `golangci-lint run ./pkg/cmd/client ./cmd/waarp-gateway`
  puis `go test ./pkg/cmd/client ./cmd/waarp-gateway`

  2026-04-01: Lot 4C est maintenant ferme. Une premiere vague de tests couvre
  `RTN provider add`, `RTN event retry/quarantine`, ainsi que les actions
  specialisees `reporting` et `signature`, y compris la serialisation des
  fichiers binaires HVS. Un deuxieme defaut reel de serialisation JSON a ete
  corrige sur les actions RTN pour ne plus envoyer les arguments positionnels
  au backend.

Ordre d'execution recommande:

1. [x] Lot 4A
2. [x] Lot 4B
3. [x] Lot 4C
4. [x] Rejouer linter + tests CLI complets

### Etape 5. Durcir l'observabilite et les statuts operateur

Objectif:

- aligner les logs, statuts et messages avec les specs fonctionnelles,
  techniques et d'architecture;
- rendre l'exploitation lisible sans debugger le code.

Fichiers cibles:

- `pkg/protocols/modules/ebics/server.go`
- `pkg/protocols/modules/ebics/client.go`
- `pkg/protocols/modules/ebics/client_transfer.go`
- `pkg/protocols/modules/ebics/client_admin.go`
- `pkg/protocols/modules/ebics/client_reporting.go`
- `pkg/protocols/modules/ebics/client_key_rotation.go`
- `pkg/admin/rest/ebics_*.go`
- `pkg/cmd/client/ebics_*.go`

Points a verifier:

- correlation `HostID / PartnerID / UserID / OrderType / TransactionID`
- restitution separee des return codes `technical` et `business`
- coherence des messages REST / CLI / logs
- statuts d'initialisation, rotation et RTN exploitables

Commande qualite minimale:

- `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
- `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

Sous-lots cochables:

- [x] Lot 5A - Normaliser les correlations et statuts EBICS
  Fichiers principaux: `pkg/protocols/modules/ebics/server.go`,
  `pkg/protocols/modules/ebics/client.go`,
  `pkg/protocols/modules/ebics/client_transfer.go`
  Attendus: correlation `HostID / PartnerID / UserID / OrderType / TransactionID`
  visible et statuts operateur coherents
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

  2026-04-01: Lot 5A est maintenant ferme pour une premiere tranche
  d'observabilite. Le detail de transaction REST/CLI expose maintenant
  `HostID`, `PartnerID`, `UserID`, `RequestID` et `CorrelationID`, et la CLI
  affiche aussi le `TransferID` archive quand il n'est plus porte directement
  par l'operation. Cela renforce la correlation operateur sans changer les
  modeles de persistance.

- [x] Lot 5B - Rendre explicites les return codes et messages operateur
  Fichiers principaux: `pkg/protocols/modules/ebics/client_admin.go`,
  `pkg/protocols/modules/ebics/client_reporting.go`,
  `pkg/protocols/modules/ebics/client_key_rotation.go`,
  `pkg/admin/rest/ebics_*.go`,
  `pkg/cmd/client/ebics_*.go`
  Attendus: restitution separee des return codes `technical` et `business`,
  coherence des messages REST / CLI / logs
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

  2026-04-01: Lot 5B est maintenant ferme. La CLI EBICS affiche
  explicitement les codes et messages `technical` / `business`, ce qui aligne
  enfin la lecture operateur avec les donnees deja exposees par le backend REST
  et stockees dans `EbicsOperation`.

- [x] Lot 5C - Rendre exploitables les workflows sensibles et RTN
  Fichiers principaux: `pkg/protocols/modules/ebics/client_admin.go`,
  `pkg/protocols/modules/ebics/client_key_rotation.go`,
  `pkg/admin/rest/ebics_initializations.go`,
  `pkg/admin/rest/ebics_key_lifecycles.go`,
  `pkg/admin/rest/ebics_rtn.go`
  Attendus: statuts d'initialisation, rotation et RTN lisibles sans debugger le
  code
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

  2026-04-01: Lot 5C est maintenant ferme. Les sorties REST/CLI exposent
  desormais l'`evidence` structuree pour les `key lifecycles` et les
  `initializations`, ainsi que les metadonnees d'action operateur RTN
  (`operatorAction`, `operatorReason`, `operatorMetadata`). Les workflows
  sensibles deviennent lisibles sans inspection directe de la base.

Ordre d'execution recommande:

1. [x] Lot 5A
2. [x] Lot 5B
3. [x] Lot 5C
4. [x] Rejouer linter + tests observabilite

### Etape 6. Fermer les exigences d'exploitation transverses

Objectif:

- solder les points de retention, protections d'etat et discipline
  multi-SGBD/XORM.

Fichiers cibles:

- `pkg/model/credentials.go`
- `pkg/model/ebics_key_lifecycle.go`
- `pkg/model/ebics_initialization_workflow.go`
- `pkg/model/ebics_rtn_event.go`
- `pkg/model/ebics_nonce.go`
- `pkg/database/migrations/*.go`

Tests a ajouter ou enrichir en priorite:

- tests de protections de mutation/suppression
- tests de contraintes de persistance et migrations
- tests de purge / retention sur `nonces`, transactions et RTN

Commande qualite minimale:

- `golangci-lint run ./pkg/model ./pkg/database/migrations ./pkg/backup`
- `go test ./pkg/model ./pkg/database/migrations ./pkg/backup`

Sous-lots cochables:

- [x] Lot 6A - Durcir les protections de mutation / suppression
  Fichiers principaux: `pkg/model/credentials.go`,
  `pkg/model/ebics_key_lifecycle.go`,
  `pkg/model/ebics_initialization_workflow.go`,
  `pkg/model/ebics_rtn_event.go`
  Attendus: tests de protections de mutation/suppression sur objets sensibles
  Validation: `golangci-lint run ./pkg/model ./pkg/database/migrations ./pkg/backup`
  puis `go test ./pkg/model ./pkg/database/migrations ./pkg/backup`

  2026-04-01: Lot 6A est maintenant ferme. La protection existante sur
  `Credential` reference par un lifecycle actif est couverte par des tests
  dedies, et des hooks `BeforeDelete` empechent desormais la suppression
  directe de `key lifecycles` et `initializations` encore actifs.

- [x] Lot 6B - Fermer la discipline multi-SGBD / XORM et migrations
  Fichiers principaux: `pkg/database/migrations/*.go`,
  `pkg/model/ebics_nonce.go`
  Attendus: tests de contraintes de persistance, migrations et comportements
  cross-SGBD sur le perimetre EBICS
  Validation: `golangci-lint run ./pkg/model ./pkg/database/migrations ./pkg/backup`
  puis `go test ./pkg/model ./pkg/database/migrations ./pkg/backup`

  2026-04-01: Lot 6B est maintenant ferme. La migration `0.16.0` est
  desormais couverte par `pkg/database/migrations/0.16.0_test.go` avec
  verification de la creation/reversion des tables EBICS, de l'unicite
  effective sur `RTN events`, `transactions`, `segments` et `nonces`, ainsi
  que de la cascade transaction -> segments. En complement, le modele
  `EbicsRTNEvent` est maintenant verrouille par des tests dedies sur l'unicite
  de `idempotenceKey` et sur la coherence `host/subscriber`.

- [x] Lot 6C - Poser la retention / purge minimale EBICS
  Fichiers principaux: `pkg/model/ebics_nonce.go`,
  `pkg/model/ebics_rtn_event.go`,
  `pkg/model/ebics_transaction.go`
  Attendus: tests de purge / retention sur `nonces`, transactions et RTN
  Validation: `golangci-lint run ./pkg/model ./pkg/database/migrations ./pkg/backup`
  puis `go test ./pkg/model ./pkg/database/migrations ./pkg/backup`

  2026-04-01: Lot 6C est maintenant ferme. Une retention minimale explicite
  existe desormais dans `pkg/model/ebics_retention.go` pour les `nonces`,
  `transactions` et `RTN events`, avec purge stricte avant cutoff et
  conservation des evenements RTN non terminaux. La couverture associee dans
  `pkg/model/ebics_retention_test.go` verrouille les cas
  `ancien vs plus recent` et la preservation des statuts RTN encore
  exploitables.

Ordre d'execution recommande:

1. [x] Lot 6A
2. [x] Lot 6B
3. [x] Lot 6C
4. [x] Rejouer linter + tests transverses

### Etape 7. Passe de sortie B5

Objectif:

- prononcer ou refuser explicitement la gate "backend pret frontend".

Verification attendue:

- repasse `rg ErrNotImplemented|not implemented` sur le perimetre EBICS
- relecture contre `specifications-fonctionnelles.md`
- relecture contre `specifications-techniques.md`
- relecture contre `architecture-logicielle.md`
- verification explicite des attentes de passe-plat metier et connecteurs
- verification explicite de la couverture de tests EBICS ajoutee pendant `B4`

Commande qualite minimale:

- `golangci-lint run ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client ./cmd/waarp-gateway ./pkg/backup ./pkg/database/migrations`
- `go test ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/admin/rest/api ./pkg/cmd/client ./cmd/waarp-gateway ./pkg/backup ./pkg/database/migrations`

Sous-lots cochables:

- [x] Lot 7A - Rejouer la passe "zero stub bloquant"
  Attendus: repasse `rg ErrNotImplemented|not implemented` sur le perimetre
  EBICS et solder tous les cas restants
  Validation: commande de recherche rejouee et conclusion tracee dans le suivi

  2026-04-01: Lot 7A est maintenant ferme. La repasse
  `rg ErrNotImplemented|not implemented` sur `pkg/protocols/modules/ebics`,
  `pkg/admin/rest`, `pkg/admin/rest/api`, `pkg/cmd/client`,
  `cmd/waarp-gateway`, `pkg/model`, `pkg/backup` et
  `pkg/database/migrations` ne remonte aucun stub EBICS bloquant actif.

- [x] Lot 7B - Rejouer la passe qualite complete
  Attendus: linter complet backend EBICS puis tests cibles / non-regression
  Validation: `golangci-lint run ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client ./cmd/waarp-gateway ./pkg/backup ./pkg/database/migrations`
  puis `go test ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/admin/rest/api ./pkg/cmd/client ./cmd/waarp-gateway ./pkg/backup ./pkg/database/migrations`

  2026-04-01: Lot 7B est maintenant ferme. La passe qualite backend EBICS a
  ete rejouee avec succes:
  `golangci-lint run ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client ./cmd/waarp-gateway ./pkg/backup ./pkg/database/migrations`
  puis
  `go test ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/admin/rest/api ./pkg/cmd/client ./cmd/waarp-gateway ./pkg/backup ./pkg/database/migrations`.

- [x] Lot 7C - Relecture finale contre les specs et les suivis
  Attendus: relecture contre `specifications-fonctionnelles.md`,
  `specifications-techniques.md`, `architecture-logicielle.md`, verification
  explicite des attentes de passe-plat metier, connecteurs et couverture de
  tests EBICS ajoutee pendant `B4`
  Validation: synthese de sortie documentee dans le suivi

  2026-04-01: Lot 7C est maintenant ferme. La relecture contre les specs et
  l'architecture confirme que le backend EBICS proprement dit est fortement
  consolide sur l'execution protocolaire, la persistance, l'administration,
  l'observabilite et la couverture de tests ajoutee pendant `B4`. En
  revanche, la cible documentaire complete inclut explicitement une couche de
  passe-plat metier et des connecteurs asynchrones `AMQP 0.9.1` / `AMQP 1.0`
  comme attente minimale ou prealable architectural. Arbitrage retenu:
  ces capacites doivent etre traitees comme protocoles Gateway autonomes, hors
  perimetre EBICS strict, mais restent un pre-requis imperatif pour le futur
  chantier de passe-plat metier. La conclusion de sortie reste donc negative a
  l'echelle de la cible documentaire globale, meme si le backend EBICS strict
  est beaucoup plus mature.

- [x] Lot 7D - Prononcer ou refuser la gate "backend pret frontend"
  Attendus: decision explicite, motivee, tracee dans les documents de suivi
  Validation: mise a jour des cases de sortie backend

  2026-04-01: Lot 7D est maintenant ferme avec une decision explicite de
  refus de la gate `backend pret frontend` a l'echelle de la cible
  documentaire complete. Motifs principaux:
  absence des protocoles natifs `amqp091` / `amqp10` et du socle
  `outbox / consumer workers` pour le passe-plat asynchrone metier, ainsi que
  l'ecart restant entre le backend EBICS strict et l'architecture cible
  documentee. Arbitrage retenu: ces sujets AMQP/passe-plat sont hors
  perimetre EBICS strict mais restent des pre-requis imperatifs du chantier
  metier cible. Le backend EBICS seul peut etre considere tres avance et
  techniquement consolide, mais pas encore conforme au perimetre complet
  annonce par les specs.

Ordre d'execution recommande:

1. [x] Lot 7A
2. [x] Lot 7B
3. [x] Lot 7C
4. [x] Lot 7D

## 11. Backlog executable apres B5

Objectif:

- sortir du mode "analyse libre" sur les restes a faire post-`B5`;
- piloter explicitement les chantiers `P0`, `P2` et `P3`;
- rappeler que `AMQP 0.9.1` / `AMQP 1.0` restent hors perimetre EBICS strict,
  tout en etant des pre-requis du futur passe-plat metier.

Regles:

- ne cocher un lot qu'apres dev, linter et tests du perimetre touche;
- tracer la limite residuelle si un lot est seulement partiellement ferme;
- ne pas fermer `P0` sans preuve de bout en bout sur le chemin
  `RTN -> Transfer planifie -> execution client -> payload final`.

### Chantier P0 - RTN reel jusqu'au payload final

Objectif:

- fermer le dernier ecart entre orchestration RTN et execution reelle du pull;
- prouver que l'auto-pull RTN s'appuie sur le moteur natif des `Transfer`
  Gateway, sans chemin parallele fragile.

Fichiers cibles:

- `pkg/protocols/modules/ebics/rtn_service.go`
- `pkg/protocols/modules/ebics/rtn_service_test.go`
- `pkg/protocols/modules/ebics/client_transfer.go`
- `pkg/controller/client_transfers.go`
- `pkg/pipeline/client.go`

Commande qualite minimale:

- `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`
- `go test ./pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`

Sous-lots cochables:

- [x] Lot P0A - Valider la programmation du `Transfer` RTN dans le moteur Gateway
  Attendus: un evenement RTN auto-pull cree un `Transfer` `PLANNED`
  immediat, un `EbicsOperation` `AUTO_TRIGGERED` en attente d'execution,
  et les correlations `event / operation / transfer` sont persistantes et
  lisibles cote operateur
  Validation: `go test ./pkg/protocols/modules/ebics -run "TestRTNService" -v`

  2026-04-01: Lot P0A est maintenant ferme. `TestRTNService` confirme qu'un
  evenement RTN `AUTO` cree un `Transfer` Gateway `PLANNED` immediat,
  une `EbicsOperation` `AUTO_TRIGGERED` en statut
  `WAITING_PAYLOAD_TRANSFER`, et les liens `RTN event -> operation ->
  transfer` sont bien persistants et relisibles.

- [x] Lot P0B - Fermer le scenario de bout en bout `RTN -> BTD -> payload final`
  Attendus: le scheduler/controller reprend bien le `Transfer` planifie,
  le runtime client EBICS reutilise l'operation pre-creee, le `BTD` aboutit
  au payload final attendu, et les statuts finaux `event / operation / transfer`
  sont coherents
  Validation: `go test ./pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`
  2026-04-01: jalon intermediaire pose via `TestTransferClientCreateOperationReusesScheduledOperation`.
  Le chemin `Transfer planifie -> runtime client -> reutilisation de l'operation
  RTN pre-creee` est maintenant couvert, avec conservation des liens
  `RTN event / operation / transfer` et absence de duplication d'`EbicsOperation`.
  Le lot reste ouvert tant qu'un vrai scenario `scheduler -> BTD -> payload final`
  n'est pas valide de bout en bout.
  2026-04-01: une integration `RTN -> controller -> client EBICS -> serveur EBICS`
  est maintenant posee dans `rtn_controller_integration_test.go`, avec deux
  constats techniques explicites:
  `server.go` devait fournir un verifieur XMLDSig au handler `lib-ebics`,
  faute de quoi toute requete signee tombait en `091304 EBICS_SIGNER_UNKNOWN`;
  ce trou runtime est maintenant corrige via un verifieur adosse au
  `providerStore`.
  2026-04-01: `Lot P0B` est maintenant ferme. Le vrai scenario
  `RTN -> Transfer planifie -> controller -> ClientPipeline -> client EBICS HTTP
  -> serveur EBICS HTTP -> payload final` passe desormais en vert dans
  `rtn_controller_integration_test.go`.
  Les causes profondes corrigees sur ce maillon final sont:
  generation a tort d'un `TransactionID` synthetique pour `BTD`, ce qui forcait
  `lib-ebics` a entrer en phase `Transfer` sans segment;
  absence de persistance du vrai `TransactionID` de download lors de la reponse
  banque;
  reutilisation manquee de l'operation RTN planifiee quand
  `TransferInfo["ebicsOperationID"]` etait relu comme `json.Number`;
  et surtout court-circuit de `EndTransfer()` cote client EBICS, car
  `completeSuccess()` marquait le client comme `finished` trop tot, ce qui
  empechait la preservation du lien d'historique `archivedTransferID`.
  Verification rejouee:
  `go test ./pkg/protocols/modules/ebics -run "TestRTNControllerExecutesScheduledBTDToFinalPayload" -v -count=1 -timeout 10m`
  puis
  `go test ./pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`.
  Point d'attention explicite: plusieurs tests EBICS actuels restent des
  tests unitaires ou semi-integres qui appellent le routeur payload ou les
  services directement (`order_router_test.go`, `server_integration_test.go`,
  `rtn_service_test.go`, une partie de `client_transfer_test.go`).
  Ils restent utiles, mais ne suffisent pas a prouver les vrais chemins
  de production `RTN -> controller -> pipeline -> client EBICS HTTP ->
  serveur EBICS HTTP -> persistence/historique`.
  Le lot `P0B` doit donc etre lu comme la fermeture d'un vrai test
  d'integration de production, et pas seulement d'un empilement de tests
  unitaires de composants.

- [x] Lot P0C - Durcir reprise, retry et observabilite operateur sur RTN reel
  Attendus: echec retryable correctement remonte, reprogrammation lisible,
  recovery sans duplication parasite, erreurs et transitions visibles en
  REST/CLI pour `RTN`, `operation` et `transfer`
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

  2026-04-01: le statut RTN n'est plus marque `PROCESSED` au simple moment de
  la programmation du `Transfer`. L'evenement reste maintenant `PROCESSING`
  tant que l'auto-pull n'a pas reellement termine, puis est synchronise avec
  l'issue finale de l'`EbicsOperation` cote client:
  succes -> `PROCESSED`, echec retryable -> `RETRYABLE`, echec terminal ->
  `FAILED`.
  Les liens et statuts d'auto-pull (`operation`, `transfer`, `orderType`,
  `status`, `outcome`, `retry`) sont exposes en REST/CLI.
  Un defaut runtime connexe a aussi ete corrige cote client EBICS:
  les decisions de retry calculees a partir des return codes EBICS ne sont plus
  ecrasees systematiquement en `TECHNICAL_FATAL_FAILURE` dans le chemin
  d'erreur pipeline.
  Verification rejouee:
  `go test ./pkg/protocols/modules/ebics -run "Test(TransferClient|RTNService|RTNController)" -v -count=1 -timeout 10m`,
  `go test ./pkg/admin/rest ./pkg/admin/rest/api -run 'Test(ActOnEbicsRTNEvent|GetEbicsRTNEvent)' -v -count=1`,
  `go test ./pkg/cmd/client ./pkg/gatewayd ./pkg/model -count=1`.
  `golangci-lint` reste bloque uniquement par la contrainte d'environnement
  Windows deja identifiee (`can't eval symlinks on wd ... Access is denied`),
  y compris lorsqu'il est relance sur le vrai chemin du depot.

Ordre d'execution recommande:

1. [x] Lot P0A
2. [x] Lot P0B
3. [x] Lot P0C

### Chantier P4 - Remise en ordre architecturale `TransferInfo` / EBICS

Objectif:

- supprimer le double usage de `TransferInfo` comme espace metier exploitable
  et comme support de correlation technique EBICS;
- remettre la correlation structurelle EBICS dans une persistance dediee et
  stable, conforme aux principes d'architecture du projet;
- preparer, si necessaire, un redeveloppement partiel du runtime EBICS pour
  revenir a un standard architectural propre.

Positionnement:

- ce chantier demarre apres la fermeture de `P0C`;
- il est considere prioritaire avant tout elargissement fonctionnel EBICS
  supplementaire non critique.

Constat de depart:

- `TransferInfo` est historiquement reutilisable par l'exploitant et expose en
  variables `#TI_*#`;
- l'implementation EBICS actuelle y stocke encore des donnees de correlation
  technique (`ebicsOperationID`, `ebicsTransactionID`, identite EBICS,
  service/profil, hints RTN), ce qui cree une fuite de details internes dans un
  espace semi-public et un couplage runtime sur une structure souple;
- ce point est contraire aux principes documentes dans les specs et notes
  d'architecture EBICS, qui imposent de ne pas faire de `TransferInfo` la cle
  de voute des relations structurelles.

Fichiers cibles:

- `pkg/protocols/modules/ebics/client_transfer.go`
- `pkg/protocols/modules/ebics/order_router.go`
- `pkg/protocols/modules/ebics/rtn_service.go`
- `pkg/model/ebics_operation.go`
- `pkg/model/ebics_transaction.go`
- migrations SQL EBICS a completer si de nouveaux champs ou tables de liaison
  sont necessaires
- surfaces REST/CLI qui lisent aujourd'hui des correlations via `TransferInfo`
- documentation d'architecture et backlog EBICS associes

Commande qualite minimale:

- `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client ./pkg/model ./pkg/database/migrations`
- `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/admin/rest/api ./pkg/cmd/client ./pkg/model ./pkg/database/migrations`

Sous-lots cochables:

- [x] Lot P4A - Cartographier et classer toutes les cles EBICS actuellement stockees dans `TransferInfo`
  Attendus: inventaire exhaustif des cles, qualification `metier exploitable /
  confort local / correlation structurelle / detail runtime transitoire`,
  et cible de relogement pour chaque cle
  Validation: synthese documentee et relue

  2026-04-01: l'inventaire exhaustif des usages actuels est etabli.
  Sources reelles d'ecriture/llecture relues:
  `pkg/protocols/modules/ebics/order_router.go`,
  `pkg/protocols/modules/ebics/client_transfer.go`,
  `pkg/protocols/modules/ebics/rtn_service.go`,
  `pkg/tasks/replacer.go`.
  Constat principal: l'implementation actuelle ne se limite pas a quelques
  cles EBICS dediees; le chemin RTN clone aussi le `PayloadMap` brut de
  l'evenement vers `TransferInfo`, ce qui ouvre une fuite structurelle
  potentiellement non bornee vers l'espace `#TI_*#`.

  Inventaire classe:

  `Correlation structurelle critique`
  - `ebicsOperationID`
    Ecriture: `order_router.go`, `client_transfer.go`, `rtn_service.go`
    Lecture: `client_transfer.go`, historique/tests
    Cible: relation primaire `ebics_operations.transfer_id` + fallback
    eventuel via table de liaison si necessaire, jamais via `TransferInfo`
  - `ebicsRTNEventID`
    Ecriture: `rtn_service.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_operations.rtn_event_id`, puis lecture via operation
    planifiee, pas via `TransferInfo`
  - `ebicsTransactionID`
    Ecriture: `client_transfer.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_operations.transaction_id` et `ebics_transactions`,
    jamais comme source primaire dans `TransferInfo`

  `Contexte protocolaire EBICS reconstituable ou deja porte ailleurs`
  - `ebicsOrderType`
    Ecriture: `order_router.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_operations.order_type`
  - `ebicsHostID`
    Ecriture: `order_router.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_hosts.host_id` / `ebics_operations.ebics_host_id`
  - `ebicsPartnerID`
    Ecriture: `order_router.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_subscribers.partner_id`
  - `ebicsUserID`
    Ecriture: `order_router.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_subscribers.user_id`
  - `ebicsRequestID`
    Ecriture: `order_router.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_operations.request_id`
  - `ebicsCorrelationID`
    Ecriture: `order_router.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_operations.correlation_id`
  - `ebicsProtocol`
    Ecriture: `order_router.go`
    Lecture: indirecte d'affichage/historique seulement
    Cible: `ebics_operations.ebics_version`
  - `ebicsService`
    Ecriture: `order_router.go`
    Lecture: surtout affichage / historique aujourd'hui
    Cible: `ebics_operations.metadata.service` ou table fille si
    exploitation REST/UI plus fine a terme
  - `ebicsProfile`
    Ecriture: `client_transfer.go`, `rtn_service.go`
    Lecture: `client_transfer.go`
    Cible: `ebics_operations.metadata.profileName` ou champ dedie si besoin
  - `ebicsEndpointURL`
    Ecriture: `client_transfer.go`
    Lecture: `client_transfer.go`
    Cible: metadata technique d'operation, pas `TransferInfo`

  `Metadonnees RTN techniques actuellement dupliquees dans TransferInfo`
  - `rtnProviderName`
    Ecriture: `rtn_service.go`
    Lecture: surtout observabilite/tests
    Cible: `ebics_operations.metadata.rtnProviderName` et `ebics_rtn_events`
  - `rtnSource`
    Ecriture: `rtn_service.go`
    Lecture: surtout observabilite/tests
    Cible: `ebics_operations.metadata.rtnSource` et `ebics_rtn_events.source`

  `Pass-through RTN brut actuellement deverse dans TransferInfo`
  - toutes les cles de `event.PayloadMap` clonees par
    `TransferInfo: maps.Clone(event.PayloadMap)` dans `rtn_service.go`
  - cela inclut au minimum, selon les chemins deja lus:
    `orderTypeHint`, `profileID`, `serviceName`, `serviceOption`, `scope`,
    `msgName`, `containerType`, `targetDirectory`, `requestID`, `ruleID`,
    `ruleName`, `clientName`, `srcFilename`, `fileName`, `remoteFilename`,
    `documentName`, `outputName`, `destFilename`, `targetFileName`,
    `providerName`, `autoPullPolicy`, `hostID`, `partnerID`, `userID`,
    `ebicsHostID`, `ebicsSubscriberID`
  - et potentiellement toute cle arbitraire apportee par le provider RTN
  Cible: rester dans `ebics_rtn_events.payload` et, si necessaire pour le
  runtime, etre recopies explicitement dans `ebics_operations.metadata`
  apres filtrage et normalisation; jamais en pass-through global vers
  `TransferInfo`

  Classification retenue pour la suite:
  - `TransferInfo` ne doit plus porter aucune correlation structurelle EBICS
    (`operation`, `transaction`, `event`, identite protocolaire, endpoint,
    resolution de service)
  - `TransferInfo` ne doit plus recevoir de clonage brut de metadata RTN
  - seules pourront rester a terme des metadonnees explicitement assumees
    comme visibles et reutilisables par l'exploitant

  Impact de conception confirme:
  - `P4B` doit definir un modele cible ou le runtime client/serveur/RTN peut
    retrouver toutes ses correlations sans aucune lecture critique de
    `TransferInfo`
  - `P4C` devra probablement introduire une couche de chargement de contexte
    technique EBICS a partir d'`EbicsOperation`, d'`EbicsTransaction` et de
    metadata operationnelles dediees

- [x] Lot P4B - Definir le modele cible de correlation EBICS hors `TransferInfo`
  Attendus: choix explicite des champs/tables/liaisons qui remplacent les
  usages actuels de `TransferInfo` pour `EbicsOperation`, `EbicsTransaction`,
  `RTN` et les metadonnees techniques de resolution
  Validation: mise a jour documentaire + migration ciblee si necessaire

  2026-04-01: le modele cible est maintenant fige.

  Principe directeur:
  - aucune correlation runtime critique EBICS ne doit dependre de
    `TransferInfo`
  - `TransferInfo` n'est plus un bus technique interne EBICS
  - aucun clonage brut de metadata RTN vers `TransferInfo` n'est autorise

  Modele cible retenu:

  `Correlation primaire`
  - `Transfer <-> EbicsOperation`
    porte par `ebics_operations.transfer_id`
  - `EbicsOperation <-> EbicsRTNEvent`
    porte par `ebics_operations.rtn_event_id`
  - `EbicsOperation <-> EbicsTransaction`
    porte par `ebics_transactions.ebics_operation_id`
  - `EbicsTransaction <-> Transfer`
    porte par `ebics_transactions.transfer_id`

  `Identifiants protocolaires et contexte d'execution`
  - `order_type`, `request_id`, `transaction_id`, `correlation_id`,
    `ebics_version`, `ebics_host_id`, `ebics_subscriber_id`
    restent portes par `ebics_operations`
  - les informations de resolution techniques non structurantes
    (`profileName`, `endpointURL`, service EBICS resolu, `rtnProviderName`,
    `rtnSource`, `autoPullReason`) vivent dans `ebics_operations.metadata`
  - le message RTN complet et ses metadata source restent portes par
    `ebics_rtn_events.payload`

  `Resolution RTN`
  - le runtime RTN prepare un `EbicsOperation` complet avant de creer le
    `Transfer`
  - le `Transfer` ne transporte plus les cles techniques EBICS necessaires a
    la reprise; le runtime recharge l'`EbicsOperation` liee au transfert puis
    derive depuis elle tout le contexte necessaire
  - les champs eventuellement necessaires a la selection de regle/client cote
    RTN restent sur l'evenement RTN ou sont recopies explicitement dans
    `EbicsOperation.Metadata`, jamais dans `TransferInfo`

  `Whitelisting TransferInfo`
  - politique par defaut: aucune cle EBICS interne n'est ecrite dans
    `TransferInfo`
  - exception possible a terme: quelques metadonnees explicitement assumees
    comme visibles et reutilisables par l'exploitant, apres decision
    explicite de whitelist
  - cette whitelist ne doit jamais inclure:
    `ebicsOperationID`, `ebicsRTNEventID`, `ebicsTransactionID`,
    `ebicsOrderType`, `ebicsHostID`, `ebicsPartnerID`, `ebicsUserID`,
    `ebicsRequestID`, `ebicsCorrelationID`, `ebicsProtocol`,
    `ebicsService`, `ebicsProfile`, `ebicsEndpointURL`,
    `rtnProviderName`, `rtnSource`, ni aucune cle issue du clonage brut
    `event.PayloadMap`

  `Regles de relogement`
  - `ebicsOperationID`
    supprime de `TransferInfo`; charge via `ebics_operations.transfer_id`
  - `ebicsRTNEventID`
    supprime de `TransferInfo`; charge via `ebics_operations.rtn_event_id`
  - `ebicsTransactionID`
    supprime de `TransferInfo`; charge via `ebics_operations.transaction_id`
    et `ebics_transactions`
  - identite EBICS (`host/partner/user`)
    supprimee de `TransferInfo`; rederivee depuis `ebics_operations` puis
    `ebics_subscribers` / `ebics_hosts`
  - `ebicsService`, `ebicsProfile`, `ebicsEndpointURL`
    deplaces dans `ebics_operations.metadata`
  - metadata RTN source
    conservees dans `ebics_rtn_events.payload`, avec recopie selective dans
    `ebics_operations.metadata` uniquement si necessaire au runtime ou a
    l'observabilite

  `Contraintes de refactor`
  - `P4C` devra introduire un chargement de contexte technique par
    `EbicsOperation` avant toute execution client EBICS
  - `P4D` devra poser une whitelist stricte de `TransferInfo` et verifier les
    variables `#TI_*#`
  - si une metadonnee devient necessaire en lecture SQL fine ou en UI, elle ne
    doit pas etre repliee par facilite dans `TransferInfo`; elle doit etre
    portee par `ebics_operations.metadata` ou par une table fille dediee

- [x] Lot P4C - Refactorer le runtime EBICS pour supprimer la dependance structurelle a `TransferInfo`
  Attendus: le code client, serveur payload et RTN n'utilise plus
  `TransferInfo` comme support principal de correlation technique; les lectures
  runtime critiques passent par la persistance dediee
  Validation: linter + tests unitaires et d'integration du perimetre touche
  2026-04-01: premiere tranche engagee et validee.
  Le runtime client payload charge maintenant l'operation liee en priorite via
  `ebics_operations.transfer_id`, ne depend plus de `TransferInfo` pour
  resoudre le subscriber, le profil, l'endpoint et les identifiants de
  correlation, et persiste `profileName` / `endpointURL` dans
  `ebics_operations.metadata`.
  Decision de chantier: aucun fallback de compatibilite interne EBICS n'est
  conserve a ce stade. Les lectures ou ecritures techniques EBICS doivent etre
  propres, structurelles et uniquement portees par les tables / metadata
  dediees.
  Le service RTN ne clone plus le `PayloadMap` brut dans `TransferInfo`.
  Un helper partage `EbicsTransferContext` charge desormais le contexte EBICS
  depuis les tables dediees; il est reutilise par les taches et par l'API
  `transfers`.
  La revue des chemins non payload confirme par ailleurs que les ordres
  `admin/reporting/key rotation/init` n'utilisent pas `TransferInfo` comme bus
  technique: le chantier `P4C` reste concentre sur les chemins payload et RTN.
  Les tests RTN confirment aussi que le residu standard Gateway `__followID__`
  peut rester present dans `TransferInfo` sans remettre en cause l'objectif du
  chantier: ce lot retire les cles techniques EBICS / RTN, pas les metadonnees
  natives du moteur deja portees par d'autres protocoles.
  2026-04-01: tranche de finalisation validee.
  Les lectures techniques EBICS residuelles ont ete retirees du runtime client
  payload et du chargement de contexte: plus de fallback `TransferInfo` pour
  `operation`, `event`, `transaction`, `profile`, `endpoint`, `request`,
  `correlation`, `host/partner/user`.
  Le runtime non payload (`admin/reporting/key rotation/init`) a ete relu et ne
  porte pas d'usage abusif de `TransferInfo`.
  Validation rejouee:
  - `go test ./pkg/protocols/modules/ebics/... -count=1`
  - `go test ./pkg/model -count=1`
  - `go test ./pkg/tasks -count=1`
  Le lot `P4C` est maintenant considere comme ferme.

- [x] Lot P4D - Preserver uniquement un `TransferInfo` metier / operateur propre
  Attendus: seules des metadonnees explicitement assumees comme visibles et
  reutilisables par l'exploitant restent dans `TransferInfo`; les cles
  techniques EBICS internes n'y figurent plus
  Rappel de conception: l'accessibilite operateur doit etre preservee via des
  canaux dedies, pas en repliant les donnees techniques dans `TransferInfo`.
  Cible:
  - variables de taches dediees `#EBICS_*#` chargees depuis
    `EbicsOperation` / `EbicsTransaction` / `EbicsRTNEvent`
  - bloc REST/CLI dedie de type `ebicsContext` / `ebicsLinks` sur les
    transferts, distinct de `transferInfo`
  Validation: tests REST/CLI/taches sur `TransferInfo`, verification des
  variables `#TI_*#` et des futures variables `#EBICS_*#`
  2026-04-01: premiere tranche engagee et validee.
  Les variables de taches `#EBICS_*#` sont maintenant disponibles dans
  `replacer.go`, et l'API `transfers` expose un bloc dedie `ebicsContext`
  distinct de `transferInfo`.
  La tache `Transfer` a aussi ete reverifiee vis-a-vis du contrat historique
  `FollowID`: la correlation multi-sauts native reste intacte, et un parent
  EBICS ne propage pas son contexte technique dedie dans le `TransferInfo` du
  transfert fils meme quand `copyInfo=true`.
  Le cadrage cible est maintenant explicite: `TransferInfo` peut conserver les
  metadonnees historiques standard du moteur Gateway comme `__followID__`, mais
  plus aucune cle de correlation EBICS / RTN ni aucun clonage brut de payload
  ou metadata provider.
  2026-04-01: tranche de fermeture validee.
  Les surfaces `history` REST/CLI exposent maintenant le meme bloc dedie
  `ebicsContext` que `transfers`, y compris apres archivage via la resolution
  `metadata.archivedTransferID`.
  Les residus de cles EBICS dans le code de production ont ete retires:
  `order_router.go` ne porte plus les anciennes constantes `transferInfoKey*`,
  conservees uniquement dans les tests de non-regression.
  La passe de verification ne remonte plus d'ecriture ou lecture technique
  EBICS active dans `TransferInfo`; seules restent les metadonnees natives du
  moteur Gateway comme `__followID__`.
  Validation rejouee:
  - `go test ./pkg/admin/rest -run "Test(FromHistoryWithDBIncludesEbicsContext|GetHistory|ListHistory)" -count=1`
  - `go test ./pkg/tasks -run "Test(ReplaceVarsAddsDedicatedEbicsVariables|TransferRun)" -count=1`
  - `go test ./pkg/cmd/client -run "TestDoesNotExist" -count=0`
  - `go test ./pkg/protocols/modules/ebics/... -count=1`
  Le lot `P4D` est maintenant considere comme ferme.

- [x] Lot P4E - Rejouer la non-regression complete apres remise en ordre
  Attendus: scenarios reels `payload client`, `payload serveur`,
  `RTN -> auto-pull -> payload final`, et verification de l'absence de derive
  sur l'historique, l'observabilite et les workflows sensibles
  Validation: linter + passe `go test` du perimetre EBICS consolide
  2026-04-01: passe de non-regression large rejouee et verte sur le perimetre
  consolide:
  - `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/admin/rest/api ./pkg/cmd/client ./pkg/model ./pkg/gatewayd -count=1`
  2026-04-01: le diagnostic linter est maintenant etabli.
  Le blocage ne venait pas du depot ni de `golangci-lint`, mais du shell
  sandboxe execute sous `DESKTOP-N3Q22LC\\CodexSandboxOffline`, qui n'avait pas
  acces a `C:\\Users\\driss\\.config\\git\\ignore` et provoquait aussi les
  erreurs `can't eval symlinks on wd ... Access is denied`.
  Hors sandbox, sous le vrai compte `DESKTOP-N3Q22LC\\driss`, `pwsh 7.6.0`,
  l'acces Git utilisateur et `golangci-lint` fonctionnent correctement.
  Commande canonique retenue pour la gate qualite locale:
  - `Set-Location C:\\MonProjet\\Waarp-Gateway`
  - `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model`
  2026-04-01: repasse linter ciblee hors sandbox validee sur le perimetre
  consolide:
  - `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client ./pkg/model ./pkg/gatewayd`
  Le lot `P4E` est maintenant considere comme ferme.

Ordre d'execution recommande:

1. [x] Lot P4A
2. [x] Lot P4B
3. [x] Lot P4C
4. [x] Lot P4D
5. [x] Lot P4E

### Chantier P2 - Automatisation d'exploitation native

Objectif:

- fermer les ecarts restants d'exploitation continue autour d'EBICS strict;
- eviter de laisser une retention ou une orchestration uniquement manuelle.

Fichiers cibles:

- `pkg/model/ebics_retention.go`
- `pkg/protocols/modules/ebics/rtn_service.go`
- `pkg/gatewayd/server.go`
- futurs jobs/services dedies a la retention ou au refresh

Commande qualite minimale:

- `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/gatewayd`
- `go test ./pkg/protocols/modules/ebics/... ./pkg/model ./pkg/gatewayd`

Sous-lots cochables:

- [ ] Lot P2A - Poser une retention automatisee minimale
  Attendus: purge automatique bornee des `nonces`, transactions anciennes et
  evenements RTN selon politique explicite et testee
  Validation: `go test ./pkg/model -run "Test(PurgeEbics|EbicsRTNEvent)" -v`

- [ ] Lot P2B - Poser l'orchestration planifiee native
  Attendus: refresh, retries et taches de maintenance critiques ne reposent
  plus uniquement sur une action manuelle ou un ordonnanceur externe non trace
  Validation: tests cibles du ou des services ajoutes

- [ ] Lot P2C - Completer la couverture de tests runtime encore faible
  Attendus: couverture mesurable sur le client EBICS direct, le runtime RTN
  reel et les chemins de reprise encore peu testes
  Validation: `go test ./pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`
  Criteres complementaires:
  au moins un scenario client payload doit passer par le vrai `controller`
  et le vrai `ClientPipeline`;
  au moins un scenario serveur payload doit passer par le vrai serveur HTTP
  EBICS, et pas seulement par `newPayloadOrderRouter(...).Upload/Download(...)`;
  au moins un scenario RTN auto-pull doit passer par `RTN -> Transfer planifie
  -> controller -> client EBICS -> serveur EBICS -> payload final`;
  les tests directs de fonctions/routage restent maintenus comme unitaires,
  mais ne peuvent plus, a eux seuls, servir de preuve de fermeture runtime.

- [ ] Lot P2D - Historiser nativement les ordres EBICS non payload
  Attendus: les ordres d'administration, d'initialisation, de gestion de cles
  et de reporting disposent d'un historique durable et interrogeable, analogue
  a l'historique natif des transferts Gateway, avec statuts, horodatages,
  codes retour et evidence operateur
  Validation: `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/model`

Ordre d'execution recommande:

1. [ ] Lot P2A
2. [ ] Lot P2B
3. [ ] Lot P2C
4. [ ] Lot P2D
5. [ ] Ne lancer les evolutions structurelles connexes qu'apres `P4`

### Chantier P3 - Workflow VEU et signature distribuee

Objectif:

- fermer le manque applicatif entre support protocolaire `HVE/HVS` et vrai
  workflow metier de validation multi-signataires.

Fichiers cibles:

- runtime/client EBICS de signatures
- REST/CLI `key lifecycle` / `initialization` / futurs workflows VEU
- documentation fonctionnelle de pilotage operateur

Commande qualite minimale:

- `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
- `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

Sous-lots cochables:

- [ ] Lot P3A - Cadrer le workflow VEU cible
  Attendus: etats, transitions, roles operateur et invariants documentes
  contre les specs fonctionnelles/techniques/architecturales
  Validation: mise a jour documentaire relue

- [ ] Lot P3B - Implementer le workflow VEU minimal exploitable
  Attendus: workflow borne de bout en bout, sans simple facade protocolaire,
  avec persistance, statuts et commandes operateur coherents
  Validation: linter + tests du perimetre touche

- [ ] Lot P3C - Completer observabilite et non-regression VEU
  Attendus: REST/CLI et tests de non-regression couvrent les cas nominaux,
  rejets et reprises principaux
  Validation: linter + tests du perimetre touche

Ordre d'execution recommande:

1. [ ] Lot P3A
2. [ ] Lot P3B
3. [ ] Lot P3C

### Hors perimetre EBICS strict mais pre-requis metier

- [ ] `AMQP 0.9.1` implemente comme protocole Gateway autonome
- [ ] `AMQP 1.0` implemente comme protocole Gateway autonome
- [ ] chantier `passe-plat metier` ouvert sur ces protocoles et sur les autres
  connecteurs Gateway cibles
