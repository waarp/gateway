# Routes REST detaillees pour EBICS

## 1. Objet

Ce document derive les `model` et DTO cibles vers des routes REST detaillees
pour l'administration EBICS dans Gateway.

Il vise une phase 1 exploitable, coherente avec les patterns REST existants de
Gateway.

## 2. Principes REST

- reutiliser le prefixe `/api`;
- suivre les patterns `list / get / create-action / update`;
- reserver les actions techniques a des sous-routes explicites;
- distinguer clairement:
  - `payloads`
  - `operations`
  - `contract-view`
  - `rtn`
  - `transactions`

## 3. Famille `payloads`

Cette famille couvre les ordres EBICS avec payload reel:

- `BTU`
- `BTD`
- `FUL` alias de compatibilite normalise vers `BTU`
- `FDL` alias de compatibilite normalise vers `BTD`

Principe:

- un appel de soumission cree d'abord une `EbicsOperation`;
- si un flux fichier est materiellement manipule, un `Transfer` y est
  associe;
- les reponses REST doivent permettre de suivre les deux objets.

### 3.1 Upload

Routes cibles:

- `POST /api/ebics/payloads/btu/upload`
- `POST /api/ebics/payloads/ful/upload`

Corps cible:

```json
{
  "profile": "sct-corp-credit-transfer",
  "rule": "ebics-send-default",
  "subscriber": {
    "hostId": "BANKHOST01",
    "partnerId": "PARTNER01",
    "userId": "USER01"
  },
  "file": {
    "path": "/payloads/pain001.xml",
    "outputName": "pain001.xml"
  },
  "service": {
    "scope": "GLB",
    "serviceName": "SCT",
    "serviceOption": "COR",
    "msgName": "pain.001"
  },
  "metadata": {
    "correlationId": "corr-123",
    "declaredAmount": "1520.45"
  }
}
```

Reponse cible:

```json
{
  "operationId": 84,
  "transferId": 9012,
  "status": "PLANNED"
}
```

Regles:

- `profile` reference un profil payload EBICS predefini;
- `rule` reference la politique technique Gateway de projection fichier;
- les champs explicites du corps surchargent les valeurs du profil.
- la resolution suit l'ordre `explicite > profile > defaults`;
- le resultat resolu doit etre valide contre `ebics_contract_view_items`;
- la politique produit peut imposer `profile-required`,
  `profile-preferred` ou `free-input-allowed`.

### 3.2 Download

Routes cibles:

- `POST /api/ebics/payloads/btd/download`
- `POST /api/ebics/payloads/fdl/download`

Corps cible:

```json
{
  "profile": "camt-bilateral-download",
  "rule": "ebics-receive-default",
  "subscriber": {
    "hostId": "BANKHOST01",
    "partnerId": "PARTNER01",
    "userId": "USER01"
  },
  "target": {
    "directory": "/collecte"
  },
  "service": {
    "scope": "BIL",
    "serviceName": "CAMT"
  },
  "metadata": {
    "correlationId": "corr-456"
  }
}
```

Regles:

- memes principes de resolution que pour l'upload;
- le profil reste le mode nominal;
- la collecte libre hors profil peut etre interdite selon la politique
  d'exploitation.

### 3.2 bis Profils payload

Routes cibles:

- `GET /api/ebics/payload-profiles`
- `POST /api/ebics/payload-profiles`
- `GET /api/ebics/payload-profiles/{profile}`
- `PATCH /api/ebics/payload-profiles/{profile}`

### 3.3 Consultation et reprise

Routes cibles:

- `GET /api/ebics/payloads`
- `GET /api/ebics/payloads/{operation}`
- `PUT /api/ebics/payloads/{operation}/retry`
- `PUT /api/ebics/payloads/{operation}/recover`

Position:

- cette famille est un point d'entree ergonomique;
- elle reste coherente avec `EbicsOperation` et `Transfer`;
- elle ne remplace pas les routes generiques d'operations.

## 4. Famille `EbicsOperation`

### 3.1 Liste

Route:

- `GET /api/ebics/operations`

Query params cibles:

- `limit`
- `offset`
- `sort`
- `status`
- `operationType`
- `orderType`
- `direction`
- `partnerId`
- `userId`
- `transactionId`
- `requestId`
- `correlationId`
- `transferId`
- `start`
- `stop`

Tri cible:

- `start+`
- `start-`
- `id+`
- `id-`
- `status+`
- `status-`
- `orderType+`
- `orderType-`

Reponse cible:

```json
{
  "operations": [
    {
      "id": 42,
      "operationType": "CONTRACT_REFRESH",
      "orderType": "HPD",
      "direction": "OUTBOUND",
      "transportMode": "SYNC",
      "status": "COMPLETED",
      "severity": "INFO",
      "hostId": "BANKHOST01",
      "partnerId": "PARTNER01",
      "userId": "USER01",
      "requestId": "req-123",
      "correlationId": "corr-123",
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
      "transferId": null
    }
  ]
}
```

### 3.2 Detail

Route:

- `GET /api/ebics/operations/{operation}`

Contenu minimal:

- detail `operation`;
- `transaction` associee si presente;
- `segments` si presents;
- `links` vers `transfer`, `contractView`, `rtnEvent`.
- scopes `technical` et `business` distincts quand presents.

### 3.3 Retry

Route:

- `PUT /api/ebics/operations/{operation}/retry`

Corps cible:

```json
{
  "reason": "operator retry after temporary bank outage",
  "metadata": {
    "requestedBy": "ops-user"
  }
}
```

Codes:

- `202 Accepted`
- `400 Bad Request`
- `409 Conflict`

Regle:

- refuser le retry si `retryDecision` ne l'autorise pas.

### 3.4 Cancel

Route:

- `PUT /api/ebics/operations/{operation}/cancel`

### 3.5 Confirm

Route:

- `PUT /api/ebics/operations/{operation}/confirm`

Usage:

- confirmation technique externe;
- notamment pour `WAITING_EXTERNAL_CONFIRMATION`.

## 5. Famille `contract-view`

Routes:

- `GET /api/ebics/partners/{partner}/contract-view`
- `GET /api/ebics/partners/{partner}/contract-view/capabilities`
- `GET /api/ebics/partners/{partner}/contract-view/permissions`
- `GET /api/ebics/partners/{partner}/contract-views`
- `POST /api/ebics/partners/{partner}/contract-view/refresh`

Le `refresh` retourne typiquement:

```json
{
  "operationId": 77,
  "status": "PLANNED"
}
```

## 6. Famille `transactions`

Routes:

- `GET /api/ebics/transactions`
- `GET /api/ebics/transactions/{transaction}`
- `GET /api/ebics/transactions/{transaction}/segments`
- `GET /api/ebics/transactions/{transaction}/segments/{segment}`

Position:

- lecture technique surtout en phase 1;
- utile au diagnostic et a la reprise protocolaire.

## 7. Famille `RTN`

### 6.1 Evenements

Routes:

- `GET /api/ebics/rtn/events`
- `GET /api/ebics/rtn/events/{event}`
- `PUT /api/ebics/rtn/events/{event}/retry`
- `PUT /api/ebics/rtn/events/{event}/quarantine`

### 6.2 Providers

Routes:

- `GET /api/ebics/rtn/providers`
- `POST /api/ebics/rtn/providers`
- `GET /api/ebics/rtn/providers/{provider}`
- `PATCH /api/ebics/rtn/providers/{provider}`

Important:

- la phase 1 vise un provider de transport `WebSocket/WSS`;
- le contrat REST reste agnostique et n'interdit pas d'autres providers plus
  tard.

DTO cible de provider:

```json
{
  "name": "bank-rtn-main",
  "transport": "wss",
  "enabled": true,
  "subscriber": {
    "hostId": "BANKHOST01",
    "partnerId": "PARTNER01",
    "userId": "USER01"
  },
  "configuration": {
    "endpoint": "wss://bank.example.net/rtn",
    "tlsProfile": "default",
    "autoPullPolicy": "auto_filtered"
  }
}
```

## 8. Famille `initialization`

Routes:

- `GET /api/ebics/initializations`
- `GET /api/ebics/initializations/{workflow}`
- `POST /api/ebics/initializations`
- `PUT /api/ebics/initializations/{workflow}/confirm`
- `PUT /api/ebics/initializations/{workflow}/cancel`

## 9. Famille `key-lifecycles`

Routes:

- `GET /api/ebics/key-lifecycles`
- `GET /api/ebics/key-lifecycles/{lifecycle}`
- `POST /api/ebics/key-lifecycles`
- `PUT /api/ebics/key-lifecycles/{lifecycle}/confirm`
- `PUT /api/ebics/key-lifecycles/{lifecycle}/cancel`

## 10. Conventions de reponse

Erreurs cibles:

- `400 Bad Request`
- `404 Not Found`
- `409 Conflict`
- `422 Unprocessable Entity`
- `500 Internal Server Error`

Convention complementaire:

- une erreur EBICS restituee par l'API doit conserver les champs `technical`
  et `business` separes;
- ne pas produire un unique `returnCode` agregat.

Pour les creations ou replanifications:

- retourner `Location` quand pertinent;
- retourner l'identifiant de l'objet cree ou planifie.

## 11. Permissions

Phase 1:

- alignement initial sur `PermTransfersRead` et `PermTransfersWrite` pour
  `operations`, `transactions` et `rtn`.

## 12. Mapping vers futurs handlers

Handlers cibles minimaux:

- `makeEbicsOperationHandlers`
- `makeEbicsContractViewHandlers`
- `makeEbicsRTNHandlers`
- `makeEbicsInitializationHandlers`
- `makeEbicsKeyLifecycleHandlers`

Handlers de phase 2:

- `makeEbicsTransactionHandlers`

## 12. Resultat attendu

Avec ces routes:

- la separation `operations / transfers` reste lisible;
- RTN reste decouple du transport bien que la phase 1 vise `WSS`;
- la derivation vers `pkg/admin/rest/router.go`, `pkg/admin/rest/api/` et les
  handlers devient quasi mecanique.
