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
- 2026-04-01: le RTN n'est plus seulement administratif.
  Un service de fond `EBICS RTN` est maintenant branche dans `gatewayd`,
  demarre les providers actives, persiste les evenements entrants avec
  idempotence, et programme un vrai `Transfer` Gateway immediat quand un
  auto-pull RTN est derive.
  Ce transfert porte une `EbicsOperation` `AUTO_TRIGGERED` pre-creee, pour
  rester dans le chemin d'execution client deja en place.
  2026-04-01: le dernier maillon d'execution client est maintenant ferme.
  Le scenario de production `RTN -> Transfer planifie -> controller ->
  ClientPipeline -> client EBICS HTTP -> serveur EBICS HTTP -> payload final`
  passe en vert dans `rtn_controller_integration_test.go`.
  Les corrections structurantes portent sur:
  l'abandon du `TransactionID` synthetique sur `BTD`,
  la persistance du vrai `TransactionID` renvoye par la banque,
  la relecture correcte de `ebicsOperationID` en `json.Number`,
  et la restauration de la finalisation standard `EndTransfer()` cote client
  EBICS pour conserver le lien vers l'historique archive.
  2026-04-01: le lot `P0C` est aussi ferme.
  Le statut operateur RTN reste desormais `PROCESSING` tant que l'auto-pull
  n'a pas reellement termine, puis est synchronise avec l'issue finale de
  l'operation cliente (`PROCESSED` / `RETRYABLE` / `FAILED`).
  Les liens et statuts d'auto-pull sont exposes en REST/CLI, et la decision
  de retry deduite des return codes EBICS n'est plus ecrasee a tort dans le
  chemin d'erreur du pipeline client.
- 2026-04-01: les restes a faire post-`B5` sont maintenant convertis en
  backlog explicite dans `suivi-backend-consolidation.md`:
  `P0A` a `P0C` pour RTN reel, `P2A` a `P2D` pour l'automatisation
  d'exploitation native, `P3A` a `P3C` pour le workflow VEU, `P4A` a `P4E`
  pour remettre en ordre l'usage de `TransferInfo` dans l'implementation
  EBICS, et `P5A` a `P5D` pour preparer explicitement le mode
  "Gateway serveur bancaire EBICS".
  `AMQP 0.9.1` / `AMQP 1.0` restent hors perimetre EBICS strict, mais traces
  comme pre-requis du futur passe-plat metier.
  Le lot `P2D` couvre explicitement l'historisation native des ordres EBICS
  non payload (`admin`, `reporting`, `initialisation`, `gestion de cles`),
  avec une exigence d'historique durable et interrogeable analogue a celle
  des transferts Gateway.
  2026-04-03: `P2D` est ferme avec une table append-only
  `ebics_history_entries`, exposee en REST/CLI, sans dupliquer l'historique
  transverse des transferts Gateway.
  Un lot `P2E` est ajoute pour supprimer la limitation artificielle
  "un seul client EBICS actif" sur les chemins non payload:
  la selection doit converger vers la reference canonique deja utilisee dans
  Gateway, a savoir `ClientID`, sauf contrainte absolue specifique a EBICS.
  Le lot `P2B` couvre explicitement le refresh planifie des vues
  contractuelles `HPD` / `HKD` / `HTD` / `HAA`, qui existe aujourd'hui comme
  action client REST/CLI mais pas encore comme orchestration periodique
  native.
  Le chantier `P4` est quant a lui sequence apres `P0C` et vise explicitement
  a sortir les correlations structurelles EBICS de `TransferInfo`, qui reste
  un espace reutilisable par l'exploitant via les variables `#TI_*#`.
  Le chantier `P5` ouvre explicitement l'autre versant fonctionnel:
  si Waarp Gateway doit un jour etre utilise cote banque, le serveur EBICS
  devra depasser `BTU/BTD` et exposer aussi:
  les ordres contractuels/admin (`HPD` / `HKD` / `HTD` / `HAA`),
  les ordres d'initialisation, gestion/rotation de cles,
  les ordres de reporting/signature,
  ainsi qu'un RTN sortant permettant de notifier les partenaires qu'un ordre
  ou document est disponible a la recuperation.
  Une feuille de route globale restante est maintenant posee:
  finir d'abord le socle EBICS exploitable (`P2E`, puis `P2A/B/D/C`),
  traiter ensuite `AMQP 0.9.1` et `AMQP 1.0` comme protocoles Gateway
  autonomes, ouvrir ensuite le passe-plat metier, puis seulement
  l'implementation complete du role banque EBICS (`P5`) et enfin `VEU` (`P3`).
  2026-04-02: `P2E` est maintenant decliné en backlog operationnel.
  La sequence retenue est:
  cartographie des resolutions implicites, choix du contrat REST/CLI cible
  autour de `ClientID`, refactor des chemins admin, alignement RTN, puis
  restitution claire de l'etat activable et des ambiguities de selection.
  L'inventaire `P2E.1` montre deja que le payload standard est correctement
  aligne sur `Transfer.ClientID`, et que l'ecart multi-client se concentre
  sur les chemins non payload (`contract refresh`, reporting/signature,
  initialisation, `HPB`, rotations) ainsi que sur la selection RTN.
  2026-04-02: `P2E.3` est maintenant ferme pour les chemins non payload.
  Les actions `contract refresh`, `reporting`, `signature`,
  `initialisation`, `HPB` et `key rotation` exigent desormais un `clientID`
  explicite en REST/CLI et ne passent plus par une resolution singleton
  implicite du client EBICS. Le prochain lot restant sur ce sujet est `P2E.4`
  pour l'alignement RTN.
  2026-04-02: `P2E.4` est maintenant ferme.
  L'auto-pull RTN converge lui aussi vers `clientID` comme reference
  canonique: le provider RTN administre porte desormais un `clientID`
  explicite, la resolution runtime ne depend plus d'un `clientName`
  optionnel ni d'un balayage des clients EBICS actifs, et les surfaces
  2026-04-02: `P2A` est maintenant ferme.
  Une retention automatisee minimale est desormais branchee dans Gateway via
  un service de maintenance technique EBICS dedie, distinct de la purge
  manuelle d'historique des transferts. La politique active est stockee en
  base dans `ebics_runtime_policies`, pas dans le fichier de configuration,
  et couvre la purge des `nonces` expires, des transactions EBICS seulement
  terminales et anciennes, ainsi que des evenements RTN seulement terminaux
  et anciens. Les transactions encore actives restent explicitement hors
  purge. La policy singleton `default` est maintenant administrable en
  REST/CLI.
  2026-04-03: `P2B` est maintenant ferme.
  Une orchestration planifiee native existe maintenant pour le refresh
  contractuel client `HEV` / `HPD` / `HKD` / `HTD` / `HAA`.
  Les executions sont pilotees par des objets administres
  `EbicsContractRefreshPolicy` stockes en base, relies explicitement a
  `clientID` + `subscriberID`, avec periodicite, statut, prochaine execution,
  dernier essai, dernier succes et dernier message d'erreur.
  Le service `EBICS Contract Refresh` est branche dans `gatewayd`, et la
  surface operateur REST/CLI est exposee via
  `/ebics/contract-refresh-policies` et
  `ebics contract-refresh-policy ...`.
  La passe linter/tests consolidee sur
  `pkg/protocols/modules/ebics/...`, `pkg/admin/rest`, `pkg/cmd/client`,
  `pkg/model`, `pkg/gatewayd` et `pkg/database/migrations` est verte.
  2026-04-03: `P2C` est maintenant ferme.
  La repasse runtime reelle est desormais posee sur deux maillons qui
  manquaient encore:
  un scenario client payload hors RTN qui passe par le vrai
  `controller` + `ClientPipeline`,
  et un scenario serveur payload qui passe par le vrai serveur HTTP EBICS
  via un client `lib-ebics` reel.
  Le scenario RTN auto-pull de bout en bout reste couvert en parallele par
  `rtn_controller_integration_test.go`.
  Les tests et le linter du perimetre
  `pkg/protocols/modules/ebics/... ./pkg/gatewayd ./pkg/model`
  sont verts.
  2026-04-03: `P5A` est maintenant ferme comme lot de cadrage.
  Le perimetre "Gateway en role banque EBICS" est formalise dans
  `gateway-role-banque-ebics.md`:
  priorite aux ordres contractuels `HPD/HKD/HTD/HAA` et au RTN sortant
  minimal, puis aux ordres serveur d'initialisation / gestion / rotation de
  cles, puis aux ordres serveur de reporting / signature.
  Le document fixe aussi les regles d'architecture:
  pas de surcharge de `TransferInfo`,
  pas de logique metier cachee dans les handlers EBICS,
  et usage de projections internes explicites pour alimenter les reponses
  serveur.
  2026-04-03: `P5B` est maintenant ferme.
  Le serveur Gateway EBICS sert desormais `HPD`, `HKD`, `HTD`, `HAA`
  par les handlers `lib-ebics` natifs, alimentes par une projection
  contractuelle serveur issue de `EbicsHost`, `EbicsSubscriber` et
  `EbicsContractViewItem`.
  La preuve passe par un test HTTP reel
  `TestServerHTTPContractOrdersServeConfiguredViews` qui telecharge les
  quatre ordres via un client `lib-ebics` reel contre le serveur Gateway.
  2026-04-03: `P5C` est maintenant ferme.
  La projection contractuelle serveur est maintenant isolee dans
  `EbicsServerContractSet` / `EbicsServerContractItem`,
  distincte des vues contractuelles client.
  Le bornage fonctionnel est impose au niveau modele:
  `HPD/HAA` scopes host,
  `HKD/HTD` scopes subscriber.
  Une lecture REST/CLI minimale est disponible via
  `server-contract-sets`, et la repasse qualite ciblee est verte.
  REST/CLI des providers RTN exposent cette reference explicitement.
  2026-04-02: `P2E.5` est maintenant ferme.
  Les surfaces REST/CLI des providers RTN exposent desormais un etat
  d'activation lisible (`activationStatus`, `activationReason`) en plus du
  `clientID` / `clientName` selectionne, ce qui rend visible le perimetre
  activable et les cas bloques cote multi-client.
  2026-04-01: `P4A` est maintenant ferme.
  L'inventaire confirme que le probleme n'est pas limite a quelques cles
  EBICS (`ebicsOperationID`, `ebicsTransactionID`, etc.): le chemin RTN clone
  aussi le `PayloadMap` brut des evenements dans `TransferInfo`, ce qui expose
  potentiellement des metadonnees techniques arbitraires via `#TI_*#`.
  Le chantier `P4B` devra donc definir un modele cible complet, sans aucune
  dependance runtime critique a `TransferInfo`.
  2026-04-01: `P4B` est maintenant ferme.
  Le modele cible est arrete:
  `Transfer <-> EbicsOperation` via `ebics_operations.transfer_id`,
  `EbicsOperation <-> EbicsRTNEvent` via `rtn_event_id`,
  `EbicsOperation <-> EbicsTransaction` via `ebics_transactions`.
  Les informations techniques de resolution migrent vers
  `ebics_operations.metadata`, le message RTN complet reste dans
  `ebics_rtn_events.payload`, et `TransferInfo` n'est plus autorise comme
  support de correlation interne EBICS ni comme receptacle de clonage RTN.
  2026-04-01: premiere tranche `P4C/P4D` engagee et validee.
  Le runtime client payload relit maintenant son contexte critique via
  l'operation liee et non plus via `TransferInfo`; le pass-through RTN brut
  vers `TransferInfo` a ete coupe; un contexte dedie `ebicsContext` est expose
  dans l'API `transfers`; et des variables de taches `#EBICS_*#` sont
  disponibles via `replacer.go`.
  Aucun fallback de compatibilite interne EBICS n'est retenu a ce stade:
  l'implementation du protocole doit converger directement vers le modele
  cible propre.
  Point de cadrage ajoute: le nettoyage vise les cles techniques EBICS / RTN.
  Les metadonnees standard du moteur Gateway comme `__followID__` restent hors
  de ce perimetre et ne doivent pas etre confondues avec une derive EBICS.
  2026-04-01: `P4C` est maintenant ferme.
  Les lectures techniques EBICS residuelles ont ete retirees du runtime
  payload; le contexte EBICS est recharge uniquement depuis les tables et
  metadata dediees; les chemins non payload `admin/reporting/key rotation/init`
  ont ete relus et n'utilisent pas `TransferInfo` a mauvais escient.
  2026-04-01: `P4D` est maintenant ferme.
  Les surfaces REST/CLI `history` sont alignees sur `transfers` avec un bloc
  dedie `ebicsContext`, y compris pour les transferts archives via
  `metadata.archivedTransferID`.
  Les anciennes constantes EBICS `transferInfoKey*` ont ete retirees du code
  de production. `TransferInfo` ne porte plus de correlation technique EBICS;
  seules restent les metadonnees natives du moteur Gateway comme `__followID__`.
  2026-04-01: la piste PowerShell/linter est maintenant clarifiee.
  Le probleme venait du shell sandboxe execute sous un compte technique
  (`CodexSandboxOffline`) sans acces a la config Git utilisateur, pas d'une
  casse durable du depot. Hors sandbox, sous `pwsh 7.6.0` et le vrai compte
  utilisateur, `golangci-lint` fonctionne normalement sur le depot.
  2026-04-01: `P4E` est maintenant ferme.
  La passe `go test` consolidee et la repasse `golangci-lint` hors sandbox sur
  `pkg/protocols/modules/ebics/...`, `pkg/admin/rest/...`, `pkg/cmd/client`,
  `pkg/model` et `pkg/gatewayd` sont vertes.
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

- [x] Plan de consolidation backend pose
- [x] Lot B1 - Execution cliente reelle
- [x] Lot B2 - Couverture backend complete
- [x] Lot B3 - Import / export / updateconf complet
- [x] Lot B3.5 - Catalogue BTF standard
- [x] Lot B4 - Durcissement exploitation
- [x] Lot B5 - Verification de sortie backend
- [ ] Gate "backend pret frontend" prononcee

Note:
- le suivi detaille est porte par `backend-consolidation-plan.md` et `suivi-backend-consolidation.md`
- objectif explicite: ne plus laisser de stub bloquant ni de fonctionnalite backend EBICS partielle avant frontend
- point de situation rejoue le 2026-03-31:
  `B1`, `B2`, `B3` et `B3.5` restent fermes;
  `B4` et `B5` restent ouverts;
  les principaux ecarts restants sont maintenant concentres sur le serveur EBICS,
  l'observabilite / l'exploitation et la passe finale de verification de sortie
- point de situation rejoue le 2026-04-01:
  `B4` et `B5` sont maintenant fermes sur le backend EBICS strict avec
  repasse linter/tests complete, relecture des specs et refus explicite de la
  gate frontend a l'echelle de la cible documentaire globale.
  Le motif principal restant est l'absence du passe-plat asynchrone metier et
  des protocoles natifs `AMQP 0.9.1` / `AMQP 1.0`, identifies comme attente
  minimale ou prealable architectural dans les documents de specification.
  Arbitrage retenu: ces protocoles doivent etre implementes comme protocoles
  Gateway autonomes hors perimetre EBICS strict, mais restent un pre-requis
  imperatif du futur chantier de passe-plat metier.
- ce point de situation doit maintenant etre relu contre
  `specifications-fonctionnelles.md`, `specifications-techniques.md` et
  `architecture-logicielle.md`, car une lecture centree seulement sur le code
  courant sous-estime encore les attentes de passe-plat metier, d'observabilite,
  de validation serveur et de verification finale
- le chantier de consolidation serveur EBICS est maintenant rendu explicite dans `Lot B4`,
  pour ne pas laisser un angle mort cote provider/serveur
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
- `Lot B3` est demarre:
  `pkg/backup` couvre maintenant les objets EBICS administres
  `hosts`, `subscribers`, `bank keys`, `payload profiles` et `RTN providers`;
  la verification de round-trip complet JSON/YAML et `updateconf` reste ouverte
- `Lot B3` est termine:
  import/export couvrent les objets EBICS administres
  `hosts`, `subscribers`, `bank keys`, `payload profiles` et `RTN providers`;
  des jeux de reference `JSON/YAML` et un test de round-trip dedie valident
  le perimetre `pkg/backup`; `updateconf` est aussi verifie sur ce socle;
  la migration `0.16.0` garantit la presence des tables EBICS dans les bases
  existantes et dans les bases de test
- `Lot B3.5` est termine:
  Gateway porte maintenant un catalogue BTF standard `GLB/FR/DE/AT/CH`,
  distinct des contrats specifiques recuperes via `HPD/HKD/HTD/HAA`;
  le seed est versionne, restaure sur base fraiche et sur base migree,
  couvert par import/export/updateconf, et la resolution runtime applique
  strictement `specific > country > GLB`;
  si un contrat specifique actif existe et qu'un tuple n'y est pas trouve,
  l'echange est rejete sans fallback vers le catalogue standard
