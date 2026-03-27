# Plan de consolidation backend EBICS

## 1. Objectif

Ce document cadre la suite des travaux backend EBICS avec une ligne simple:

- ne plus laisser de `stub` bloquant;
- ne plus laisser de fonctionnalite "posee mais partielle" sur le perimetre EBICS;
- atteindre un niveau d'exploitation/production suffisant avant d'ouvrir le chantier frontend.

Le frontend n'entre en phase active qu'une fois ce plan considere termine.

## 2. Definition de sortie

Le backend EBICS est considere "pret frontend" quand les conditions suivantes sont simultanement vraies:

- aucun flux EBICS critique ne s'arrete sur `ErrNotImplemented`;
- toutes les familles REST/CLI EBICS exposees sont branchees sur une logique runtime reelle;
- le serveur et le client EBICS utilisent `lib-ebics` sans `replace` local;
- les objets durables critiques (`operations`, `transactions`, `segments`, `key lifecycles`, `initializations`, `RTN`) sont exploitables de bout en bout;
- l'import/export/updateconf couvre tous les objets administres EBICS;
- les journaux, erreurs et statuts sont exploitables par un operateur;
- les comportements de recovery/replay/retry sont explicites, sans zone grise;
- aucun point critique de persistance ne repose sur un comportement implicite ou fragile.

## 3. Reste a consolider

## 3.1 Client EBICS reel

Constat actuel:

- `pkg/protocols/modules/ebics/client.go` demarre un profil `lib-ebics`;
- le chemin nominal payload client `BTU/BTD` est branche sur `lib-ebics`,
  avec creation `EbicsOperation` / `EbicsTransaction`, contrat actif,
  transport TLS, recovery et correlation `Transfer`.

Objectif:

- brancher une vraie execution cliente EBICS pour `BTU/BTD`, avec `FUL/FDL` limites a des alias de compatibilite normalises, puis couvrir ordres d'administration, reporting, initialisation et key management;
- integrer la creation/mise a jour de `EbicsOperation`, `EbicsTransaction` et `Transfer` cote client;
- fermer completement le stub `InitTransfer`.

Sortie attendue:

- plus aucun `ErrNotImplemented` sur le chemin nominal client;
- pipeline client Gateway reliee a `lib-ebics`.

## 3.2 Couverture fonctionnelle EBICS complete cote backend

Constat actuel:

- le socle data/runtime est pose;
- une partie des familles REST/CLI reste surtout centree sur exposition/lecture/action simple.

Objectif:

- couvrir de bout en bout les familles suivantes:
  - payloads,
  - operations,
  - transactions,
  - contract views,
  - payload profiles,
  - initializations,
  - key lifecycles,
  - RTN.

Sortie attendue:

- aucune ressource admin exposee sans logique backend suffisante;
- aucun ecran frontend ne devra compenser une faiblesse backend structurelle.

## 3.3 Import / export / updateconf complet

Constat actuel:

- le socle `ProtoConfig` est couvert;
- le suivi signale encore des items ouverts sur `pkg/backup`.

Objectif:

- prendre en charge tous les objets EBICS administres dans `export/import/updateconf`;
- garantir un round-trip JSON/YAML fiable.

Sortie attendue:

- un exploitant peut recreer une configuration EBICS complete sans script ad hoc.

## 3.4 Serveur EBICS reel et durcissement serveur

Constat actuel:

- le serveur Gateway est deja branche sur `lib-ebics`;
- les handlers payload `FUL` / `FDL` / `BTU` / `BTD` sont exposes et relies au routeur Gateway;
- ce socle reel n'a toutefois pas encore fait l'objet d'un lot de consolidation explicite
  comparable a celui du client.

Objectif:

- relire le chemin nominal serveur `BTU/BTD` avec une exigence d'exploitation/production;
- verifier la couverture normative des ordres serveur non payload reels exposes par `lib-ebics`;
- durcir la segmentation, le recovery, la reprise, les statuts et les erreurs cote serveur;
- verifier que les retours et journaux serveur sont exploitables sans ambiguite.

Sortie attendue:

- le serveur EBICS n'est plus seulement "branche", il est considere consolide;
- aucun angle mort serveur critique ne subsiste avant frontend.

## 3.5 Production / exploitation

Constat actuel:

- le socle de modele est solide;
- il reste a verrouiller les exigences d'exploitation.

Objectif:

- completer:
  - journalisation operationnelle,
  - messages d'erreur contextualises,
  - correlation entre objets,
  - purge/retention,
  - reprise/recovery,
  - protection contre les incoherences d'etat,
  - discipline multi-SGBD/XORM.

Sortie attendue:

- l'operateur comprend ce qui se passe sans debugger le code;
- les interventions manuelles sont bornees et tracables.

## 4. Lots de consolidation proposes

## Lot B1 - Execution cliente reelle

Perimetre:

- `pkg/protocols/modules/ebics/client.go`
- runners/client runtime lies a `lib-ebics`
- creation des `EbicsOperation` / `EbicsTransaction` / `Transfer` cote client.

Bloquant frontend:

- oui.

## Lot B2 - Orders backend complets

Perimetre:

- ordres payload, reporting, initialisation, key management, signatures protocolaires;
- fermeture des "actions presentes mais non reliees".

Bloquant frontend:

- oui.

## Lot B3 - Import / export / updateconf complet

Perimetre:

- `pkg/backup/*`
- couverture de tous les objets EBICS administres.

Bloquant frontend:

- oui pour une cible production;
- non pour une simple demo locale.

Statut:

- 2026-03-27: termine
- couverture posee pour `hosts`, `subscribers`, `bank keys`,
  `payload profiles` et `RTN providers`
- round-trip verifie par fixtures `JSON/YAML`, test dedie `pkg/backup`
  et scenario `updateconf`
- migration `0.16.0` ajoutee pour garantir la presence des tables EBICS
  dans les bases migrees et dans les environnements de test

## Lot B4 - Durcissement exploitation + consolidation serveur

Perimetre:

- execution serveur reelle EBICS;
- couverture normative et operationnelle cote serveur;
- messages d'erreur,
- statuts operateur,
- correlation,
- purge,
- retention,
- supervision,
- hygiene des actions CLI/REST.

Bloquant frontend:

- oui.

## Lot B5 - Verification backend de sortie

Perimetre:

- passe finale "zero stub bloquant";
- verification de coherence backend globale;
- pre-requis de lancement frontend.

Bloquant frontend:

- oui, c'est la porte de sortie.

## 5. Ordre recommande

Ordre de traitement:

1. `Lot B1 - Execution cliente reelle`
2. `Lot B2 - Orders backend complets`
3. `Lot B3 - Import / export / updateconf complet`
4. `Lot B4 - Durcissement exploitation + consolidation serveur`
5. `Lot B5 - Verification backend de sortie`

## 6. Regle de conduite

Pour cette phase de consolidation:

- on evite toute nouvelle surface frontend;
- on prefere fermer un flux complet plutot qu'ouvrir un nouveau morceau partiel;
- tout nouvel endpoint/commande doit etre branche sur une logique reelle;
- toute divergence par rapport aux specs doit etre tracee immediatement dans les documents de suivi.
