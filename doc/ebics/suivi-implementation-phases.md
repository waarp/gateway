# Suivi d'implementation par phases

## 1. Vue d'ensemble

Ce document sert de tableau de progression synthétique.

Utilisation:

- cocher une phase quand ses prerequis et son socle sont termines;
- ajouter une note courte si la phase est partiellement ouverte;
- ne pas cocher une phase si ses invariants structurants ne sont pas tenus.

## 2. Phases

## Phase A - Socle modele + config

- [x] Modeles `EbicsHost`, `EbicsSubscriber`, `EbicsBankKey` poses
- [x] Module `ebics` enregistrable
- [x] `ProtoConfig` validables
- [x] Bootstrap serveur/client propre
- [x] `updateconf` compatible

Note:
- socle code pose et compilation ciblee validee sur `pkg/model`, `pkg/protocols` et `pkg/protocols/modules/ebics`
- `updateconf` s'appuie deja sur le transport generique `ProtoConfig` de `pkg/backup`
- `golangci-lint` realigne sur `go1.26.1`, puis passe avant la compilation ciblee

## Phase B - Payload profiles + contract view

- [x] `EbicsContractView` pose
- [x] `EbicsContractViewItem` pose
- [x] `EbicsPayloadProfile` pose
- [x] Resolution payload implemente
- [x] Validation contractuelle implemente

Note:
- DTO REST `payload profiles` / `contract views` poses
- import/export `backup/updateconf` etendu pour les `payload profiles`
- linter et compilations ciblees valides sur `pkg/model`, `pkg/protocols/modules/ebics`, `pkg/admin/rest/api` et `pkg/backup`

## Phase C - Operations + payload requests

- [ ] `EbicsOperation` pose
- [ ] `EbicsTransaction` pose
- [ ] `EbicsTransactionSegment` pose
- [ ] Mapping payload -> operation implemente
- [ ] Policy retry/replay/recovery implemente
- [ ] Projection vers `Transfer` bornee

Note:

## Phase D - Workflows sensibles

- [ ] `EbicsKeyLifecycle` pose
- [ ] `EbicsInitializationWorkflow` pose
- [ ] `signatureState` centralise
- [ ] Transitions runners bornees
- [ ] Protection des `Credential` references

Note:

## Phase E - RTN

- [ ] `EbicsRTNEvent` pose
- [ ] `EbicsRTNProvider` pose
- [ ] Provider `WSS` pose
- [ ] Idempotence durable
- [ ] Auto-pull trace

Note:

## 3. Jalons transverses

- [ ] REST EBICS minimal exploitable
- [ ] CLI EBICS minimale exploitable
- [x] Import/export/updateconf coherents pour le socle `ProtoConfig` de la Phase A
- [ ] Documentation de dev a jour
- [ ] Dossier EBICS toujours coherent avec les specs

## 4. GO Implementation

- [ ] Les phases A a E sont suffisamment stables pour lancer l'implementation large

Decision / date:
