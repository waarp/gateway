# Cadrage concret Phase B EBICS

## 1. Objet

Ce document detaille la `Phase B` du squelette technique EBICS.

La `Phase B` couvre:

- `EbicsPayloadProfile`;
- `EbicsContractView`;
- `EbicsContractViewItem`;
- la resolution des parametres payload;
- la validation contractuelle.

L'objectif est de rendre le systeme capable de porter une configuration
payload sure et gouvernee, avant meme d'introduire toutes les executions
operationnelles EBICS.

## 2. Position de travail

La `Phase B` doit rester dans la logique:

- protocole et securisation technique cote Gateway;
- pas de logique metier bancaire embarquee;
- objets structurants deja exploitables pour REST, CLI, UI et `updateconf`;
- compatibilite stricte multi-SGBD via l'ORM existant.

## 3. Perimetre de la Phase B

### 3.1 Inclus

- model `EbicsPayloadProfile`;
- model `EbicsContractView`;
- model `EbicsContractViewItem`;
- constantes de statuts et de types utiles a ces objets;
- service de resolution `explicite > profile > defaults`;
- service de validation contre la vue contractuelle active;
- hooks minimaux pour admin REST/CLI ulterieure;
- prise en compte `updateconf` / import-export.

### 3.2 Exclus

- `EbicsOperation`;
- soumission effective `BTU/BTD/FUL/FDL`;
- projection vers `Transfer`;
- gestion transactionnelle `transaction/segment`;
- `RTN`;
- rotation de cles et initialisation;
- orchestration de signature.

## 4. Exigences `production grade`

### 4.1 Gouvernance

- un profil payload est un objet gouverne, pas un raccourci fragile de CLI;
- la vue contractuelle est une photographie technique constatee, pas une
  politique metier;
- la validation contractuelle doit etre explicable et auditable.

### 4.2 Multi-SGBD

- aucun type SQL specifique moteur;
- pas de dependance a des colonnes JSON natives;
- tout ce qui doit etre exploitable finement en UI/API doit etre structure en
  colonnes et tables filles plutot qu'en blob.

### 4.3 Exploitation

- il doit etre possible de comprendre pourquoi un profil est valide, invalide
  ou desactive;
- la relation entre un profil et une ligne de contrat doit etre observable;
- la resolution d'une demande payload doit produire un resultat journalisable;
- les erreurs de validation doivent etre deterministes et orientables.

## 5. Fichiers a cadrer concretement

## 5.1 `pkg/model/table_names.go`

Ajouts cibles:

- `TableEbicsContractViews = "ebics_contract_views"`
- `TableEbicsContractViewItems = "ebics_contract_view_items"`
- `TableEbicsPayloadProfiles = "ebics_payload_profiles"`

## 5.2 `pkg/model/display_names.go`

Ajouts cibles:

- `NameEbicsContractView = "ebics contract view"`
- `NameEbicsContractViewItem = "ebics contract view item"`
- `NameEbicsPayloadProfile = "ebics payload profile"`

## 5.3 `pkg/model/ebics_contract_view.go`

Responsabilite:

- representer l'entete d'un snapshot de contrat technique EBICS;
- servir de racine a une collection de lignes `EbicsContractViewItem`;
- tracer l'origine de la collecte (`HPD`, `HKD`, `HTD`, `HAA`).

Invariants:

- reference a un `EbicsHost` obligatoire;
- `EbicsSubscriberID` optionnel selon le scope du snapshot;
- `SourceOrderType` obligatoire;
- `Status` borne;
- `VersionTag` stable et lisible en exploitation.

## 5.4 `pkg/model/ebics_contract_view_item.go`

Responsabilite:

- porter les lignes structurantes exploitables du contrat;
- permettre la validation fine sans parser un blob JSON cote UI ou REST;
- representer les capacites payload, admin et contraintes associees.

Invariants:

- rattachement a un `ContractViewID` obligatoire;
- `ItemType` borne;
- `ItemKey` obligatoire;
- coherence minimale entre `ItemType` et les colonnes renseignees;
- `IsEnabled` explicite.

Point important:

- `Payload` libre ne doit servir qu'aux details secondaires non eleves au rang
  de colonne structuree.

## 5.5 `pkg/model/ebics_payload_profile.go`

Responsabilite:

- porter un profil reusable d'emission ou de collecte payload;
- rester distinct de `Rule`;
- rester distinct de la vue contractuelle;
- pouvoir etre valide a froid contre le contrat connu.

Invariants:

- unicite `(owner, name)`;
- `OrderType` borne a `BTU/BTD/FUL/FDL`;
- `Direction` borne;
- cohérence entre `OrderType` et `Direction`;
- `DefaultRuleID` optionnel mais valide s'il est renseigne;
- `StrictContractCheck` explicite;
- `IsEnabled` explicite.

## 5.6 `pkg/protocols/modules/ebics/runtime/payload_resolution.go`

Responsabilite:

- resoudre les champs d'une demande payload a partir:
  - des champs explicites;
  - d'un profil;
  - des defaults du `ProtoConfig`.

Resultat attendu:

- un objet resolu complet et journalisable;
- aucune emission reseau;
- aucune creation d'`EbicsOperation` en `Phase B`.

## 5.7 `pkg/protocols/modules/ebics/runtime/contract_validation.go`

Responsabilite:

- valider un payload resolu contre la vue contractuelle active;
- retrouver la ou les lignes contractuelles compatibles;
- produire un resultat exploitable par REST, CLI, logs et audit.

Resultat attendu:

- `MATCHED`, `MISMATCH`, `NO_ACTIVE_CONTRACT`, `NO_MATCHING_ITEM` ou equivalent;
- liste des `contract_item_id` apparies quand c'est pertinent;
- message de validation stable.

## 5.8 `pkg/admin/rest/api/ebics_payload_profiles.go`

Responsabilite:

- porter les DTO officiels `In/Out`;
- ne pas y cacher de logique de resolution;
- rester stable pour `JSON/YAML`.

## 5.9 `pkg/admin/rest/api/ebics_contract_views.go`

Responsabilite:

- porter les DTO de lecture du contrat;
- exposer clairement les items structurels;
- ne pas rebasculer vers une restitution en blob opaque.

## 5.10 `pkg/tasks/updateconf.go` et `pkg/backup/*`

Responsabilite:

- export/import des profils payload;
- export/import des vues contractuelles si retenu;
- round-trip propre des defaults et policies.

## 6. Migrations et ordre de pose

Ordre recommande:

1. `table_names.go` et `display_names.go`
2. `ebics_contract_view.go`
3. `ebics_contract_view_item.go`
4. `ebics_payload_profile.go`
5. `runtime/payload_resolution.go`
6. `runtime/contract_validation.go`
7. DTO `api`
8. `updateconf` / import-export

## 7. Definition de done de la Phase B

La `Phase B` est terminee si:

- les trois models existent avec leurs invariants de base;
- les statuts, types et policies sont fixes;
- un profil peut etre valide a froid contre une vue contractuelle active;
- le resultat de resolution payload est explicite et journalisable;
- la structure contractuelle reste exploitable sans blob JSON dominant;
- les objets sont serialisables proprement via REST et `updateconf`;
- aucune logique `Transfer` ou `EbicsOperation` n'a ete introduite prematurement.

## 8. Point de vigilance principal

Le risque principal de la `Phase B` serait de melanger:

- contrat technique observe;
- configuration reusable Gateway;
- et demande ponctuelle d'execution.

La bonne ligne est donc:

- `ContractView` constate;
- `PayloadProfile` prepare;
- `ResolvedPayloadRequest` calcule;
- et seulement plus tard, en `Phase C`, `EbicsOperation` execute.
