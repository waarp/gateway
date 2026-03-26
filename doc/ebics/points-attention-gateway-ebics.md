# Points d'attention restants entre Gateway et EBICS

## 1. Objet

Ce document consolide trois zones de friction encore ouvertes entre l'existant
Gateway et l'integration EBICS:

- gestion des identifiants;
- retry Gateway versus replay/non-replay EBICS;
- reprise Gateway versus segmentation/recovery EBICS.

L'objectif est de verrouiller les ambiguities avant passage au design plus
technique.

## 2. Gestion des identifiants

## 2.1 Constat sur l'existant Gateway

L'existant Gateway repose deja sur plusieurs identifiants:

- `Transfer.ID`: identifiant local base de donnees;
- `Transfer.RemoteTransferID`: identifiant externe ou protocolaire de transfert;
- `FollowID` dans certains protocoles comme `R66`;
- une notion de `flow` connue fonctionnellement, mais qui n'apparait pas ici
  comme un objet pivot du modele de persistance consulte.

References utiles:

- [transfer.go](/c:/MonProjet/Waarp-Gateway/pkg/model/transfer.go)
- [history.go](/c:/MonProjet/Waarp-Gateway/pkg/model/history.go)

## 2.2 Constat EBICS

EBICS introduit ses propres identifiants techniques selon les phases:

- `TransactionID`
- `RequestID`
- identifiants de correlation ou d'evenement selon les cas d'usage
- identifiants RTN / outbox / integration metier a conserver localement

## 2.3 Conclusion de design

Il ne faut surtout pas chercher un identifiant unique commun.

La bonne approche est une grille d'identites distinctes:

- identifiants Gateway d'exploitation:
  - `transfer_id`
  - `remote_transfer_id`
  - eventuel `flow_id` si un objet de flow est materialise plus tard
- identifiants EBICS:
  - `transaction_id`
  - `request_id`
  - `order_type`
- identifiants d'integration:
  - `correlation_id`
  - `event_id`
  - `outbox_delivery_id`

Recommendation:

- `EbicsOperation` doit devenir le point de convergence de ces identifiants;
- `Transfer` ne doit porter que les identifiants utiles a son propre cycle de
  vie;
- la correlation doit etre explicite, jamais implicite.

## 2.4 Decision recommandee

- `Transfer.ID` reste l'identifiant local du transfert;
- `RemoteTransferID` reste l'identifiant de transfert expose aux protocoles
  Gateway qui en ont besoin;
- `EbicsOperation` porte `transaction_id`, `request_id`, `correlation_id`;
- si un concept `Flow` doit exister pour EBICS, il doit etre un objet ou au
  minimum un identifiant dedie, et non une surcharge de `RemoteTransferID`.

## 3. Retry Gateway versus replay EBICS

## 3.1 Constat sur l'existant Gateway

Gateway possede une logique native de retry cote transfert:

- `RemainingTries`
- `NextRetryDelay`
- `RetryIncrementFactor`
- `NextRetry`

Le controleur reprogramme ensuite les transferts eligibles.

References utiles:

- [transfer.go](/c:/MonProjet/Waarp-Gateway/pkg/model/transfer.go)
- [client_transfers.go](/c:/MonProjet/Waarp-Gateway/pkg/controller/client_transfers.go)

## 3.2 Risque d'inadequation avec EBICS

En EBICS, rejouer n'est pas neutre.

Il faut distinguer:

- retenter une connexion ou une requete interrompue;
- reprendre une transaction EBICS existante;
- relancer un ordre de maniere propre en creant une nouvelle operation;
- ne surtout pas rejouer un ordre quand le protocole ou l'etat banque ne le
  permet pas.

Autrement dit:

- le retry automatique natif Gateway ne peut pas etre applique aveuglement aux
  operations EBICS.

## 3.3 Decision recommandee

Pour EBICS:

- desactiver toute interpretation naive du retry de `Transfer` pour les ordres
  non payload;
- porter la decision de replay/retry dans `EbicsOperation`;
- classer chaque type d'ordre EBICS selon une politique explicite:
  - `retryable`
  - `resumable`
  - `manual-only`
  - `non-replayable`

Consequences:

- les flux fichier EBICS projetes en `Transfer` peuvent continuer a utiliser le
  moteur Gateway de retry, mais seulement si la politique EBICS associee le
  permet;
- les operations EBICS administratives doivent etre replanifiees via
  `EbicsOperation`, pas via `Transfer retry`.

## 4. Reprise Gateway versus segmentation EBICS

## 4.1 Constat sur l'existant Gateway

Le pipeline Gateway gere une reprise orientee flux de donnees:

- reprise a partir de `Transfer.Progress`;
- reprise du stream de fichier;
- reprise des taches selon l'etape courante.

Reference utile:

- [pipeline.go](/c:/MonProjet/Waarp-Gateway/pkg/pipeline/pipeline.go)

## 4.2 Constat EBICS

EBICS gere nativement:

- segmentation;
- recovery transactionnel;
- suivi des segments et de leur etat;
- stores dedies (`TxStore`).

La granularite n'est donc pas la meme.

Gateway raisonne principalement en:

- transfert;
- progression octets;
- etapes de pipeline.

EBICS raisonne aussi en:

- transaction protocolaire;
- segment;
- sequence de requetes/reponses.

## 4.3 Decision recommandee

Il faut eviter de confondre les deux mecanismes.

Regle de conception:

- la reprise protocolaire EBICS s'appuie d'abord sur `TxStore`,
  `ebics_transactions` et `ebics_transaction_segments`;
- la reprise Gateway par `Transfer.Progress` n'est valable que pour la portion
  fichier locale ou pour un flux deja projete en transfert;
- `Transfer resume` ne doit pas etre l'API de reprise principale d'une
  transaction EBICS segmentee.

## 4.4 Consequence

Pour un flux EBICS segmente:

- `EbicsOperation` pilote l'etat protocolaire;
- `ebics_transactions` et `ebics_transaction_segments` pilotent la reprise
  segment par segment;
- un `Transfer` associe, s'il existe, ne represente que la projection fichier.

## 5. Synthese des decisions

- les identifiants Gateway et EBICS doivent coexister, pas fusionner;
- `EbicsOperation` devient le pivot de correlation;
- le retry automatique natif Gateway n'est pas applicable tel quel a tous les
  ordres EBICS;
- la reprise EBICS doit rester pilotee par le modele transactionnel EBICS, pas
  par le seul `Transfer.Progress`.

## 6. Impacts backlog

Ces points doivent etre fermes dans les lots 1 et 2:

- politique d'identifiants;
- politique de replay/retry par type d'ordre;
- politique de reprise et de segmentation.
