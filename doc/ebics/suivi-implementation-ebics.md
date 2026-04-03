# Suivi d'implementation EBICS

## 1. Mode d'emploi

Ce document sert de checklist de suivi pendant le developpement.

Regles:

- cocher `[x]` quand l'item est termine et relu;
- laisser `[ ]` si non commence;
- remplacer par `[-]` si volontairement differe;
- ajouter une courte note si une divergence par rapport aux specs est decidee.
- executer `golangci-lint` avant chaque compilation ou test Go cible;
- si le linter est bloque par un ecart d'outillage, tracer explicitement le blocage
  avant de poursuivre les verifications de compilation.

## 2. Socle Phase A

- [x] Ajouter les constantes EBICS dans `pkg/model/table_names.go`
- [x] Ajouter les appellations EBICS dans `pkg/model/display_names.go`
- [x] Creer `pkg/model/ebics_host.go`
- [x] Creer `pkg/model/ebics_subscriber.go`
- [x] Creer `pkg/model/ebics_bank_key.go`
- [x] Creer `pkg/protocols/modules/ebics/constants.go`
- [x] Creer `pkg/protocols/modules/ebics/errors.go`
- [x] Creer `pkg/protocols/modules/ebics/config.go`
- [x] Creer `pkg/protocols/modules/ebics/server.go`
- [x] Creer `pkg/protocols/modules/ebics/client.go`
- [x] Creer `pkg/protocols/modules/ebics/module.go`
- [x] Enregistrer `ebics` dans `pkg/protocols/modules.go`
- [x] Valider `updateconf` pour `ProtoConfig` EBICS

## 3. Payload / contrat Phase B

- [x] Ajouter les constantes contract view/payload profile dans `table_names.go`
- [x] Ajouter les appellations correspondantes dans `display_names.go`
- [x] Creer `pkg/model/ebics_contract_view.go`
- [x] Creer `pkg/model/ebics_contract_view_item.go`
- [x] Creer `pkg/model/ebics_payload_profile.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/payload_resolution.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/contract_validation.go`
- [x] Creer `pkg/admin/rest/api/ebics_payload_profiles.go`
- [x] Creer `pkg/admin/rest/api/ebics_contract_views.go`
- [x] Etendre `updateconf` pour payload profiles

## 4. Operations / transactions Phase C

- [x] Ajouter les constantes operations/transactions/segments dans `table_names.go`
- [x] Ajouter les appellations correspondantes dans `display_names.go`
- [x] Creer `pkg/model/ebics_operation.go`
- [x] Creer `pkg/model/ebics_transaction.go`
- [x] Creer `pkg/model/ebics_transaction_segment.go`
- [x] Creer `pkg/protocols/modules/ebics/stores/operation_store.go`
- [x] Creer `pkg/protocols/modules/ebics/stores/tx_store.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/operation_mapper.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/retry_policy.go`
- [x] Creer `pkg/admin/rest/api/ebics_payload_requests.go`
- [x] Creer `pkg/admin/rest/api/ebics_operations.go`
- [x] Creer `pkg/admin/rest/api/ebics_transactions.go`

## 5. Workflows sensibles Phase D

- [x] Ajouter les constantes lifecycle/init dans `table_names.go`
- [x] Ajouter les appellations correspondantes dans `display_names.go`
- [x] Creer `pkg/model/ebics_key_lifecycle.go`
- [x] Creer `pkg/model/ebics_initialization_workflow.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/key_lifecycle_runner.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/initialization_runner.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/signature_state.go`
- [x] Creer `pkg/admin/rest/api/ebics_key_lifecycles.go`
- [x] Creer `pkg/admin/rest/api/ebics_initializations.go`

## 6. RTN Phase E

- [x] Ajouter les constantes RTN dans `table_names.go`
- [x] Ajouter les appellations RTN dans `display_names.go`
- [x] Creer `pkg/model/ebics_rtn_event.go`
- [x] Creer `pkg/model/ebics_rtn_provider.go`
- [x] Creer `pkg/protocols/modules/ebics/rtn/provider.go`
- [x] Creer `pkg/protocols/modules/ebics/rtn/wss_provider.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/rtn_ingestion.go`
- [x] Creer `pkg/protocols/modules/ebics/runtime/rtn_autopull.go`
- [x] Creer `pkg/admin/rest/api/ebics_rtn.go`
- [x] Creer `pkg/cmd/client/ebics_rtn.go`

## 7. REST handlers

- [x] Creer `pkg/admin/rest/ebics_operations.go`
- [x] Creer `pkg/admin/rest/ebics_payload_profiles.go`
- [x] Creer `pkg/admin/rest/ebics_contract_views.go`
- [x] Creer `pkg/admin/rest/ebics_payloads.go`
- [x] Creer `pkg/admin/rest/ebics_key_lifecycles.go`
- [x] Creer `pkg/admin/rest/ebics_initializations.go`
- [x] Creer `pkg/admin/rest/ebics_rtn.go`
- [x] Etendre `pkg/admin/rest/router.go`

## 8. CLI

- [x] Creer `pkg/cmd/client/ebics_operations.go`
- [x] Creer `pkg/cmd/client/ebics_payload.go`
- [x] Creer `pkg/cmd/client/ebics_payload_profiles.go`
- [x] Creer `pkg/cmd/client/ebics_contract_views.go`
- [x] Creer `pkg/cmd/client/ebics_key_lifecycles.go`
- [x] Creer `pkg/cmd/client/ebics_initializations.go`
- [x] Creer `pkg/cmd/client/ebics_rtn.go`

## 9. Import / export / administration transverse

- [x] Etendre `pkg/backup/export.go`
- [x] Etendre `pkg/backup/import.go`
- [x] Ajouter les helpers `*_export.go` / `*_import.go` si necessaire
- [x] Verifier le round-trip JSON/YAML de tous les objets administres

## 10. Points de controle obligatoires

- [ ] Respect strict de la contrainte multi-SGBD / XORM
- [ ] Aucun `returnCode` EBICS agrege en un seul champ
- [ ] Aucun glissement metier dans les workflows techniques
- [ ] `Transfer` reserve aux flux reellement fichier
- [ ] `Credential` reste generique, `EbicsKeyLifecycle` gouverne
- [ ] RTN reste decouple du transport `WSS`

## 11. Notes d'avancement

- Date: 2026-03-26
- Auteur: Codex
- Discipline: tout changement code EBICS doit desormais rejouer
  systematiquement `golangci-lint` avant compilation ou tests Go cibles, puis
  les tests unitaires du perimetre touche et une passe de non-regression
  backend EBICS avant cloture
- Blocage: aucun blocage actif sur la boucle `linter -> compilation` de la `Phase A`
- Decision: `updateconf` ne demande pas de code specifique pour EBICS en `Phase A`, car le round-trip `ProtoConfig`
  est deja assure generiquement par `pkg/backup` et la validation protocolaire passe ensuite par les `BeforeWrite`
- Verification: `golangci-lint run ./pkg/model ./pkg/protocols/...` passe apres reinstallation du binaire avec `go1.26.1`
- Verification: `go test ./pkg/model`, `go test ./pkg/protocols`, `go test ./pkg/protocols/modules/ebics`
  passent apres la passe linter
- Verification: `Phase B` validee avec `golangci-lint run ./pkg/model ./pkg/protocols/... ./pkg/admin/rest/api ./pkg/backup/...`
- Verification: `go test ./pkg/model`, `./pkg/protocols/modules/ebics/...`, `./pkg/admin/rest/api`, `./pkg/backup/...`
  passent apres la passe linter de la `Phase B`
- Verification: `Phase C` validee avec `golangci-lint run ./pkg/model ./pkg/protocols/... ./pkg/admin/rest/api`
- Verification: `go test ./pkg/model`, `./pkg/protocols/modules/ebics/...` et `./pkg/admin/rest/api`
  passent apres la passe linter de la `Phase C`
- Verification: `Phase D` validee avec `golangci-lint run ./pkg/model ./pkg/protocols/... ./pkg/admin/rest/api`
- Verification: `go test ./pkg/model`, `./pkg/protocols/modules/ebics/...` et `./pkg/admin/rest/api`
  passent apres la passe linter de la `Phase D`
- Verification: `Phase E` validee avec `golangci-lint run ./pkg/model ./pkg/protocols/... ./pkg/admin/rest/api ./pkg/cmd/client`
- Verification: `go test ./pkg/model`, `./pkg/protocols/modules/ebics/...`, `./pkg/admin/rest/api`
  et `./pkg/cmd/client` passent apres la passe linter de la `Phase E`
- Verification: socle REST EBICS valide avec `golangci-lint run ./pkg/admin/rest/... ./pkg/admin/rest/api`
- Verification: `go test ./pkg/admin/rest` et `./pkg/admin/rest/api` passent apres la passe linter REST
- Verification: CLI EBICS validee avec `golangci-lint run ./pkg/cmd/client ./cmd/waarp-gateway`
- Verification: `go test ./pkg/cmd/client` et `./cmd/waarp-gateway` passent apres la passe linter CLI
- Verification: integration `lib-ebics` Gateway validee avec `golangci-lint run ./pkg/model ./pkg/protocols/modules/ebics/... ./pkg/cmd/client ./cmd/waarp-gateway`
- Verification: `go test ./pkg/model`, `./pkg/protocols/modules/ebics/...`, `./pkg/cmd/client` et `./cmd/waarp-gateway`
  passent apres suppression du `replace` local et bascule sur `code.waarp.fr/lib/ebics@v0.0.0-20260326163504-771a9550a5be`
- Decision: la protection des `Credential` references par un lifecycle actif est posee directement dans
  `pkg/model/credentials.go` pour couvrir uniformement REST, GUI et CLI
- Verification: `Lot B1` payload client valide avec `golangci-lint run ./pkg/protocols/modules/ebics ./pkg/model`
- Verification: `go test ./pkg/protocols/modules/ebics`, `./pkg/model` et `./pkg/protocols/...`
  passent apres remplacement du stub `InitTransfer` par une execution cliente reelle `BTU/BTD`
  avec `EbicsOperation`, `EbicsTransaction`, contrat actif, TLS et recovery

## 12. Suite backend

Le suivi des prochaines etapes backend avant frontend est desormais porte par:

- `backend-consolidation-plan.md`
- `suivi-backend-consolidation.md`

Point de situation:

- 2026-03-31: `B1`, `B2`, `B3` et `B3.5` restent fermes sur le depot;
  `B4` et `B5` restent ouverts.
  Les ecarts restants sont maintenant priorises dans
  `suivi-backend-consolidation.md`, avec un focus explicite sur le durcissement
  serveur EBICS, l'observabilite et la verification finale avant frontend.
- 2026-03-31: la couverture de tests EBICS reste encore partielle au regard du
  perimetre implemente.
  Les tests visibles couvrent surtout `pkg/backup`, `updateconf` et
  `runtime/contract_validation`.
  Il manque encore une couverture plus directe du client EBICS, du serveur
  EBICS, des handlers REST EBICS et de la CLI EBICS.
- 2026-03-31: l'analyse d'avancement doit maintenant etre relue au regard des
  documents `specifications-fonctionnelles.md`,
  `specifications-techniques.md` et `architecture-logicielle.md`, car les
  attentes sur le passe-plat metier, les connecteurs standards et la lecture
  globale de l'architecture cible restent encore seulement partiellement
  refletees dans le statut courant.
- 2026-04-01: la passe finale `B5` est maintenant rejouee avec succes sur le
  perimetre backend EBICS (`rg`, `golangci-lint`, `go test`, relecture specs).
  La conclusion globale reste toutefois negative pour la gate
  `backend pret frontend` a l'echelle de la cible documentaire complete:
  le backend EBICS strict est consolide, mais les protocoles natifs
  `AMQP 0.9.1` / `AMQP 1.0` et le socle de passe-plat asynchrone metier
  restent absents. Arbitrage retenu: ces sujets AMQP/passe-plat sont hors
  perimetre EBICS strict et doivent etre implementes comme protocoles Gateway
  autonomes, tout en restant des pre-requis imperatifs du chantier metier
  cible.
- 2026-04-01: le chantier `P0` RTN reel est maintenant entame cote code.
  `pkg/protocols/modules/ebics/rtn_service.go` ajoute un service de fond
  branche dans `gatewayd`, qui charge les providers RTN activees, consomme
  leurs evenements, persiste l'idempotence et programme un vrai `Transfer`
  Gateway immediat pour l'auto-pull.
  Ce `Transfer` est relie a une `EbicsOperation` `AUTO_TRIGGERED` pre-creee,
  ensuite reutilisee par le runtime client EBICS existant.
  La premiere vague de tests est posee dans
  `pkg/protocols/modules/ebics/rtn_service_test.go`.
  2026-04-01: le scenario RTN complet jusqu'au payload final est maintenant
  ferme par `pkg/protocols/modules/ebics/rtn_controller_integration_test.go`.
  Le vrai chemin de production
  `RTN -> Transfer planifie -> controller -> ClientPipeline -> client EBICS HTTP
  -> serveur EBICS HTTP -> payload final`
  passe en vert.
  Les defauts runtime corriges sur ce maillon final sont:
  abandon du `TransactionID` synthetique sur `BTD`,
  persistance du vrai `TransactionID` de download,
  relecture correcte de `ebicsOperationID` en `json.Number`,
  et restauration de `EndTransfer()` cote client EBICS pour conserver le lien
  vers l'historique archive.
  2026-04-01: `P0C` est maintenant ferme.
  L'evenement RTN ne passe plus prematurement a `PROCESSED` lors de la simple
  programmation du `Transfer` auto-triggered. Il reste `PROCESSING` jusqu'a
  l'issue reelle du flux client EBICS, puis est synchronise en
  `PROCESSED`, `RETRYABLE` ou `FAILED` selon l'operation finale.
  Les informations operateur d'auto-pull (`operation`, `transfer`, `status`,
  `outcome`, `retry`) sont visibles en REST/CLI.
- 2026-04-01: le reste a faire post-`B5` n'est plus seulement formule en
  analyse libre.
  Il est maintenant decliné en lots cochables `P0`, `P2`, `P3`, `P4` et `P5` dans
  `suivi-backend-consolidation.md`, avec attendus, validations minimales et
  rappel explicite du hors-perimetre EBICS strict pour `AMQP 0.9.1` /
  `AMQP 1.0`.
  Le lot `P2D` ajoute en particulier l'historisation native des ordres EBICS
  non payload, pour ne pas limiter la tracabilite durable aux seuls
  transferts Gateway.
  2026-04-03: le lot `P2D` est ferme avec une persistance append-only
  `ebics_history_entries`, exposee en REST/CLI, pour les operations non
  payload terminales et les actions locales `initialisation / key lifecycle /
  rotation`.
  Un lot `P2E` est aussi ajoute pour remettre l'implementation EBICS en
  conformite avec la philosophie Gateway sur la selection de client:
  les chemins admin/RTN ne doivent plus reposer sur l'hypothese
  "un seul client EBICS actif", mais converger vers une reference canonique
  `ClientID` ou equivalente.
  Le lot `P2B` couvre aussi desormais le refresh planifie des vues
  contractuelles `HPD` / `HKD` / `HTD` / `HAA`, aujourd'hui disponible comme
  action client mais pas encore automatise nativement.
  Le chantier `P4` est dedie a la remise en ordre architecturale de
  l'implementation EBICS autour de `TransferInfo`: l'objectif est de separer
  proprement l'espace metier/exploitant expose en `#TI_*#` des correlations
  techniques EBICS, quitte a redevelopper une partie importante du runtime et
  de la persistance de correlation.
  Le chantier `P5` traite explicitement le cas ou Waarp Gateway joue le role
  de serveur bancaire EBICS et doit donc exposer, au-dela de `BTU/BTD`,
  des ordres contractuels/admin comme `HPD` / `HKD` / `HTD` / `HAA`,
  mais aussi couvrir les ordres d'initialisation, de gestion/rotation de
  cles, de reporting/signature, ainsi qu'un RTN sortant pour notifier aux
  partenaires qu'un ordre/document est disponible a la recuperation.
  L'ordre de traitement recommande est maintenant explicite:
  terminer d'abord le socle EBICS exploitable et multi-client (`P2E`, puis
  `P2A/B/D/C`), implementer ensuite `AMQP 0.9.1` et `AMQP 1.0` comme
  protocoles Gateway autonomes, ouvrir ensuite le passe-plat metier, puis
  seulement derouler le mode banque EBICS (`P5`) et enfin le workflow VEU
  (`P3`).
  2026-04-02: `P2E` est maintenant detaille en sous-lots operationnels dans
  `suivi-backend-consolidation.md`:
  inventaire des resolutions implicites, fixation du contrat `ClientID`,
  refactor des chemins admin, alignement RTN, puis lisibilite REST/CLI de
  l'etat activable et des ambiguities multi-client.
  L'inventaire `P2E.1` est deja tranche:
  le payload standard Gateway est bien aligne sur `Transfer.ClientID`,
  tandis que l'ecart multi-client reste concentre sur les chemins
  non payload et sur la resolution RTN.
  2026-04-02: `P2E.3` est maintenant ferme pour les chemins non payload.
  Les actions `contract refresh`, `reporting`, `signature`,
  `initialisation`, `HPB` et `key rotation` exigent desormais un `clientID`
  explicite en REST/CLI et n'utilisent plus de resolution singleton
  implicite du client EBICS. Le lot restant sur ce sujet est `P2E.4` pour RTN.
  2026-04-02: `P2E.4` est maintenant ferme.
  Les providers RTN administres portent desormais eux aussi un `clientID`
  explicite, et l'auto-pull ne depend plus d'un `clientName` optionnel ni
  d'une resolution implicite parmi les clients EBICS actifs. La selection
  multi-client est donc alignee sur `ClientID` pour tous les chemins EBICS
  hors `Transfer`.
  2026-04-02: `P2E.5` est maintenant ferme.
  Les providers RTN exposes en REST/CLI rendent maintenant visible le client
  selectionne (`clientID`, `clientName`) ainsi qu'un etat d'activation
  operateur avec raison bloquante si le perimetre n'est pas activable.
  2026-04-02: `P2A` est maintenant ferme.
  Une retention automatisee minimale est maintenant integree au runtime
  Gateway via un service de maintenance technique EBICS dedie, distinct de la
  purge manuelle de l'historique des transferts. La politique active est
  stockee en base dans `ebics_runtime_policies`, pas dans le fichier de
  configuration. Les `nonces` expires sont purges automatiquement, les
  transactions EBICS ne sont purgees que si elles sont terminales et
  anciennes, les evenements RTN ne sont purges que s'ils sont terminaux et
  anciens, et les transactions encore actives restent explicitement hors
  purge. La policy singleton `default` est maintenant administrable en
  REST/CLI, sans detour par un parametrage statique de l'instance.
  2026-04-03: `P2B` est maintenant ferme.
  Le refresh planifie des vues contractuelles client est desormais natif:
  une policy `EbicsContractRefreshPolicy` administree en base pilote
  periodiquement `HEV` / `HPD` / `HKD` / `HTD` / `HAA` pour un couple explicite
  `clientID` + `subscriberID`.
  Le runtime expose l'etat d'orchestration (`status`, `nextRunAt`,
  `lastAttemptAt`, `lastSuccessAt`, `lastError`) et une lecture operateur
  `activationStatus` / `activationReason`.
  L'administration correspondante est disponible en REST/CLI via
  `/ebics/contract-refresh-policies` et
  `ebics contract-refresh-policy add/list/get/update/delete`.
  Les migrations, tests et la passe `golangci-lint` du perimetre touche sont
  maintenant verts.
  2026-04-03: `P2C` est maintenant ferme.
  La preuve runtime ne repose plus seulement sur des tests directs du routeur
  payload ou sur le scenario RTN:
  un test passe maintenant par le vrai `controller` et le vrai
  `ClientPipeline` pour un `BTD` planifie hors RTN,
  et un autre passe par le vrai serveur HTTP EBICS avec un client
  `lib-ebics` reel.
  La repasse `go test` et `golangci-lint` sur
  `pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`
  est verte.
  2026-04-03: `P5A` est maintenant ferme comme cadrage fonctionnel/technique.
  Le document `gateway-role-banque-ebics.md` fixe le perimetre cible quand
  Waarp Gateway joue le role banque:
  `HPD/HKD/HTD/HAA` et `RTN` sortant minimal en priorite,
  puis initialisation / key management / rotations,
  puis reporting / signature.
  Il fixe aussi la doctrine de conception:
  reponses serveur alimentees par des projections internes explicites,
  aucun glissement metier cache dans les handlers EBICS,
  et aucune reutilisation de `TransferInfo` comme bus technique.
  2026-04-03: `P5B` est maintenant ferme.
  Le serveur Gateway EBICS expose maintenant `HPD`, `HKD`, `HTD`, `HAA`
  via les handlers `lib-ebics` natifs.
  Les reponses sont alimentees par une projection contractuelle serveur
  dediee, pas par un detour via le client.
  Un test HTTP reel couvre le telechargement de `HPD/HKD/HTD/HAA`
  depuis un client `lib-ebics` vers le serveur Gateway.
  2026-04-03: `P5C` est maintenant ferme.
  La projection contractuelle serveur est maintenant explicitement
  separee des snapshots client:
  `EbicsServerContractSet` / `EbicsServerContractItem` remplacent
  l'usage de `EbicsContractView` pour les ordres serveur.
  Le bornage fonctionnel est impose par le modele:
  `HPD/HAA` restent host-scoped,
  `HKD/HTD` restent subscriber-scoped.
  Une surface REST/CLI minimale permet d'inspecter ces projections
  via `server-contract-sets`, et la repasse
  `go test` / `golangci-lint` sur le perimetre touche est verte.
  2026-04-03: premiere tranche `P5D` engagee et validee.
  Les ordres serveur `INI/HIA/HPB/PUB/HSA/H3K/HCA/HCS/SPR`
  sont maintenant explicitement bornes par une policy serveur dediee,
  au lieu de reposer sur `AllowAllPolicy`.
  Le perimetre reel valide est:
  - `INI/HIA/HPB` nominaux sur HTTP/TLS avec client `lib-ebics` reel;
  - rejet d'un subscriber serveur desactive;
  - correction du XML `INI/HIA` emis cote client pour produire un
    `ds:X509Data` conforme;
  - correction des fixtures de cles banque serveur `HPB`,
    qui doivent etre stockees comme fragments XML et non comme PEM bruts.
  2026-04-03: seconde tranche `P5D` engagee sur le reporting serveur
  `HVD/HVU/HVZ/HVT/HAC`, avec projection dediee
  `EbicsServerReportingSet/Item`, integration HTTP/TLS reelle et lecture
  REST/CLI minimale.
  2026-04-03: `P5D` est maintenant ferme.
  Les ordres serveur `HVE/HVS` sont couverts sur le vrai chemin HTTP/TLS, le
  correctif `lib-ebics` necessaire a `HVT completeOrderData=true` est integre
  proprement cote dependance, et le bornage des rotations serveur reste
  explicitement limite aux workflows serveur deja exposes
  (`PUB/HSA/H3K/HCA/HCS/SPR`).
  2026-04-01: `P4A` est maintenant ferme.
  La cartographie exhaustive montre deux problemes distincts:
  les cles EBICS structurelles (`ebicsOperationID`, `ebicsRTNEventID`,
  `ebicsTransactionID`, identite protocolaire, profil, endpoint, service)
  sont utilisees comme support runtime dans `TransferInfo`,
  et le chemin RTN y deverse en plus le `PayloadMap` brut des evenements.
  La suite `P4B` devra donc traiter a la fois le relogement des correlations
  critiques et la suppression du pass-through RTN vers `TransferInfo`.
  2026-04-01: `P4B` est maintenant ferme.
  Le modele cible retenu interdit toute dependance runtime critique a
  `TransferInfo`.
  Les correlations passent par `EbicsOperation`, `EbicsTransaction` et
  `EbicsRTNEvent`; les informations de resolution techniques vont dans
  `ebics_operations.metadata`; le message RTN complet reste dans
  `ebics_rtn_events.payload`; `TransferInfo` ne conserve, au mieux, qu'une
  whitelist future de metadonnees explicitement assumees comme visibles pour
  l'exploitant.
  2026-04-01: premiere tranche technique engagee et validee.
  Le client payload recharge maintenant son contexte critique depuis
  `EbicsOperation`; le chemin RTN ne clone plus le `PayloadMap` brut dans
  `TransferInfo`; un helper partage de contexte EBICS alimente un bloc REST
  `ebicsContext` sur les transferts; et les taches disposent de variables
  dediees `#EBICS_*#`.
  Aucun fallback de compatibilite interne EBICS n'est retenu a ce stade:
  l'implementation converge directement vers le modele cible propre.
  Point de cadrage ajoute: cette remise en ordre vise les cles techniques
  EBICS / RTN. Les metadonnees standard du moteur Gateway comme `__followID__`
  restent hors de ce perimetre.
  2026-04-01: `P4C` est maintenant ferme.
  Les lectures techniques EBICS residuelles ont ete retirees du runtime
  payload; les chemins non payload `admin/reporting/key rotation/init` ont ete
  verifies et n'utilisent pas `TransferInfo` comme bus technique.
  2026-04-01: `P4D` est maintenant ferme.
  Les surfaces operateur, historique et CLI sont alignees sur le canal dedie
  `ebicsContext` / `#EBICS_*#`, y compris apres archivage via
  `metadata.archivedTransferID`.
  `TransferInfo` ne porte plus de correlation technique EBICS; seules restent
  les metadonnees natives du moteur Gateway comme `__followID__`.
  2026-04-01: le point linter est maintenant qualifie.
  L'echec ne venait pas du code EBICS ni de `golangci-lint`, mais du shell
  sandboxe sans acces a `C:\\Users\\driss\\.config\\git\\ignore`.
  Hors sandbox, sous `pwsh 7.6.0` et le compte utilisateur reel, la passe
  linter fonctionne et redevient une gate qualite exploitable.
  2026-04-01: `P4E` est maintenant ferme.
  La passe de non-regression consolidee et la repasse linter hors sandbox sur
  le perimetre EBICS/REST/CLI/model/gatewayd sont vertes. Le chantier `P4`
  est maintenant considere comme ferme.
