# Etats de signature protocolaire et restitution REST/CLI/UI

## 1. Objet

Ce document fixe un modele cible pour les etats de signature protocolaire
EBICS et leur restitution dans Gateway.

Il couvre principalement:

- `HVD`
- `HVE`
- `HVT`
- `HVU`
- `HVZ`
- `HVS`

## 2. Principes

- les etats stockes par Gateway doivent rester techniques et protocolaires;
- ils ne doivent pas devenir un workflow humain metier;
- ils doivent etre lisibles pour l'exploitation;
- ils doivent etre exploitables pour le passe-plat metier.

## 3. Modele d'etat cible

### 3.1 Etat principal d'operation

L'`EbicsOperation.status` reste l'etat principal d'execution:

- `PLANNED`
- `RUNNING`
- `WAITING_BANK`
- `COMPLETED`
- `COMPLETED_WITH_WARNINGS`
- `FAILED`
- `CANCELLED`

### 3.2 Etat de signature dedie

Pour les ordres lies a la signature, ajouter un sous-etat logique:

- `NOT_APPLICABLE`
- `SIGNATURES_UNKNOWN`
- `WAITING_SIGNATURES`
- `SIGNATURE_PARTIALLY_AVAILABLE`
- `SIGNATURE_READY`
- `SIGNATURE_ADDED`
- `SIGNATURE_CANCELLED`
- `SIGNATURE_REJECTED`
- `SIGNATURE_INVALID`

Ce sous-etat peut vivre:

- soit comme champ dedie dans `EbicsOperation.metadata` en phase 1;
- soit comme champ first-class si la profondeur fonctionnelle l'exige.

## 4. Interpretation par ordre

### 4.1 `HVU`, `HVZ`, `HVD`, `HVT`

Ces ordres servent surtout a constater et exposer.

Sous-etats typiques:

- `SIGNATURES_UNKNOWN`
- `WAITING_SIGNATURES`
- `SIGNATURE_PARTIALLY_AVAILABLE`
- `SIGNATURE_READY`

### 4.2 `HVE`

Ordre d'ajout technique de signature.

Sous-etats typiques:

- `WAITING_SIGNATURES`
- `SIGNATURE_READY`
- `SIGNATURE_ADDED`
- `SIGNATURE_REJECTED`
- `SIGNATURE_INVALID`

### 4.3 `HVS`

Ordre d'annulation technique.

Sous-etats typiques:

- `SIGNATURE_ADDED`
- `SIGNATURE_CANCELLED`
- `SIGNATURE_REJECTED`

## 5. DTO REST cibles

Le detail d'une operation de signature devrait exposer:

```json
{
  "id": 120,
  "orderType": "HVE",
  "status": "COMPLETED",
  "signatureState": "SIGNATURE_ADDED",
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
  "signatureInfo": {
    "transactionId": "A1B2C3",
    "requestId": "req-789",
    "signerReference": "USER01",
    "detailsAvailable": true
  }
}
```

## 6. Restitution CLI

La CLI doit faire ressortir:

- l'ordre EBICS;
- l'etat principal d'operation;
- le `signatureState`;
- les scopes `technical` et `business`;
- l'action possible ou non.

Exemple:

```text
=== EBICS Operation ===
-Operation 120 (HVE / SIGNATURE) [COMPLETED]
  -Signature state: SIGNATURE_ADDED
  -Technical return code: 000000
  -Business return code: 000000
  -Gateway outcome: SUCCESS
  -Retry decision: NO_RETRY
```

## 7. Restitution UI

La vue UI doit permettre:

- filtrage par `signatureState`;
- vue rapide des ordres en attente de signature;
- distinction claire entre:
  - fait technique constate
  - action technique possible
  - decision metier attendue ailleurs.

## 8. Evenements vers le metier

Le passe-plat metier doit transporter des faits techniques du type:

- `signature_waiting`
- `signature_ready`
- `signature_added`
- `signature_cancelled`
- `signature_rejected`

Il ne doit pas pretendre transporter:

- une decision humaine deja qualifiee metierement au nom de Gateway.

## 9. Retry / replay

Regles:

- `HVE` et `HVS` restent des ordres sensibles;
- par defaut, pas d'auto-retry aveugle;
- toute reprise doit etre fortement contrainte par `retryDecision` et les
  return codes.

## 10. Decision recommandee

La bonne cible est:

- un `status` principal d'operation;
- un `signatureState` dedie pour les ordres de signature;
- une restitution REST/CLI/UI explicite;
- un passe-plat d'evenements techniques vers le metier.
