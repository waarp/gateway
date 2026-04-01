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
  Il est maintenant decliné en lots cochables `P0`, `P2`, `P3` et `P4` dans
  `suivi-backend-consolidation.md`, avec attendus, validations minimales et
  rappel explicite du hors-perimetre EBICS strict pour `AMQP 0.9.1` /
  `AMQP 1.0`.
  Le lot `P2D` ajoute en particulier l'historisation native des ordres EBICS
  non payload, pour ne pas limiter la tracabilite durable aux seuls
  transferts Gateway.
  Le chantier `P4` est dedie a la remise en ordre architecturale de
  l'implementation EBICS autour de `TransferInfo`: l'objectif est de separer
  proprement l'espace metier/exploitant expose en `#TI_*#` des correlations
  techniques EBICS, quitte a redevelopper une partie importante du runtime et
  de la persistance de correlation.
