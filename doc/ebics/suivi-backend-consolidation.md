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

### Bloquants frontend

- Valider l'execution serveur EBICS reelle sur le chemin nominal `BTU/BTD`
- Valider la couverture normative des ordres serveur non payload exposes
- Valider la segmentation / reprise / recovery cote serveur
- Solder la gate "plus aucun endpoint/commande EBICS expose sans logique runtime suffisante"
- Rejouer la passe de sortie backend `B5` avant de prononcer la gate frontend
- Rejouer la lecture de sortie backend au regard des specs fonctionnelles,
  techniques et d'architecture, pas seulement du code courant et des suivis

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

- [ ] Lot 4B - Couvrir les commandes CLI contract views / key lifecycles / initializations
  Fichiers principaux: `pkg/cmd/client/ebics_contract_views.go`,
  `pkg/cmd/client/ebics_key_lifecycles.go`,
  `pkg/cmd/client/ebics_initializations.go`
  Attendus: tests des actions et sorties operateur, coherence avec le backend
  REST expose
  Validation: `golangci-lint run ./pkg/cmd/client ./cmd/waarp-gateway`
  puis `go test ./pkg/cmd/client ./cmd/waarp-gateway`

- [ ] Lot 4C - Couvrir les commandes CLI RTN / actions specialisees
  Fichiers principaux: `pkg/cmd/client/ebics_rtn.go`,
  `pkg/cmd/client/ebics_payload_profiles.go`
  Attendus: tests des actions specialisees `reporting`, `signature`, `retry`,
  `recover` et messages utilisateur associes
  Validation: `golangci-lint run ./pkg/cmd/client ./cmd/waarp-gateway`
  puis `go test ./pkg/cmd/client ./cmd/waarp-gateway`

Ordre d'execution recommande:

1. [x] Lot 4A
2. [ ] Lot 4B
3. [ ] Lot 4C
4. [ ] Rejouer linter + tests CLI complets

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

- [ ] Lot 5A - Normaliser les correlations et statuts EBICS
  Fichiers principaux: `pkg/protocols/modules/ebics/server.go`,
  `pkg/protocols/modules/ebics/client.go`,
  `pkg/protocols/modules/ebics/client_transfer.go`
  Attendus: correlation `HostID / PartnerID / UserID / OrderType / TransactionID`
  visible et statuts operateur coherents
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

- [ ] Lot 5B - Rendre explicites les return codes et messages operateur
  Fichiers principaux: `pkg/protocols/modules/ebics/client_admin.go`,
  `pkg/protocols/modules/ebics/client_reporting.go`,
  `pkg/protocols/modules/ebics/client_key_rotation.go`,
  `pkg/admin/rest/ebics_*.go`,
  `pkg/cmd/client/ebics_*.go`
  Attendus: restitution separee des return codes `technical` et `business`,
  coherence des messages REST / CLI / logs
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

- [ ] Lot 5C - Rendre exploitables les workflows sensibles et RTN
  Fichiers principaux: `pkg/protocols/modules/ebics/client_admin.go`,
  `pkg/protocols/modules/ebics/client_key_rotation.go`,
  `pkg/admin/rest/ebics_initializations.go`,
  `pkg/admin/rest/ebics_key_lifecycles.go`,
  `pkg/admin/rest/ebics_rtn.go`
  Attendus: statuts d'initialisation, rotation et RTN lisibles sans debugger le
  code
  Validation: `golangci-lint run ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client`
  puis `go test ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/cmd/client`

Ordre d'execution recommande:

1. [ ] Lot 5A
2. [ ] Lot 5B
3. [ ] Lot 5C
4. [ ] Rejouer linter + tests observabilite

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

- [ ] Lot 6A - Durcir les protections de mutation / suppression
  Fichiers principaux: `pkg/model/credentials.go`,
  `pkg/model/ebics_key_lifecycle.go`,
  `pkg/model/ebics_initialization_workflow.go`,
  `pkg/model/ebics_rtn_event.go`
  Attendus: tests de protections de mutation/suppression sur objets sensibles
  Validation: `golangci-lint run ./pkg/model ./pkg/database/migrations ./pkg/backup`
  puis `go test ./pkg/model ./pkg/database/migrations ./pkg/backup`

- [ ] Lot 6B - Fermer la discipline multi-SGBD / XORM et migrations
  Fichiers principaux: `pkg/database/migrations/*.go`,
  `pkg/model/ebics_nonce.go`
  Attendus: tests de contraintes de persistance, migrations et comportements
  cross-SGBD sur le perimetre EBICS
  Validation: `golangci-lint run ./pkg/model ./pkg/database/migrations ./pkg/backup`
  puis `go test ./pkg/model ./pkg/database/migrations ./pkg/backup`

- [ ] Lot 6C - Poser la retention / purge minimale EBICS
  Fichiers principaux: `pkg/model/ebics_nonce.go`,
  `pkg/model/ebics_rtn_event.go`,
  `pkg/model/ebics_transaction.go`
  Attendus: tests de purge / retention sur `nonces`, transactions et RTN
  Validation: `golangci-lint run ./pkg/model ./pkg/database/migrations ./pkg/backup`
  puis `go test ./pkg/model ./pkg/database/migrations ./pkg/backup`

Ordre d'execution recommande:

1. [ ] Lot 6A
2. [ ] Lot 6B
3. [ ] Lot 6C
4. [ ] Rejouer linter + tests transverses

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

- [ ] Lot 7A - Rejouer la passe "zero stub bloquant"
  Attendus: repasse `rg ErrNotImplemented|not implemented` sur le perimetre
  EBICS et solder tous les cas restants
  Validation: commande de recherche rejouee et conclusion tracee dans le suivi

- [ ] Lot 7B - Rejouer la passe qualite complete
  Attendus: linter complet backend EBICS puis tests cibles / non-regression
  Validation: `golangci-lint run ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/admin/rest/... ./pkg/cmd/client ./cmd/waarp-gateway ./pkg/backup ./pkg/database/migrations`
  puis `go test ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/admin/rest ./pkg/admin/rest/api ./pkg/cmd/client ./cmd/waarp-gateway ./pkg/backup ./pkg/database/migrations`

- [ ] Lot 7C - Relecture finale contre les specs et les suivis
  Attendus: relecture contre `specifications-fonctionnelles.md`,
  `specifications-techniques.md`, `architecture-logicielle.md`, verification
  explicite des attentes de passe-plat metier, connecteurs et couverture de
  tests EBICS ajoutee pendant `B4`
  Validation: synthese de sortie documentee dans le suivi

- [ ] Lot 7D - Prononcer ou refuser la gate "backend pret frontend"
  Attendus: decision explicite, motivee, tracee dans les documents de suivi
  Validation: mise a jour des cases de sortie backend

Ordre d'execution recommande:

1. [ ] Lot 7A
2. [ ] Lot 7B
3. [ ] Lot 7C
4. [ ] Lot 7D
