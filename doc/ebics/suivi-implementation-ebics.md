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

- [ ] Etendre `pkg/backup/export.go`
- [ ] Etendre `pkg/backup/import.go`
- [ ] Ajouter les helpers `*_export.go` / `*_import.go` si necessaire
- [ ] Verifier le round-trip JSON/YAML de tous les objets administres

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
