# Backlog d'implementation concret fichier par fichier

## 1. Objet

Ce document transforme les phases `A` a `E` en plan de developpement concret
fichier par fichier.

Il doit servir de reference pratique pendant l'implementation:

- quoi creer ou modifier;
- dans quel ordre;
- avec quelles dependances;
- avec quel critere de completion local.

## 2. Regles de lecture

- les documents `phase-a` a `phase-e` font foi pour les structs, signatures et
  invariants;
- ce backlog ne redecide pas l'architecture;
- un fichier est considere termine quand son contrat local est rempli, meme si
  les integrations amont/aval ne sont pas encore toutes actives;
- aucun lot ne doit etre traite comme un developpement minimaliste ou jetable;
- chaque objet, methode ou interface posee doit etre directement exploitable,
  documentee et coherente avec les invariants de production retenus;
- les items REST/CLI peuvent etre poses en stubs propres tant que le socle
  model/runtime n'est pas pret, mais un stub ne doit jamais masquer une
  decision d'architecture ou un comportement implicite dangereux;
- `golangci-lint` doit etre execute avant chaque compilation ou test Go cible;
- tout blocage de version entre le linter et la version Go du projet doit etre
  trace dans les documents de suivi.

## 3. Phase A - Socle modele + config

## 3.1 `pkg/model/table_names.go`

Travail:

- ajouter les constantes:
  - `TableEbicsHosts`
  - `TableEbicsSubscribers`
  - `TableEbicsBankKeys`

Dependances:

- aucune.

Definition locale de termine:

- les constantes existent;
- leur nom suit les conventions existantes `Table...`.

## 3.2 `pkg/model/display_names.go`

Travail:

- ajouter les constantes:
  - `NameEbicsHost`
  - `NameEbicsSubscriber`
  - `NameEbicsBankKey`

Dependances:

- aucune.

Definition locale de termine:

- les messages de validation peuvent utiliser des appellations stables.

## 3.3 `pkg/model/ebics_host.go`

Travail:

- creer la struct `EbicsHost`;
- implementer:
  - `TableName`
  - `Appellation`
  - `GetID`
  - `BeforeWrite`;
- ajouter les helpers de validation prives.

Dependances:

- `table_names.go`
- `display_names.go`

Definition locale de termine:

- la struct compile;
- les validations obligatoires sont posees;
- les unicites logiques sont verifiees via `db.Count`.

## 3.4 `pkg/model/ebics_subscriber.go`

Travail:

- creer la struct `EbicsSubscriber`;
- implementer:
  - `TableName`
  - `Appellation`
  - `GetID`
  - `BeforeWrite`.

Dependances:

- `ebics_host.go`

Definition locale de termine:

- verification d'existence du host parent;
- unicite `(host, partner, user)` posee.

## 3.5 `pkg/model/ebics_bank_key.go`

Travail:

- creer la struct `EbicsBankKey`;
- implementer:
  - `TableName`
  - `Appellation`
  - `GetID`
  - `BeforeWrite`.

Dependances:

- `ebics_host.go`

Definition locale de termine:

- types et etats de cle bornes;
- verification du host parent;
- unicite logique par host/type/version.

## 3.6 `pkg/protocols/modules/ebics/constants.go`

Travail:

- creer les constantes module:
  - `EBICS`
  - versions
  - transports
  - policies de profils.

Dependances:

- aucune.

Definition locale de termine:

- aucun litteral de protocole n'est requis ailleurs pour demarrer le module.

## 3.7 `pkg/protocols/modules/ebics/errors.go`

Travail:

- centraliser les erreurs nommees du module;
- ajouter le helper de wrapping config si retenu.

Dependances:

- `constants.go`

Definition locale de termine:

- les erreurs principales de bootstrap/config sont nommees.

## 3.8 `pkg/protocols/modules/ebics/config.go`

Travail:

- creer `serverConfig`, `clientConfig`, `partnerConfig`;
- implementer:
  - `ValidServer`
  - `ValidClient`
  - `ValidPartner`;
- ajouter les helpers de normalisation et validation.

Dependances:

- `constants.go`
- `errors.go`

Definition locale de termine:

- `CheckServerConfig`, `CheckClientConfig`, `CheckPartnerConfig` peuvent
  fonctionner sans panic;
- configuration vide valide ou normalisee proprement.

## 3.9 `pkg/protocols/modules/ebics/server.go`

Travail:

- creer la struct `Server`;
- implementer:
  - `NewServer`
  - `Start`
  - `Stop`
  - `State`.

Dependances:

- `config.go`
- `errors.go`

Definition locale de termine:

- le serveur peut etre instancie;
- `Start` et `Stop` sont propres, meme si le protocole n'est pas encore actif.

## 3.10 `pkg/protocols/modules/ebics/client.go`

Travail:

- creer la struct `Client`;
- implementer:
  - `NewClient`
  - `Start`
  - `Stop`
  - `State`
  - `InitTransfer`.

Dependances:

- `config.go`
- `errors.go`

Definition locale de termine:

- le client est instanciable;
- `InitTransfer` retourne une erreur/stub explicite, pas un comportement flou.

## 3.11 `pkg/protocols/modules/ebics/module.go`

Travail:

- implementer `protocols.Module`;
- brancher `NewServer`, `NewClient`, `Make*Config`.

Dependances:

- `server.go`
- `client.go`
- `config.go`

Definition locale de termine:

- le module est complet du point de vue de l'interface `protocols.Module`.

## 3.12 `pkg/protocols/modules.go`

Travail:

- enregistrer `ebics` dans la map globale.

Dependances:

- `pkg/protocols/modules/ebics/module.go`

Definition locale de termine:

- `protocols.IsValid("ebics") == true`.

## 3.13 `pkg/tasks/updateconf.go`

Travail:

- verifier la prise en charge des `ProtoConfig` EBICS;
- ajouter ce qui manque pour import/export/update si un ecart est constate.

Dependances:

- `config.go`

Definition locale de termine:

- `ProtoConfig` EBICS round-trippable via les chemins nominaux Gateway;
- si aucun ecart n'est detecte, solder l'item sans code supplementaire et tracer
  la justification dans les documents de suivi.

## 4. Phase B - Payload profiles + contract view

## 4.1 `pkg/model/table_names.go`

Travail:

- ajouter:
  - `TableEbicsContractViews`
  - `TableEbicsContractViewItems`
  - `TableEbicsPayloadProfiles`

## 4.2 `pkg/model/display_names.go`

Travail:

- ajouter:
  - `NameEbicsContractView`
  - `NameEbicsContractViewItem`
  - `NameEbicsPayloadProfile`

## 4.3 `pkg/model/ebics_contract_view.go`

Travail:

- creer la struct;
- poser les validations de status/source/scope.

Dependances:

- `ebics_host.go`
- `ebics_subscriber.go`

## 4.4 `pkg/model/ebics_contract_view_item.go`

Travail:

- creer la struct;
- poser les validations de coherence `item_type`.

Dependances:

- `ebics_contract_view.go`

## 4.5 `pkg/model/ebics_payload_profile.go`

Travail:

- creer la struct;
- poser:
  - validation `order_type`
  - validation `direction`
  - validation `DefaultRuleID`
  - serialisation/deserialisation des champs libres.

Dependances:

- `rule.go`

## 4.6 `pkg/protocols/modules/ebics/runtime/payload_resolution.go`

Travail:

- creer les structs input/output;
- implementer la resolution:
  - explicite
  - profil
  - defaults.

Dependances:

- `ebics_payload_profile.go`
- `config.go`

## 4.7 `pkg/protocols/modules/ebics/runtime/contract_validation.go`

Travail:

- creer les interfaces resolver;
- implementer la validation d'un payload resolu contre les items de contrat.

Dependances:

- `ebics_contract_view.go`
- `ebics_contract_view_item.go`
- `payload_resolution.go`

## 4.8 `pkg/admin/rest/api/ebics_payload_profiles.go`

Travail:

- creer DTO `In/Out`.

Dependances:

- `ebics_payload_profile.go`

## 4.9 `pkg/admin/rest/api/ebics_contract_views.go`

Travail:

- creer DTO `OutEbicsContractView`
- creer DTO `OutEbicsContractViewItem`.

Dependances:

- `ebics_contract_view.go`
- `ebics_contract_view_item.go`

## 4.10 `pkg/tasks/updateconf.go`

Travail:

- prendre en charge les payload profiles;
- definir la position des objets dans les imports/exports.

## 5. Phase C - Operations + payload requests

## 5.1 `pkg/model/table_names.go`

Travail:

- ajouter:
  - `TableEbicsOperations`
  - `TableEbicsTransactions`
  - `TableEbicsTransactionSegments`

## 5.2 `pkg/model/display_names.go`

Travail:

- ajouter:
  - `NameEbicsOperation`
  - `NameEbicsTransaction`
  - `NameEbicsTransactionSegment`

## 5.3 `pkg/model/ebics_operation.go`

Travail:

- creer la struct complete;
- poser:
  - validations de type/statut/direction;
  - validations de scopes return codes;
  - serialisation `Metadata`.

Dependances:

- `ebics_host.go`
- `ebics_subscriber.go`

## 5.4 `pkg/model/ebics_transaction.go`

Travail:

- creer la struct;
- poser validations transactionnelles et de counters.

Dependances:

- `ebics_operation.go`

## 5.5 `pkg/model/ebics_transaction_segment.go`

Travail:

- creer la struct;
- poser l'unicite logique par transaction/segment.

Dependances:

- `ebics_transaction.go`

## 5.6 `pkg/protocols/modules/ebics/stores/operation_store.go`

Travail:

- definir interface store;
- poser le contrat complet necessaire a la phase pour la persistance des
  operations, sans laisser de zone floue sur les responsabilites du store.

Dependances:

- `ebics_operation.go`

## 5.7 `pkg/protocols/modules/ebics/stores/tx_store.go`

Travail:

- definir interfaces transaction/segment;
- poser le contrat complet necessaire a la phase pour les transactions et la
  segmentation, sans laisser de comportement implicite.

Dependances:

- `ebics_transaction.go`
- `ebics_transaction_segment.go`

## 5.8 `pkg/protocols/modules/ebics/runtime/operation_mapper.go`

Travail:

- mapper demande payload resolue -> `EbicsOperation`;
- brancher correlation et `TransferID` optionnel.

Dependances:

- `payload_resolution.go`
- `ebics_operation.go`

## 5.9 `pkg/protocols/modules/ebics/runtime/retry_policy.go`

Travail:

- implementer `gatewayOutcome`;
- implementer `retryDecision`;
- centraliser les cas sensibles de `returncodes-ebics-gateway.md`.

Dependances:

- `returncodes-ebics-gateway.md`

## 5.10 `pkg/admin/rest/api/ebics_payload_requests.go`

Travail:

- creer DTO de soumission payload;
- creer DTO de reponse de soumission.

## 5.11 `pkg/admin/rest/api/ebics_operations.go`

Travail:

- creer DTO `OutEbicsOperation`;
- creer DTO d'action operateur.

## 5.12 `pkg/admin/rest/api/ebics_transactions.go`

Travail:

- creer DTO transaction/segment de lecture.

## 6. Phase D - Workflows sensibles

## 6.1 `pkg/model/table_names.go`

Travail:

- ajouter:
  - `TableEbicsKeyLifecycles`
  - `TableEbicsInitializationWorkflows`

## 6.2 `pkg/model/display_names.go`

Travail:

- ajouter:
  - `NameEbicsKeyLifecycle`
  - `NameEbicsInitializationWorkflow`

## 6.3 `pkg/model/ebics_key_lifecycle.go`

Travail:

- creer la struct;
- valider le mapping `Credential <-> lifecycle`;
- proteger les transitions illegales.

Dependances:

- `credentials.go`
- `ebics_subscriber.go`
- `ebics_operation.go`

## 6.4 `pkg/model/ebics_initialization_workflow.go`

Travail:

- creer la struct;
- poser les etats, steps, references d'operations;
- valider les ruptures d'automatisme.

Dependances:

- `ebics_subscriber.go`
- `ebics_operation.go`

## 6.5 `pkg/protocols/modules/ebics/runtime/key_lifecycle_runner.go`

Travail:

- implementer les actions;
- controler les transitions;
- tracer operator/reason/evidence.

## 6.6 `pkg/protocols/modules/ebics/runtime/initialization_runner.go`

Travail:

- implementer les actions;
- controler les transitions;
- bloquer les passages illicites vers `ACTIVATED`.

## 6.7 `pkg/protocols/modules/ebics/runtime/signature_state.go`

Travail:

- centraliser les constantes;
- deriver `signatureState`.

## 6.8 `pkg/admin/rest/api/ebics_key_lifecycles.go`

Travail:

- creer DTO de sortie lifecycle;
- creer DTO d'action lifecycle.

## 6.9 `pkg/admin/rest/api/ebics_initializations.go`

Travail:

- creer DTO de sortie workflow d'initialisation;
- creer DTO d'action.

## 7. Phase E - RTN

## 7.1 `pkg/model/table_names.go`

Travail:

- ajouter:
  - `TableEbicsRTNEvents`
  - `TableEbicsRTNProviders`

## 7.2 `pkg/model/display_names.go`

Travail:

- ajouter:
  - `NameEbicsRTNEvent`
  - `NameEbicsRTNProvider`

## 7.3 `pkg/model/ebics_rtn_event.go`

Travail:

- creer la struct;
- poser idempotence, statuts, retry, quarantine.

## 7.4 `pkg/model/ebics_rtn_provider.go`

Travail:

- creer la struct;
- poser transport/policy/configuration.

## 7.5 `pkg/protocols/modules/ebics/rtn/provider.go`

Travail:

- definir interface provider.

## 7.6 `pkg/protocols/modules/ebics/rtn/wss_provider.go`

Travail:

- implementer provider `WSS`;
- connexion/reconnexion/lecture/normalisation.

## 7.7 `pkg/protocols/modules/ebics/runtime/rtn_ingestion.go`

Travail:

- normaliser evenement;
- calculer la cle d'idempotence;
- inserer ou marquer duplicate.

## 7.8 `pkg/protocols/modules/ebics/runtime/rtn_autopull.go`

Travail:

- construire plan d'auto-pull;
- declencher les operations correspondantes.

## 7.9 `pkg/admin/rest/api/ebics_rtn.go`

Travail:

- creer DTO event/provider/action.

## 7.10 `pkg/cmd/client/ebics_rtn.go`

Travail:

- poser les commandes d'exploitation RTN.

## 8. REST et CLI transverses a poser apres socle

## 8.1 `pkg/admin/rest/ebics_operations.go`

Travail:

- lister/get/retry/cancel/confirm.

## 8.2 `pkg/admin/rest/ebics_payload_profiles.go`

Travail:

- list/add/get/update.

## 8.3 `pkg/admin/rest/ebics_contract_views.go`

Travail:

- get/list/refresh.

## 8.4 `pkg/admin/rest/ebics_payloads.go`

Travail:

- upload/download/get/retry/recover.

## 8.5 `pkg/admin/rest/ebics_key_lifecycles.go`

Travail:

- list/get/actions.

## 8.6 `pkg/admin/rest/ebics_initializations.go`

Travail:

- list/get/actions.

## 8.7 `pkg/admin/rest/ebics_rtn.go`

Travail:

- list/get/retry/quarantine/providers.

## 8.8 `pkg/admin/rest/router.go`

Travail:

- enregistrer les familles de routes EBICS dans un bloc coherent.

## 8.9 `pkg/cmd/client/ebics_operations.go`

Travail:

- `list/get/retry/cancel/confirm`.

## 8.10 `pkg/cmd/client/ebics_payload.go`

Travail:

- `upload/download/get/list/retry/recover`.

## 8.11 `pkg/cmd/client/ebics_payload_profiles.go`

Travail:

- `add/get/list/update`.

## 8.12 `pkg/cmd/client/ebics_contract_views.go`

Travail:

- `get/list/refresh/capabilities/permissions`.

## 8.13 `pkg/cmd/client/ebics_key_lifecycles.go`

Travail:

- actions lifecycle.

## 8.14 `pkg/cmd/client/ebics_initializations.go`

Travail:

- actions initialization.

## 9. Fichiers de suivi d'implementation recommandes

Pendant le developpement, utiliser:

- `suivi-implementation-ebics.md`
- `suivi-implementation-phases.md`

comme checklists vivantes a cocher.
