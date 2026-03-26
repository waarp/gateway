# Premier cadrage de squelette technique EBICS

## 1. Objet

Ce document propose un premier cadrage de squelette technique pour l'integration
EBICS dans Gateway.

Il est strictement derive des objets deja figes dans les specifications.

Il ne cherche pas a redecider l'architecture, mais a transformer les decisions
existantes en structure de packages et de fichiers plausible.

## 2. Principes

- ne pas coder trop tot de logique metier implicite;
- partir des objets et contrats deja valides;
- reutiliser les patterns existants de Gateway;
- introduire les briques EBICS de maniere incrementale.

## 3. Packages cibles

### 3.1 Modele

Nouveaux fichiers cibles sous `pkg/model`:

- `ebics_host.go`
- `ebics_subscriber.go`
- `ebics_bank_key.go`
- `ebics_operation.go`
- `ebics_transaction.go`
- `ebics_transaction_segment.go`
- `ebics_nonce_entry.go`
- `ebics_contract_view.go`
- `ebics_contract_view_item.go`
- `ebics_key_lifecycle.go`
- `ebics_initialization_workflow.go`
- `ebics_rtn_event.go`
- `ebics_payload_profile.go`

Responsabilite:

- structs XORM;
- validations minimales;
- hooks `BeforeWrite/AfterRead` si necessaire;
- constantes d'etats et de types.

### 3.2 Module protocolaire

Nouveau dossier cible:

- `pkg/protocols/modules/ebics`

Fichiers minimaux recommandes:

- `module.go`
- `config.go`
- `constants.go`
- `server.go`
- `client.go`
- `builders.go`
- `errors.go`

Responsabilite:

- implementer l'interface `protocols.Module`;
- faire le mapping `ProtoConfig <-> lib ebics`;
- instancier serveur et client EBICS.

### 3.3 Stores et persistance dediee

Sous `pkg/protocols/modules/ebics/stores`:

- `tx_store.go`
- `nonce_store.go`
- `key_store.go`
- `subscriber_store.go`
- `operation_store.go`

Responsabilite:

- adapter la persistance Gateway aux interfaces attendues par la lib EBICS;
- centraliser les acces aux objets EBICS durables.

### 3.4 Mapping et orchestration

Sous `pkg/protocols/modules/ebics/runtime`:

- `operation_mapper.go`
- `payload_resolution.go`
- `contract_validation.go`
- `retry_policy.go`
- `signature_state.go`
- `key_lifecycle_runner.go`
- `initialization_runner.go`

Responsabilite:

- resolution `explicite > profile > defaults`;
- validation contre `contract_view_items`;
- mapping vers `EbicsOperation` et `Transfer`;
- decisions `retry/recovery`;
- orchestration technique des workflows.

## 4. Administration REST

### 4.1 DTO API

Nouveaux fichiers cibles sous `pkg/admin/rest/api`:

- `ebics_operations.go`
- `ebics_payload_profiles.go`
- `ebics_contract_views.go`
- `ebics_key_lifecycles.go`
- `ebics_initializations.go`
- `ebics_rtn.go`
- `ebics_payload_requests.go`

Responsabilite:

- DTO `In/Out`;
- formes JSON/YAML officielles.

### 4.2 Handlers REST

Nouveaux fichiers cibles sous `pkg/admin/rest`:

- `ebics_operations.go`
- `ebics_payload_profiles.go`
- `ebics_contract_views.go`
- `ebics_key_lifecycles.go`
- `ebics_initializations.go`
- `ebics_rtn.go`
- `ebics_payloads.go`

Ajout dans:

- `pkg/admin/rest/router.go`

Responsabilite:

- routes `/api/ebics/...`;
- lecture/ecriture admin;
- validation d'entree;
- actions `retry/cancel/confirm/recover`.

## 5. CLI

Nouveaux fichiers cibles sous `pkg/cmd/client`:

- `ebics_payload.go`
- `ebics_operations.go`
- `ebics_payload_profiles.go`
- `ebics_contract_views.go`
- `ebics_key_lifecycles.go`
- `ebics_initializations.go`
- `ebics_rtn.go`

Responsabilite:

- commandes `waarp-gateway ebics ...`;
- affichage texte/json/yaml;
- garde-fous d'exploitation.

## 6. Import / export / updateconf

Points a etendre:

- `pkg/backup/export.go`
- `pkg/backup/import.go`
- fichiers `*_export.go` / `*_import.go` dedies EBICS si necessaire;
- `pkg/tasks/updateconf.go`

Objets a couvrir:

- hosts
- subscribers
- payload profiles
- contract views
- key lifecycles
- initialization workflows
- RTN providers/events selon la granularite retenue

## 7. Ordre d'implementation recommande

### 7.1 Phase A - socle modele + config

- structs `model`
- `ProtoConfig` EBICS
- enregistrement du module dans `pkg/protocols/modules.go`

### 7.2 Phase B - payload profiles + contract view

- `EbicsPayloadProfile`
- `EbicsContractView`
- `EbicsContractViewItem`
- resolution et validation payload

### 7.3 Phase C - operations + payload requests

- `EbicsOperation`
- soumission `BTU/BTD/FUL/FDL`
- projection vers `Transfer`

### 7.4 Phase D - workflows sensibles

- `EbicsKeyLifecycle`
- `EbicsInitializationWorkflow`
- `signatureState`

### 7.5 Phase E - RTN

- `EbicsRTNEvent`
- provider `WebSocket/WSS`
- auto-pull

## 8. Ce qu'il ne faut pas faire dans le premier squelette

- implementer tout EBICS d'un coup;
- reouvrir la frontiere protocole/metier;
- surcharger `Transfer` pour tous les ordres;
- enfouir les validations contractuelles dans `Rule`;
- faire reposer la rotation des cles sur un simple patch de `Credential`.

## 9. Definition de done du squelette

Le squelette est acceptable si:

- le module `ebics` est enregistrable;
- les `ProtoConfig` sont validables;
- les models existent avec leurs types de base;
- les routes REST et commandes CLI sont posees a vide ou en stub coherent;
- les objets sensibles sont representes explicitement:
  - `EbicsOperation`
  - `EbicsPayloadProfile`
  - `EbicsContractViewItem`
  - `EbicsKeyLifecycle`
  - `EbicsInitializationWorkflow`

## 10. Recommandation

Le premier squelette technique doit etre lance a partir de ce cadrage, lot par
lot, en commencant par le socle modele/config, puis par les payload profiles et
la validation contractuelle.

Le detail concret de la `Phase A` est complete dans:

- `phase-a-cadrage-concret.md`
- `phase-a-structs-et-signatures.md`

Le detail concret de la `Phase B` est complete dans:

- `phase-b-cadrage-concret.md`
- `phase-b-structs-et-signatures.md`

Le detail concret de la `Phase C` est complete dans:

- `phase-c-cadrage-concret.md`
- `phase-c-structs-et-signatures.md`

Le detail concret de la `Phase D` est complete dans:

- `phase-d-cadrage-concret.md`
- `phase-d-structs-et-signatures.md`

Le detail concret de la `Phase E` est complete dans:

- `phase-e-cadrage-concret.md`
- `phase-e-structs-et-signatures.md`
