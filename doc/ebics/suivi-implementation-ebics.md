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

- [ ] Ajouter les constantes operations/transactions/segments dans `table_names.go`
- [ ] Ajouter les appellations correspondantes dans `display_names.go`
- [ ] Creer `pkg/model/ebics_operation.go`
- [ ] Creer `pkg/model/ebics_transaction.go`
- [ ] Creer `pkg/model/ebics_transaction_segment.go`
- [ ] Creer `pkg/protocols/modules/ebics/stores/operation_store.go`
- [ ] Creer `pkg/protocols/modules/ebics/stores/tx_store.go`
- [ ] Creer `pkg/protocols/modules/ebics/runtime/operation_mapper.go`
- [ ] Creer `pkg/protocols/modules/ebics/runtime/retry_policy.go`
- [ ] Creer `pkg/admin/rest/api/ebics_payload_requests.go`
- [ ] Creer `pkg/admin/rest/api/ebics_operations.go`
- [ ] Creer `pkg/admin/rest/api/ebics_transactions.go`

## 5. Workflows sensibles Phase D

- [ ] Ajouter les constantes lifecycle/init dans `table_names.go`
- [ ] Ajouter les appellations correspondantes dans `display_names.go`
- [ ] Creer `pkg/model/ebics_key_lifecycle.go`
- [ ] Creer `pkg/model/ebics_initialization_workflow.go`
- [ ] Creer `pkg/protocols/modules/ebics/runtime/key_lifecycle_runner.go`
- [ ] Creer `pkg/protocols/modules/ebics/runtime/initialization_runner.go`
- [ ] Creer `pkg/protocols/modules/ebics/runtime/signature_state.go`
- [ ] Creer `pkg/admin/rest/api/ebics_key_lifecycles.go`
- [ ] Creer `pkg/admin/rest/api/ebics_initializations.go`

## 6. RTN Phase E

- [ ] Ajouter les constantes RTN dans `table_names.go`
- [ ] Ajouter les appellations RTN dans `display_names.go`
- [ ] Creer `pkg/model/ebics_rtn_event.go`
- [ ] Creer `pkg/model/ebics_rtn_provider.go`
- [ ] Creer `pkg/protocols/modules/ebics/rtn/provider.go`
- [ ] Creer `pkg/protocols/modules/ebics/rtn/wss_provider.go`
- [ ] Creer `pkg/protocols/modules/ebics/runtime/rtn_ingestion.go`
- [ ] Creer `pkg/protocols/modules/ebics/runtime/rtn_autopull.go`
- [ ] Creer `pkg/admin/rest/api/ebics_rtn.go`
- [ ] Creer `pkg/cmd/client/ebics_rtn.go`

## 7. REST handlers

- [ ] Creer `pkg/admin/rest/ebics_operations.go`
- [ ] Creer `pkg/admin/rest/ebics_payload_profiles.go`
- [ ] Creer `pkg/admin/rest/ebics_contract_views.go`
- [ ] Creer `pkg/admin/rest/ebics_payloads.go`
- [ ] Creer `pkg/admin/rest/ebics_key_lifecycles.go`
- [ ] Creer `pkg/admin/rest/ebics_initializations.go`
- [ ] Creer `pkg/admin/rest/ebics_rtn.go`
- [ ] Etendre `pkg/admin/rest/router.go`

## 8. CLI

- [ ] Creer `pkg/cmd/client/ebics_operations.go`
- [ ] Creer `pkg/cmd/client/ebics_payload.go`
- [ ] Creer `pkg/cmd/client/ebics_payload_profiles.go`
- [ ] Creer `pkg/cmd/client/ebics_contract_views.go`
- [ ] Creer `pkg/cmd/client/ebics_key_lifecycles.go`
- [ ] Creer `pkg/cmd/client/ebics_initializations.go`
- [ ] Creer `pkg/cmd/client/ebics_rtn.go`

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
