# Design cible - `EbicsOperation`

## 1. Objet

Ce document fixe le design cible de l'objet `EbicsOperation`, introduit pour
porter les ordres et traitements EBICS non payload sans les forcer dans
`Transfer`.

L'objectif est double:

- proteger le modele EBICS contre une projection artificielle dans `Rule` et
  `Transfer`;
- rester compatible avec le fonctionnement nominal de Gateway.

## 2. Positionnement

`EbicsOperation` est un objet d'exploitation technique.

Il ne remplace pas:

- `Transfer` pour les flux fichier;
- `Rule` pour les politiques techniques de routage;
- les workflows metier portes par l'application cliente.

Il porte:

- les ordres administratifs;
- les ordres de cycle de vie protocolaire;
- les ordres de consultation non projetes en transfert;
- les evenements RTN et actions techniques associees;
- les etats techniques de signature au sens EBICS.

## 3. Perimetre cible

Doivent relever en priorite de `EbicsOperation`:

- `HEV`
- `INI`, `HIA`, `H3K`
- `HPB`
- `HPD`, `HKD`, `HTD`, `HAA`
- `PUB`, `HCA`, `HCS`, `HSA`, `SPR`
- `HAC`
- `HVD`, `HVE`, `HVT`, `HVU`, `HVZ`, `HVS` tant qu'ils ne manipulent pas un
  payload a remettre dans le pipeline fichier
- ingestion RTN, polling et auto-pull en tant qu'actions techniques

Doivent rester portes par `Transfer`:

- `FUL`
- `FDL`
- `BTU`
- `BTD`
- plus generalement tout ordre materialise par un fichier a recevoir, deposer
  ou remettre a l'application metier

## 4. Modele logique

## 4.1 Table `ebics_operations`

Champs minimaux recommandes:

- `id`
- `owner`
- `local_agent_id`
- `client_id`
- `remote_agent_id`
- `local_account_id`
- `remote_account_id`
- `host_id`
- `partner_id`
- `user_id`
- `operation_type`
- `order_type`
- `direction`
- `transport_mode`
- `transaction_id`
- `request_id`
- `correlation_id`
- `ebics_version`
- `status`
- `severity`
- `technical_return_code`
- `technical_return_message`
- `business_return_code`
- `business_return_message`
- `gateway_outcome`
- `retry_decision`
- `manual_action_required`
- `transfer_id`
- `contract_view_id`
- `rtn_event_id`
- `started_at`
- `finished_at`
- `created_at`
- `updated_at`
- `metadata`

## 4.2 Champs structurants

`operation_type`:

- `INITIALIZATION`
- `KEY_MANAGEMENT`
- `CONTRACT_REFRESH`
- `REPORTING`
- `SIGNATURE`
- `RTN`
- `ADMIN`

`direction`:

- `INBOUND`
- `OUTBOUND`
- `INTERNAL`

`transport_mode`:

- `SYNC`
- `ASYNC`
- `AUTO_TRIGGERED`

`severity`:

- `INFO`
- `WARNING`
- `ERROR`

`gateway_outcome`:

- `SUCCESS`
- `SUCCESS_WITH_WARNING`
- `PENDING_BANK`
- `EMPTY_SUCCESS`
- `TECHNICAL_RETRYABLE_FAILURE`
- `TECHNICAL_FATAL_FAILURE`
- `BUSINESS_REJECTED`
- `MANUAL_CONFIRMATION_REQUIRED`

## 5. Cycle de vie

Statuts recommandes:

- `PLANNED`
- `READY`
- `RUNNING`
- `WAITING_EXTERNAL_CONFIRMATION`
- `WAITING_BANK`
- `WAITING_PAYLOAD_TRANSFER`
- `COMPLETED`
- `COMPLETED_WITH_WARNINGS`
- `FAILED`
- `CANCELLED`

Principes:

- `WAITING_EXTERNAL_CONFIRMATION` couvre par exemple la rupture
  d'automatisme de l'initialisation;
- `WAITING_PAYLOAD_TRANSFER` couvre les cas ou une operation administrative
  declenche ensuite un flux fichier associe;
- `COMPLETED` ne signifie pas necessairement "decision metier prise", mais
  "phase protocolaire terminee".
- les return codes `technical` et `business` restent visibles, meme si un
  statut derive Gateway est calcule.

## 6. Relation avec `Transfer`

La relation entre `EbicsOperation` et `Transfer` doit etre optionnelle.

Regles:

- une `EbicsOperation` peut exister seule;
- un `Transfer` EBICS devrait pouvoir referencer une `EbicsOperation` source;
- plusieurs `EbicsOperation` peuvent, selon les cas, pointer vers le meme
  `Transfer` ou vers aucun.

Recommendation:

- garder `Transfer` inchange a court terme;
- utiliser en priorite `ebics_operations.transfer_id` comme lien technique
  primaire quand un `Transfer` est associe;
- n'utiliser `TransferInfo` que comme metadonnee locale optionnelle de
  correlation ou de confort d'exploitation;
- introduire une table de liaison si les cas reels deviennent plus riches.

Important:

- `TransferInfo` est un mecanisme souple de metadonnees de transfert dans
  Gateway;
- il peut etre expose via REST et parfois transporte par certains protocoles;
- mais il ne constitue pas un contrat d'interoperabilite fiable avec un systeme
  tiers non Waarp.

Conclusion:

- `TransferInfo` ne doit pas etre la cle de voute du lien `EbicsOperation` <->
  `Transfer`;
- au mieux, il peut dupliquer localement une correlation deja portee par la
  persistance dediee.

`EbicsOperation` doit aussi devenir le pivot de correlation entre:

- identifiants Gateway (`transfer_id`, eventuel `flow_id`);
- identifiants EBICS (`transaction_id`, `request_id`);
- identifiants d'integration (`correlation_id`, `event_id`).

## 7. Relation avec `Rule`

`Rule` n'est jamais obligatoire pour creer une `EbicsOperation`.

`Rule` n'intervient que si:

- l'operation produit un flux fichier;
- ou si une politique technique Gateway doit etre appliquee au fichier derive.

Autrement dit:

- `EbicsOperation` est l'objet protocolaire;
- `Rule` reste un objet de politique technique fichier.

## 8. Surfaces REST cibles

Ressources minimales recommandees:

- `GET /ebics/operations`
- `GET /ebics/operations/{id}`
- `POST /ebics/operations/{id}/retry`
- `POST /ebics/operations/{id}/cancel`
- `POST /ebics/operations/{id}/confirm`

Filtres minimaux:

- `status`
- `operationType`
- `orderType`
- `partnerId`
- `userId`
- `from`
- `to`
- `transferId`

## 9. Surfaces CLI cibles

Commandes minimales recommandees:

- `waarp-gateway ebics operation list`
- `waarp-gateway ebics operation get <id>`
- `waarp-gateway ebics operation retry <id>`
- `waarp-gateway ebics operation cancel <id>`
- `waarp-gateway ebics operation confirm <id>`

## 10. Visibilite UI

L'UI d'exploitation devrait separer:

- une vue `Operations EBICS`
- une vue `Transfers`

et proposer une correlation visible entre les deux quand elle existe.

L'erreur a eviter est une UI qui afficherait un ordre `HPD` ou `INI` comme un
faux transfert de fichier.

## 11. Cas de projection types

### 11.1 `HPD`

- creation d'une `EbicsOperation`
- pas de `Transfer`
- mise a jour de `ebics_contract_views`

### 11.2 `INI`

- creation d'une `EbicsOperation`
- pas de `Transfer`
- statut possible `WAITING_EXTERNAL_CONFIRMATION`

### 11.3 `HAC` suivi d'un telechargement

- creation d'une `EbicsOperation` pour `HAC`
- creation eventuelle d'une seconde `EbicsOperation` pour l'action technique de
  collecte
- creation d'un `Transfer` uniquement si un fichier est reellement recupere

### 11.4 `FDL`

- creation d'une `EbicsOperation`
- creation d'un `Transfer` associe
- application d'une `Rule` technique si necessaire

## 12. Decision recommandee

Je recommande de conserver le nom `EbicsOperation` a ce stade.

Motifs:

- le besoin est immediatement EBICS;
- le nom est plus clair pour les exploitants et pour la documentation;
- il sera toujours possible de generaliser plus tard vers `ProtocolOperation`
  si un second protocole en a vraiment besoin.

## 13. Impact backlog

Ce design doit etre traite comme une sortie du lot 1 et une entree du lot 2.

Concretement:

- lot 1: valider le concept, les statuts et les surfaces d'administration;
- lot 2: persister `ebics_operations` et leurs correlations.
