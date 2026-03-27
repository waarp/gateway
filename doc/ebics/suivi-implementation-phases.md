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
- dependance `lib-ebics` publiee puis referencee sans `replace` local, avec bootstrap client et provider store Gateway valides

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

- [x] `EbicsOperation` pose
- [x] `EbicsTransaction` pose
- [x] `EbicsTransactionSegment` pose
- [x] Mapping payload -> operation implemente
- [x] Policy retry/replay/recovery implemente
- [x] Projection vers `Transfer` bornee

Note:
- stores d'operations/transactions poses sous forme d'interfaces explicites
- DTO REST `payload requests`, `operations` et `transactions` poses
- linter et compilations ciblees valides sur `pkg/model`, `pkg/protocols/modules/ebics/...` et `pkg/admin/rest/api`

## Phase D - Workflows sensibles

- [x] `EbicsKeyLifecycle` pose
- [x] `EbicsInitializationWorkflow` pose
- [x] `signatureState` centralise
- [x] Transitions runners bornees
- [x] Protection des `Credential` references

Note:
- DTO REST `key lifecycles` et `initializations` poses
- protection des suppressions/modifications sensibles de `Credential` posee au niveau model
- linter et compilations ciblees valides sur `pkg/model`, `pkg/protocols/modules/ebics/...` et `pkg/admin/rest/api`

## Phase E - RTN

- [x] `EbicsRTNEvent` pose
- [x] `EbicsRTNProvider` pose
- [x] Provider `WSS` pose
- [x] Idempotence durable
- [x] Auto-pull trace

Note:
- DTO REST `RTN` et commandes CLI RTN poses
- provider `WSS` borne, avec reconnexion et normalisation d'evenements
- linter et compilations ciblees valides sur `pkg/model`, `pkg/protocols/modules/ebics/...`,
  `pkg/admin/rest/api` et `pkg/cmd/client`

## 3. Jalons transverses

- [x] REST EBICS minimal exploitable
- [x] CLI EBICS minimale exploitable
- [x] Import/export/updateconf coherents pour le socle `ProtoConfig` de la Phase A
- [x] Documentation de dev a jour
- [x] Dossier EBICS toujours coherent avec les specs

Note:
- socle REST pose pour `payload profiles`, `contract views`, `operations`, `transactions`,
  `payloads`, `key lifecycles`, `initializations` et `RTN`
- arbre CLI `ebics` branche dans `cmd/waarp-gateway/main.go` avec `operation`, `payload`,
  `contract-view`, `key-lifecycle`, `initialization` et `rtn`

## 4. GO Implementation

- [x] Les phases A a E sont suffisamment stables pour lancer l'implementation large

Decision / date:
- `GO implementation large` / 2026-03-26

## 5. Consolidation backend avant frontend

- [ ] Plan de consolidation backend pose
- [x] Lot B1 - Execution cliente reelle
- [x] Lot B2 - Couverture backend complete
- [ ] Lot B3 - Import / export / updateconf complet
- [ ] Lot B4 - Durcissement exploitation
- [ ] Lot B5 - Verification de sortie backend
- [ ] Gate "backend pret frontend" prononcee

Note:
- le suivi detaille est porte par `backend-consolidation-plan.md` et `suivi-backend-consolidation.md`
- objectif explicite: ne plus laisser de stub bloquant ni de fonctionnalite backend EBICS partielle avant frontend
- `Lot B1` est termine:
  le chemin nominal client payload `BTU/BTD` est branche sur `lib-ebics`
  avec correlation `operation/transaction/transfer`, contrat actif, TLS et recovery;
  les ordres client hors payload couvrent maintenant initialisation, refresh contractuel,
  reporting, signatures protocolaires et rotation coordonnee des cles;
  `FUL/FDL` restent de simples alias de compatibilite vers `BTU/BTD`
- `Lot B2` est demarre:
  premiers ecarts de couverture CLI fermes avec `ebics transaction list/get`,
  `ebics payload profile delete` et `ebics rtn provider delete`;
  la famille `transactions` couvre maintenant aussi les segments en REST/CLI;
  la famille `operations` est alignee comme vue d'observabilite detaillee et
  d'actions specialisees (`reporting` / `signature`), sans facade generique
  `retry/cancel/confirm`;
  la revue exploitable est maintenant bouclee pour `contract views`,
  `payload profiles`, `initializations`, `key lifecycles`, `RTN` et `payloads`;
  les operations payload sont explicitement typees `PAYLOAD` au lieu de
  reutiliser `REPORTING`;
  `Lot B2` est maintenant considere ferme
