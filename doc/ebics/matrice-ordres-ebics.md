# Matrice de decision par ordre EBICS

## 1. Objet

Cette matrice fixe, par ordre EBICS, les decisions d'integration minimales
dans Gateway:

- objet principal (`EbicsOperation` / `Transfer`);
- identifiants a porter;
- politique de replay/retry;
- mode de reprise;
- points d'attention.

Elle sert de reference pour les lots 1 et 2.

## 2. Legende

`Objet principal`

- `EO` = `EbicsOperation`
- `TR` = `Transfer`
- `EO + TR` = operation protocolaire avec projection fichier

`Retry/replay`

- `NR` = non-replayable par defaut
- `MR` = manual-only replay
- `RT` = retryable techniquement
- `RS` = resumable via mecanisme protocolaire

`Reprise`

- `TX` = reprise via transaction/store EBICS
- `SEG` = reprise via segmentation EBICS
- `GW` = reprise fichier Gateway
- `NA` = non applicable

## 3. Matrice

| Ordre | Famille | Objet principal | Identifiants prioritaires | Retry/replay | Reprise | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `HEV` | Capacites version | `EO` | `request_id`, `correlation_id` | `RT` | `NA` | Appel simple, pas de `Transfer` |
| `INI` | Initialisation | `EO` | `request_id`, `correlation_id` | `MR` | `NA` | Rupture d'automatisme possible |
| `HIA` | Initialisation | `EO` | `request_id`, `correlation_id` | `MR` | `NA` | Meme logique que `INI` |
| `H3K` | Initialisation / rotation | `EO` | `request_id`, `correlation_id` | `MR` | `NA` | A traiter aussi dans le cycle de vie des cles |
| `HPB` | Bootstrap confiance | `EO` | `request_id`, `correlation_id` | `RT` | `NA` | Met a jour les bank keys |
| `PUB` | Rotation de cles | `EO` | `request_id`, `correlation_id` | `MR` | `NA` | Pas de rejeu aveugle |
| `HCA` | Rotation de cles | `EO` | `request_id`, `correlation_id` | `MR` | `NA` | Politique banque a verifier |
| `HCS` | Rotation de cles | `EO` | `request_id`, `correlation_id` | `MR` | `NA` | Politique banque a verifier |
| `HSA` | Rotation / certificat | `EO` | `request_id`, `correlation_id` | `MR` | `NA` | Selon profils supportes |
| `SPR` | Revocation / rotation | `EO` | `request_id`, `correlation_id` | `MR` | `NA` | Sensible, pas d'auto-replay |
| `HPD` | Contrat technique | `EO` | `request_id`, `correlation_id` | `RT` | `NA` | Alimente `contract_view` |
| `HKD` | Contrat technique | `EO` | `request_id`, `correlation_id` | `RT` | `NA` | Alimente `contract_view` |
| `HTD` | Contrat technique | `EO` | `request_id`, `correlation_id` | `RT` | `NA` | Alimente `contract_view` |
| `HAA` | Contrat / services | `EO` | `request_id`, `correlation_id` | `RT` | `NA` | Peut aussi alimenter RTN / BTF |
| `FUL` | Upload fichier | `EO + TR` | `transaction_id`, `request_id`, `transfer_id`, `remote_transfer_id`, `correlation_id` | `RS` | `TX + SEG + GW` | Ordre fichier, projection `Transfer` obligatoire |
| `FDL` | Download fichier | `EO + TR` | `transaction_id`, `request_id`, `transfer_id`, `remote_transfer_id`, `correlation_id` | `RS` | `TX + SEG + GW` | Ordre fichier, projection `Transfer` obligatoire |
| `BTU` | Upload BTF | `EO + TR` | `transaction_id`, `request_id`, `transfer_id`, `remote_transfer_id`, `correlation_id` | `RS` | `TX + SEG + GW` | A traiter comme flux fichier |
| `BTD` | Download BTF | `EO + TR` | `transaction_id`, `request_id`, `transfer_id`, `remote_transfer_id`, `correlation_id` | `RS` | `TX + SEG + GW` | A traiter comme flux fichier |
| `HAC` | Reporting / ack | `EO` ou `EO + TR` | `transaction_id`, `request_id`, `correlation_id` | `RT` | `TX` | Devient `EO + TR` seulement si fichier reellement collecte |
| `HVD` | Reporting | `EO` | `transaction_id`, `request_id`, `correlation_id` | `RT` | `TX` | Pas de `Transfer` par defaut |
| `HVE` | Signature protocolaire | `EO` | `transaction_id`, `request_id`, `correlation_id` | `MR` | `TX` | Pas de workflow metier dans Gateway |
| `HVT` | Reporting / signature | `EO` | `transaction_id`, `request_id`, `correlation_id` | `RT` | `TX` | Objet operationnel dedie |
| `HVU` | Reporting / signature | `EO` | `transaction_id`, `request_id`, `correlation_id` | `RT` | `TX` | Objet operationnel dedie |
| `HVZ` | Reporting / signature | `EO` | `transaction_id`, `request_id`, `correlation_id` | `RT` | `TX` | Objet operationnel dedie |
| `HVS` | Signature / annulation | `EO` | `transaction_id`, `request_id`, `correlation_id` | `MR` | `TX` | Action sensible, jamais assimilee a un transfert |

## 4. Regles structurantes

## 4.1 Projection vers `Transfer`

La projection vers `Transfer` n'est autorisee que si:

- un fichier est effectivement transmis ou recupere;
- un pipeline fichier Gateway a une valeur reelle;
- une `Rule` technique peut s'appliquer de maniere legitime.

Sinon:

- l'ordre reste uniquement en `EbicsOperation`.

## 4.2 Politique d'identifiants

Pour les ordres non fichier:

- `EbicsOperation` porte `request_id`, `transaction_id` si applicable,
  `correlation_id`;
- `Transfer` n'intervient pas.

Pour les ordres fichier:

- `EbicsOperation` porte l'identite protocolaire;
- `Transfer` porte l'identite de transfert Gateway;
- la correlation explicite entre les deux est obligatoire.

## 4.3 Politique de replay

Par defaut:

- tout ordre sensible de cycle de vie ou de signature est `MR`;
- les ordres de consultation/refresh sont `RT`;
- les ordres fichier sont `RS` mais via les mecanismes EBICS d'abord.

## 4.4 Politique de reprise

Par defaut:

- la reprise protocolaire passe par `TxStore` et les transactions EBICS;
- la reprise Gateway ne traite que la projection fichier locale;
- `Transfer resume` n'est jamais la seule reponse a un besoin de recovery
  EBICS.

## 5. Decisions a confirmer en spike d'implementation

Points restant a verifier techniquement:

- granularite exacte de `request_id` dans la librairie et sa persistance cible;
- conditions precises de bascule `HAC` vers `EO + TR`;
- politique effective de reprise pour chaque ordre segmente selon la banque;
- besoin ou non d'un identifiant `flow_id` explicite cote EBICS/Gateway.

## 6. Usage recommande

Pendant le developpement:

1. identifier l'ordre EBICS concerne;
2. lire sa ligne dans cette matrice;
3. verifier qu'on ne force pas un `Transfer` la ou seule une `EbicsOperation`
   est legitime;
4. verifier qu'on n'active pas un retry/replay plus agressif que prevu.
