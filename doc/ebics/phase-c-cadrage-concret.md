# Cadrage concret Phase C EBICS

## 1. Objet

Ce document detaille la `Phase C` du squelette technique EBICS.

La `Phase C` introduit les premiers objets d'execution reels:

- `EbicsOperation`;
- `EbicsTransaction`;
- `EbicsTransactionSegment`;
- la soumission des ordres payload;
- la projection optionnelle vers `Transfer`.

## 2. Position de travail

La `Phase C` est la premiere phase ou Gateway commence a porter une execution
EBICS materielle.

Elle doit donc etre plus stricte que les phases precedentes sur:

- la correlation d'identifiants;
- la lecture des return codes;
- la politique de retry/replay/recovery;
- la separation `operation` / `transaction` / `segment` / `transfer`.

## 3. Perimetre de la Phase C

### 3.1 Inclus

- model `EbicsOperation`;
- model `EbicsTransaction`;
- model `EbicsTransactionSegment`;
- DTO et contrats de soumission payload `BTU/BTD/FUL/FDL`;
- creation d'une `EbicsOperation` pour une demande payload;
- creation d'une `EbicsTransaction` pour une transaction EBICS reelle;
- projection vers `Transfer` quand un flux fichier existe reellement;
- runtime de mapping:
  - `operation_mapper`
  - `retry_policy`
  - debut de `transaction` / `segment tracking`.

### 3.2 Exclus

- `RTN`;
- `EbicsKeyLifecycle`;
- `EbicsInitializationWorkflow`;
- orchestration detaillee des signatures;
- administration complete de tous les workflows sensibles;
- toutes les politiques avancees de supervision.

## 4. Exigences `production grade`

### 4.1 Correlation

Chaque execution doit pouvoir etre reliee clairement a:

- ses identifiants EBICS:
  - `transaction_id`
  - `request_id`;
- ses identifiants Gateway:
  - `operation_id`
  - `transfer_id` si present;
- ses identifiants d'integration:
  - `correlation_id`.

### 4.2 Return codes

La `Phase C` ne doit jamais stocker ou exposer un return code EBICS unique.

Il faut porter explicitement:

- `technical_return_code`
- `technical_return_message`
- `business_return_code`
- `business_return_message`
- `gateway_outcome`
- `retry_decision`
- `manual_action_required`

### 4.3 Retry / replay / recovery

La `Phase C` doit garder la distinction stricte:

- `retry` de transport ou tentative technique;
- `replay` volontaire d'un ordre;
- `recovery` protocolaire via transaction/segmentation.

Le moteur natif de retry `Transfer` ne doit jamais devenir la politique globale
des executions EBICS.

### 4.4 Multi-SGBD

Comme pour les phases precedentes:

- models simples et portables via XORM;
- pas de type specifique a un moteur SQL;
- pas de logique critique basee sur des contraintes non portables seulement
  presentes dans le DDL de reference.

## 5. Fichiers a cadrer concretement

## 5.1 `pkg/model/table_names.go`

Ajouts cibles:

- `TableEbicsOperations = "ebics_operations"`
- `TableEbicsTransactions = "ebics_transactions"`
- `TableEbicsTransactionSegments = "ebics_transaction_segments"`

## 5.2 `pkg/model/display_names.go`

Ajouts cibles:

- `NameEbicsOperation = "ebics operation"`
- `NameEbicsTransaction = "ebics transaction"`
- `NameEbicsTransactionSegment = "ebics transaction segment"`

## 5.3 `pkg/model/ebics_operation.go`

Responsabilite:

- porter l'objet operationnel principal EBICS;
- servir de pivot de correlation et d'exploitation;
- conserver les resultats EBICS et le statut derive Gateway.

Invariants:

- `OperationType` obligatoire;
- `OrderType` obligatoire;
- `Direction` obligatoire;
- `Status` obligatoire;
- `Severity` obligatoire;
- `EbicsHostID` obligatoire;
- `EbicsSubscriberID` obligatoire;
- pas de `TransferID` si l'ordre ne projette jamais de transfert.

## 5.4 `pkg/model/ebics_transaction.go`

Responsabilite:

- porter l'etat transactionnel protocolaire EBICS;
- devenir le pivot de recovery;
- suivre la segmentation et la progression transactionnelle.

Invariants:

- `TransactionID` obligatoire;
- `OrderType` obligatoire;
- `Status` obligatoire;
- `Direction` obligatoire;
- reference a `EbicsHostID` et `EbicsSubscriberID` obligatoire;
- `TransferID` optionnel;
- `EbicsOperationID` optionnel mais fortement recommande.

## 5.5 `pkg/model/ebics_transaction_segment.go`

Responsabilite:

- tracer les segments d'une transaction EBICS;
- conserver leur etat et les informations minimales de reprise;
- servir au `TxStore` futur.

Invariants:

- `EbicsTransactionID` obligatoire;
- `SegmentNumber` obligatoire et strictement positif;
- `SegmentStatus` obligatoire;
- unicite `(ebics_transaction_id, segment_number)`.

## 5.6 `pkg/protocols/modules/ebics/runtime/operation_mapper.go`

Responsabilite:

- convertir une demande payload resolue en `EbicsOperation`;
- appliquer les conventions de correlation et d'identifiants;
- centraliser la politique de creation initiale des objets.

## 5.7 `pkg/protocols/modules/ebics/runtime/retry_policy.go`

Responsabilite:

- deriver `gatewayOutcome` et `retryDecision`;
- centraliser la lecture des return codes EBICS;
- interdire les retries dangereux ou ambigus.

## 5.8 `pkg/protocols/modules/ebics/stores/operation_store.go`

Responsabilite:

- fournir les acces de persistance necessaires a `EbicsOperation`;
- isoler les acces de lecture/ecriture utiles a l'execution.

## 5.9 `pkg/protocols/modules/ebics/stores/tx_store.go`

Responsabilite:

- servir de facade de persistance pour les transactions et segments;
- preparer l'implementation du `TxStore` attendu par la lib EBICS.

## 5.10 `pkg/admin/rest/api/ebics_payload_requests.go`

Responsabilite:

- porter les DTO de soumission `BTU/BTD/FUL/FDL`;
- exposer le resultat de resolution et les erreurs de policy/contrat;
- ne pas y cacher la logique protocolaire.

## 5.11 `pkg/admin/rest/api/ebics_operations.go`

Responsabilite:

- porter les DTO `In/Out` d'exploitation de `EbicsOperation`;
- exposer les deux scopes de return codes et les decisions derivees.

## 5.12 `pkg/admin/rest/api/ebics_transactions.go`

Responsabilite:

- porter les DTO de lecture transaction/segment;
- rester d'abord orientee diagnostic et support.

## 6. Projection vers `Transfer`

La `Phase C` doit formaliser la projection vers `Transfer` sans la banaliser.

Regles:

- `FUL`, `FDL`, `BTU`, `BTD` creent une `EbicsOperation`;
- un `Transfer` n'est cree que si un fichier reel existe a pousser/recevoir;
- la liaison principale est `ebics_operations.transfer_id`;
- `TransferInfo` peut dupliquer une correlation locale, mais jamais la porter
  a lui seul;
- `Rule` n'intervient que pour la politique technique fichier.

## 7. Migrations et ordre de pose

Ordre recommande:

1. `table_names.go` et `display_names.go`
2. `ebics_operation.go`
3. `ebics_transaction.go`
4. `ebics_transaction_segment.go`
5. `stores/operation_store.go`
6. `stores/tx_store.go`
7. `runtime/operation_mapper.go`
8. `runtime/retry_policy.go`
9. DTO `api` payload/operation/transaction

## 8. Definition de done de la Phase C

La `Phase C` est terminee si:

- un payload resolu peut produire une `EbicsOperation` coherente;
- les identifiants `transaction_id/request_id/correlation_id` ont une place
  stable;
- les return codes EBICS sont portes sur deux scopes distincts;
- une `EbicsTransaction` et ses segments peuvent etre persistants;
- les ordres fichier projettent proprement un `Transfer` sans denaturer le
  modele Gateway;
- les politiques de retry/replay/recovery sont centralisees et explicites.

## 9. Point de vigilance principal

Le risque central de la `Phase C` est de vouloir faire rentrer EBICS dans le
modele `Transfer` plus que raisonnable.

La bonne ligne est:

- `EbicsOperation` pour l'exploitation protocolaire;
- `EbicsTransaction` pour la reprise;
- `EbicsTransactionSegment` pour la segmentation;
- `Transfer` pour le flux fichier seulement.
