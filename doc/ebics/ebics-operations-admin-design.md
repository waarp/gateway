# Design d'administration - `EbicsOperation`

## 1. Objet

Ce document definit les surfaces minimales d'administration de
`EbicsOperation` pour REST, CLI et UI.

L'objectif est de rester au plus proche des conventions existantes de Gateway,
sans creer une filiere d'administration parallele.

## 2. Principes

- reutiliser les patterns des routes `transfers`;
- distinguer clairement `Operations EBICS` et `Transfers`;
- ne proposer que des actions techniquement defensables;
- ne pas transformer l'admin Gateway en poste de decision metier;
- garantir une correlation visible entre `EbicsOperation` et `Transfer` quand
  elle existe.

## 3. Positionnement dans l'admin Gateway

`EbicsOperation` doit etre expose comme une ressource propre.

Recommandation:

- ne pas le cacher sous `/transfers`;
- ne pas le noyer sous `/partners` ou `/clients`;
- creer une famille de routes dediee `/ebics/operations`.

Cette ressource reste toutefois integree a la couche d'administration standard
de Gateway.

## 4. Permissions

Recommendation pragmatique de phase 1:

- rattacher `EbicsOperation` aux permissions existantes `transfers` pour
  lecture/ecriture.

Motifs:

- l'objet est d'exploitation;
- il est proche de la supervision et des reprises;
- cela evite d'ouvrir prematurement une nouvelle famille de permissions.

Reserve:

- si les operations EBICS deviennent suffisamment riches et sensibles, une
  permission dediee pourra etre introduite plus tard.

## 5. REST cible

## 5.1 Routes minimales

Routes recommandees:

- `GET /api/ebics/operations`
- `GET /api/ebics/operations/{operation}`
- `POST /api/ebics/operations/actions/reporting`
- `POST /api/ebics/operations/actions/signature`

## 5.2 Liste

`GET /api/ebics/operations`

Filtres minimaux:

- `status`
- `operationType`
- `orderType`
- `direction`
- `partnerId`
- `userId`
- `transferId`
- `start`
- `stop`
- `sort`
- `limit`
- `offset`

Tri minimal:

- `start+`
- `start-`
- `id+`
- `id-`
- `status+`
- `status-`
- `orderType+`
- `orderType-`

## 5.3 Consultation unitaire

`GET /api/ebics/operations/{operation}`

Le detail doit au minimum exposer:

- identite de l'operation;
- ordre EBICS;
- type d'operation;
- direction;
- statut;
- scopes `technical` et `business`;
- `gatewayOutcome` et `retryDecision`;
- identites EBICS;
- horodatages;
- lien vers `Transfer` si present;
- metadonnees techniques utiles;
- transaction associee si presente;
- segments associes si presents;
- references vers workflow d'initialisation, RTN ou contrat si present.

## 5.4 Actions

La famille `operations` n'est pas une facade generique de controle.

Regle de phase 1:

- `retry`, `cancel` et `confirm` restent sur les familles specialisees
  (`payload`, `initialization`, `key-rotation`, `rtn`);
- `operations` sert a observer et a lancer les ordres specialises
  `reporting` et `signature`;
- on evite ainsi une surface generique ambiguë, difficile a rendre
  techniquement sure.

## 5.5 Reponse REST de reference

Structure cible minimale:

```json
{
  "id": 42,
  "operationType": "CONTRACT_REFRESH",
  "orderType": "HPD",
  "direction": "OUTBOUND",
  "status": "COMPLETED",
  "severity": "INFO",
  "hostId": "BANKHOST01",
  "partnerId": "PARTNER01",
  "userId": "USER01",
  "transactionId": "A1B2C3",
  "requestId": "req-123",
  "technical": {
    "returnCode": "000000",
    "reportText": "[EBICS_OK] OK"
  },
  "business": {
    "returnCode": "000000",
    "reportText": "[EBICS_OK] OK"
  },
  "gatewayOutcome": "SUCCESS",
  "retryDecision": "NO_RETRY",
  "manualActionRequired": false,
  "transferId": null,
  "startedAt": "2026-03-25T10:15:00Z",
  "finishedAt": "2026-03-25T10:15:02Z",
  "metadata": {
    "contractViewId": 12
  }
}
```

## 6. CLI cible

## 6.1 Commandes minimales

Commandes recommandees:

- `waarp-gateway ebics operation list`
- `waarp-gateway ebics operation get <id>`
- `waarp-gateway ebics operation reporting ...`
- `waarp-gateway ebics operation signature ...`

## 6.2 Filtres CLI

Le `list` devrait proposer au minimum:

- `--status`
- `--operation-type`
- `--order-type`
- `--partner-id`
- `--user-id`
- `--transfer-id`
- `--date`
- `--sort`

## 6.3 Presentation CLI

La sortie humaine doit faire ressortir:

- l'ID;
- l'ordre EBICS;
- le type d'operation;
- la direction;
- le statut;
- les return codes `technical` et `business`;
- le `gatewayOutcome`;
- les identites EBICS;
- le `Transfer` associe si present.

Il faut eviter un affichage qui ressemble a celui d'un transfert de fichier.

## 7. UI cible

## 7.1 Vue liste

La vue `Operations EBICS` doit proposer:

- filtres par statut, ordre et type;
- tri;
- acces rapide a l'erreur, aux return codes `technical/business` et a la
  decision derivee;
- acces rapide au `Transfer` associe;
- badge visuel distinguant `payload` et `non payload`.

## 7.2 Vue detail

La vue detail doit proposer:

- resume technique;
- timeline de statuts;
- scopes `technical` et `business`;
- `gatewayOutcome` et `retryDecision`;
- correlation vers `Transfer`, `RTN`, `contract view`, `initialization workflow`
  si presents;
- liens vers les familles specialisees qui portent l'action effective.

## 8. Regles d'administration

Regles minimales:

- un ordre non payload n'apparait jamais comme faux `Transfer`;
- une action admin ne doit etre visible que si l'etat la permet;
- l'UI et la CLI doivent expliciter la nature technique de l'action;
- l'objet doit etre retrouvable sans connaitre les details internes EBICS.

## 9. Mapping des actions par type

### 9.1 `HPD`, `HKD`, `HTD`, `HAA`

Actions:

- `get`
- `list`
- relance via la famille specialisee porteuse de l'action

### 9.2 `INI`, `HIA`, `H3K`

Actions:

- `get`
- `list`
- action via la famille `initialization`

### 9.3 `RTN`

Actions:

- `get`
- `list`
- action via la famille `rtn`

## 10. Decision recommandee

Pour la phase 1:

- ressource REST dediee `/api/ebics/operations`;
- commandes CLI dediees `ebics operation`;
- vue UI dediee `Operations EBICS`;
- permissions initialement alignees sur `transfers`.

Cette approche est la plus lisible et la moins risquee.
