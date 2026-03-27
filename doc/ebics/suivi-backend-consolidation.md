# Suivi de consolidation backend EBICS

## 1. Usage

Cette checklist sert a piloter la fermeture du backend EBICS avant le frontend.

Regles:

- cocher `[x]` quand l'item est termine, relu et valide techniquement;
- laisser `[ ]` si l'item n'est pas encore ferme;
- utiliser `[-]` pour un report explicite;
- documenter toute divergence structurelle.

## 2. Gate de sortie backend

- [x] Plus aucun `ErrNotImplemented` sur le chemin nominal EBICS
- [ ] Plus aucun endpoint/commande EBICS expose sans logique runtime suffisante
- [ ] Plus aucun `replace` local vers `lib-ebics`
- [x] Import/export/updateconf complets pour les objets EBICS administres
- [x] Catalogue BTF standard disponible pour les validations pre-contractuelles
- [ ] Politique d'exploitation documentee et relue
- [ ] Backend declare "pret frontend"

## 3. Lot B1 - Execution cliente reelle

- [x] Remplacer le stub `InitTransfer` dans `pkg/protocols/modules/ebics/client.go`
- [x] Definir le mapping `Transfer -> ordre client EBICS`
- [x] Creer la creation d'`EbicsOperation` cote client
- [x] Creer la creation d'`EbicsTransaction` cote client quand necessaire
- [x] Brancher `BTU/BTD` cote client
- [x] Confirmer que `FUL/FDL` restent des alias de compatibilite normalises vers `BTU/BTD` en cible `EBICS 3.0.2`
- [x] Brancher `HEV` et le refresh contractuel `HPD` / `HKD` / `HTD` / `HAA` cote client
- [x] Brancher le reste du reporting / des ordres admin cote client
- [x] Brancher initialisation / key management cote client
- [x] Garantir la correlation `operation / transaction / transfer`
- [x] Verifier l'exploitation des return codes `technical/business`

## 4. Lot B2 - Couverture backend complete

- [x] Revoir chaque famille REST EBICS et confirmer l'absence de logique partielle
- [x] Revoir chaque famille CLI EBICS et confirmer l'absence de logique partielle
- [x] Verifier que `payloads` est bien exploitable de bout en bout
- [x] Verifier que `operations` est bien exploitable de bout en bout
- [x] Verifier que `transactions` est bien exploitable de bout en bout
- [x] Verifier que `contract views` est bien exploitable de bout en bout
- [x] Verifier que `payload profiles` est bien exploitable de bout en bout
- [x] Verifier que `initializations` est bien exploitable de bout en bout
- [x] Verifier que `key lifecycles` est bien exploitable de bout en bout
- [x] Verifier que `RTN` est bien exploitable de bout en bout

## 5. Lot B3 - Import / export / updateconf

- [x] Etendre `pkg/backup/export.go`
- [x] Etendre `pkg/backup/import.go`
- [x] Ajouter les helpers `*_export.go`
- [x] Ajouter les helpers `*_import.go`
- [x] Cadrer les jeux JSON/YAML de reference
- [x] Verifier le round-trip complet des `ProtoConfig`
- [x] Verifier le round-trip complet des objets EBICS administres

## 6. Lot B4 - Durcissement exploitation

## 5 bis. Lot B3.5 - Catalogue BTF standard

- [x] Cadrer l'objet de persistance dedie
- [x] Arreter la source initiale `GLB`
- [x] Arreter la source initiale `FR`, `DE`, `AT`, `CH`
- [x] Figer la regle `specific > country > GLB`
- [x] Cadrer la strategie de seed versionne
- [x] Cadrer les surfaces REST/CLI minimales
- [x] Cadrer le fallback runtime dans `contract_validation`

## 6. Lot B4 - Durcissement exploitation

- [ ] Revoir l'execution serveur EBICS reelle sur le chemin nominal `BTU/BTD`
- [ ] Revoir la couverture normative des ordres serveur non payload exposes
- [ ] Revoir la segmentation / reprise / recovery cote serveur
- [ ] Revoir la journalisation des flux serveur EBICS
- [ ] Revoir la journalisation des flux client EBICS
- [ ] Revoir les messages d'erreur REST EBICS
- [ ] Revoir les messages CLI EBICS
- [ ] Revoir les statuts operateur visibles
- [ ] Revoir la purge / retention des nonces
- [ ] Revoir la purge / retention des transactions
- [ ] Revoir la purge / retention des evenements RTN
- [ ] Revoir la coherences des reprises / recovery
- [ ] Revoir la discipline multi-SGBD / XORM
- [ ] Revoir les protections de suppression / mutation sur objets sensibles

## 7. Lot B5 - Verification de sortie

- [ ] Rejouer une passe `rg ErrNotImplemented|not implemented` et solder tous les cas EBICS
- [ ] Rejouer une passe linter sur le perimetre backend EBICS
- [ ] Rejouer une passe compilation/test ciblee sur le perimetre backend EBICS
- [ ] Revoir les documents de suivi
- [ ] Declarer la gate backend "GO frontend"

## 8. Notes

- Date de creation: 2026-03-27
- Cible: backend EBICS complet avant chantier frontend
- 2026-03-27: `Lot B1` est entame et couvre maintenant le chemin nominal payload client
  `BTU/BTD` avec creation `EbicsOperation` / `EbicsTransaction`, contrat actif,
  TLS, recovery et correlation `transfer`.
  La cible `EBICS 3.0.2` est maintenant figee: `BTU/BTD` sont canoniques,
  `FUL/FDL` restent de simples alias de compatibilite normalises.
  Reste a fermer dans `B1`: les familles client
  reporting/admin/initialisation/key-management.
- 2026-03-27: le client hors payload couvre maintenant une execution reelle des
  actions d'initialisation `INI` / `HIA` / `H3K` et de la synchronisation banque
  `HPB`, avec creation d'`EbicsOperation`, persistance des references dans
  `EbicsInitializationWorkflow`, generation de la lettre `H3K` et persistance des
  cles banque.
- 2026-03-27: le client hors payload couvre maintenant aussi `HEV` et le refresh
  contractuel `HPD` / `HKD` / `HTD` / `HAA`, avec persistance de snapshots dans
  `EbicsContractView` / `EbicsContractViewItem`, exposition REST/CLI de l'action
  de refresh, et correction du modele `EbicsOperation` pour accepter les ordres
  non payload.
- 2026-03-27: le client hors payload couvre maintenant aussi le reporting
  `HVD` / `HVU` / `HVZ` / `HVT` / `HAC` et les signatures protocolaires
  `HVE` / `HVS`, avec actions REST/CLI dediees sur `ebics operation`,
  execution reelle via `lib-ebics`, persistance d'`EbicsOperation` et
  metadonnees d'exploitation.
- 2026-03-27: le key management de rotation client couvre maintenant une
  orchestration coordonnee `PUB` / `HCA` / `HCS` / `HSA` / `SPR`, avec
  coexistence `ACTIVATED + pending`, `coordinationID`, actions REST/CLI
  `prepare/send/confirm/cancel/reject/revoke`, et activation qui retire
  explicitement l'ancien lifecycle au profit du nouveau.
  `Lot B1` est maintenant ferme sur le perimetre d'execution cliente reelle.
- 2026-03-27: `Lot B2` est demarre avec fermeture des premiers ecarts
  objectifs de couverture CLI par rapport au backend REST:
  ajout de `ebics transaction list/get`, ajout de `ebics payload profile delete`
  et ajout de `ebics rtn provider delete`.
  Le lot reste ouvert tant que la revue complete REST/CLI et l'exploitation
  bout en bout de chaque famille EBICS n'ont pas ete bouclees.
- 2026-03-27: une passe `rg ErrNotImplemented|not implemented` sur
  `pkg/protocols/modules/ebics`, `pkg/admin/rest` et `pkg/cmd/client`
  ne remonte plus aucun cas EBICS actif.
- 2026-03-27: la famille `transactions` couvre maintenant aussi la lecture
  detaillee des segments en REST et en CLI:
  `GET /api/ebics/transactions/{transaction}/segments`,
  `GET /api/ebics/transactions/{transaction}/segments/{segment}`,
  `waarp-gateway ebics transaction segments` et
  `waarp-gateway ebics transaction segment`.
- 2026-03-27: la famille `operations` est maintenant explicitee comme une
  famille d'observabilite et d'actions specialisees:
  detail enrichi avec transaction, segments, identites et liens techniques;
  maintien des actions `reporting` et `signature`;
  abandon assume d'une facade generique `retry/cancel/confirm`, ces actions
  restant portees par les familles specialisees.
- 2026-03-27: les familles `contract views`, `payload profiles`,
  `initializations`, `key lifecycles` et `RTN` ont ete relues cote REST et
  CLI. Leur couverture est consideree suffisante pour `B2`, avec enrichissement
  des sorties operateur (items contractuels, metadonnees de profil, horodatages
  et references d'operations/workflows). Le point restant principal du lot est
  maintenant la verification bout en bout de `payloads`, puis la cloture
  synthese des familles REST/CLI EBICS.
- 2026-03-27: la famille `payloads` est maintenant consideree exploitable de
  bout en bout pour `B2`, avec soumission REST/CLI, controle contractuel,
  creation/correlation d'`EbicsOperation`, typage d'operation `PAYLOAD`
  explicite cote client et cote serveur, et actions operateur `retry/recover`.
  Le `Lot B2` est considere ferme.
- 2026-03-27: le plan de consolidation est precise pour rendre explicite le
  chantier serveur. Le durcissement de l'execution serveur reelle EBICS
  (payload nominal, ordres non payload, segmentation/reprise, observabilite)
  est maintenant porte explicitement par le `Lot B4`.
- 2026-03-27: `Lot B3` est demarre. Le backup EBICS couvre maintenant les
  objets administres `hosts`, `subscribers`, `bank keys`, `payload profiles`
  et `RTN providers`, avec orchestration d'import/export dans `pkg/backup`.
  Les objets purement operationnels (`operations`, `transactions`,
  `initializations`, `key lifecycles`, `RTN events`, `contract views`)
  restent volontairement hors sauvegarde de configuration. Les round-trips
  complets JSON/YAML et `updateconf` restent a verifier.
- 2026-03-27: `Lot B3` est maintenant ferme. Le perimetre EBICS administre
  (`hosts`, `subscribers`, `bank keys`, `payload profiles`, `RTN providers`)
  est couvert en import/export, avec jeux de reference `JSON/YAML`,
  test de round-trip dedie dans `pkg/backup`, verification `updateconf`,
  et migration `0.16.0` pour garantir la presence des tables EBICS sur les
  bases creees ou migrees.
- 2026-03-27: `Lot B3.5` est maintenant ferme. Gateway porte un catalogue BTF
  standard distinct des contrats banques, avec tables dediees,
  seed versionne `GLB/FR/DE/AT/CH`, bootstrap sur base fraiche,
  migration de rattrapage sur base existante, import/export/updateconf,
  et fallback runtime strict `specific > country > GLB`.
  Si un contrat specifique actif existe et qu'un tuple n'y est pas trouve,
  l'echange est rejete sans fallback vers le catalogue standard.
