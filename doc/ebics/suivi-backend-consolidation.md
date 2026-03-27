# Suivi de consolidation backend EBICS

## 1. Usage

Cette checklist sert a piloter la fermeture du backend EBICS avant le frontend.

Regles:

- cocher `[x]` quand l'item est termine, relu et valide techniquement;
- laisser `[ ]` si l'item n'est pas encore ferme;
- utiliser `[-]` pour un report explicite;
- documenter toute divergence structurelle.

## 2. Gate de sortie backend

- [ ] Plus aucun `ErrNotImplemented` sur le chemin nominal EBICS
- [ ] Plus aucun endpoint/commande EBICS expose sans logique runtime suffisante
- [ ] Plus aucun `replace` local vers `lib-ebics`
- [ ] Import/export/updateconf complets pour les objets EBICS administres
- [ ] Politique d'exploitation documentee et relue
- [ ] Backend declare "pret frontend"

## 3. Lot B1 - Execution cliente reelle

- [x] Remplacer le stub `InitTransfer` dans `pkg/protocols/modules/ebics/client.go`
- [x] Definir le mapping `Transfer -> ordre client EBICS`
- [x] Creer la creation d'`EbicsOperation` cote client
- [x] Creer la creation d'`EbicsTransaction` cote client quand necessaire
- [x] Brancher `BTU/BTD` cote client
- [x] Confirmer que `FUL/FDL` restent des alias de compatibilite normalises vers `BTU/BTD` en cible `EBICS 3.0.2`
- [ ] Brancher reporting / ordres admin cote client
- [ ] Brancher initialisation / key management cote client
- [x] Garantir la correlation `operation / transaction / transfer`
- [x] Verifier l'exploitation des return codes `technical/business`

## 4. Lot B2 - Couverture backend complete

- [ ] Revoir chaque famille REST EBICS et confirmer l'absence de logique partielle
- [ ] Revoir chaque famille CLI EBICS et confirmer l'absence de logique partielle
- [ ] Verifier que `payloads` est bien exploitable de bout en bout
- [ ] Verifier que `operations` est bien exploitable de bout en bout
- [ ] Verifier que `transactions` est bien exploitable de bout en bout
- [ ] Verifier que `contract views` est bien exploitable de bout en bout
- [ ] Verifier que `payload profiles` est bien exploitable de bout en bout
- [ ] Verifier que `initializations` est bien exploitable de bout en bout
- [ ] Verifier que `key lifecycles` est bien exploitable de bout en bout
- [ ] Verifier que `RTN` est bien exploitable de bout en bout

## 5. Lot B3 - Import / export / updateconf

- [ ] Etendre `pkg/backup/export.go`
- [ ] Etendre `pkg/backup/import.go`
- [ ] Ajouter les helpers `*_export.go`
- [ ] Ajouter les helpers `*_import.go`
- [ ] Cadrer les jeux JSON/YAML de reference
- [ ] Verifier le round-trip complet des `ProtoConfig`
- [ ] Verifier le round-trip complet des objets EBICS administres

## 6. Lot B4 - Durcissement exploitation

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
  Reste a fermer dans `B1`: reporting/admin client et key management de rotation
  hors simple initialisation.
