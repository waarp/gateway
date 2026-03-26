# Cadrage concret Phase D EBICS

## 1. Objet

Ce document detaille la `Phase D` du squelette technique EBICS.

La `Phase D` couvre les workflows sensibles:

- `EbicsKeyLifecycle`;
- `EbicsInitializationWorkflow`;
- `signatureState`;
- les runners techniques associes.

Cette phase est la plus sensible conceptuellement, car elle touche:

- la cryptographie;
- la rupture d'automatisme;
- les actions operateur;
- les etats bancaires non strictement derives du pipeline fichier.

## 2. Position de travail

La `Phase D` doit rester strictement dans la frontiere deja fixee:

- Gateway porte le protocole, les etats techniques, les evidences, les
  confirmations et l'exploitation;
- Gateway ne porte pas la decision metier ni le workflow humain bancaire du
  client;
- `Credential` reste le conteneur de materiel cryptographique;
- les workflows EBICS portent la gouvernance de transition.

## 3. Perimetre de la Phase D

### 3.1 Inclus

- model `EbicsKeyLifecycle`;
- model `EbicsInitializationWorkflow`;
- representation explicite du `signatureState`;
- transitions d'etats et actions operateur autorisees;
- runners techniques:
  - `key_lifecycle_runner`
  - `initialization_runner`
  - `signature_state`;
- DTO REST/CLI/UI minimaux pour exploiter ces workflows;
- regles de protection sur `Credential` references.

### 3.2 Exclus

- moteur complet de workflow humain de signature;
- moteur documentaire avance pour la lettre EBICS;
- RTN;
- enrichissements metier clients;
- automatisation bancaire non protocolaire.

## 4. Exigences `production grade`

### 4.1 Gouvernance des etats

- chaque workflow sensible doit avoir un `status` explicite;
- les transitions interdites doivent etre connues et bloquees;
- les confirmations operateur doivent etre tracees;
- les dates de passage d'etat doivent etre conservees.

### 4.2 Evidence et audit

- toute transition manuelle importante doit pouvoir porter:
  - `operator`
  - `reason`
  - `evidence`;
- les evidences doivent rester portables multi-SGBD;
- les objets sensibles ne doivent jamais reposer sur un simple commentaire libre
  non structure.

### 4.3 Protection de l'existant

- un `Credential` implique dans un lifecycle actif ne doit pas etre supprime
  ou modifie sans regle explicite;
- l'initialisation EBICS ne doit jamais etre modelee comme un enchainement
  purement automatique irreversible;
- les ordres de signature sensibles ne doivent jamais heriter d'un retry
  aveugle.

### 4.4 Multi-SGBD

- pas de dependance a des types SQL proprietaires;
- stockage d'evidence et de metadonnees sous forme portable;
- contraintes de coherence posees d'abord au niveau model/runtime, pas
  uniquement au niveau DDL.

## 5. Fichiers a cadrer concretement

## 5.1 `pkg/model/table_names.go`

Ajouts cibles:

- `TableEbicsKeyLifecycles = "ebics_key_lifecycles"`
- `TableEbicsInitializationWorkflows = "ebics_initialization_workflows"`

## 5.2 `pkg/model/display_names.go`

Ajouts cibles:

- `NameEbicsKeyLifecycle = "ebics key lifecycle"`
- `NameEbicsInitializationWorkflow = "ebics initialization workflow"`

## 5.3 `pkg/model/ebics_key_lifecycle.go`

Responsabilite:

- porter le workflow durable de rotation de cles EBICS;
- relier `current_credential_id` et `next_credential_id`;
- tracer l'ordre declencheur et les preuves de transition;
- gouverner l'activation et le retrait logique.

Invariants:

- `EbicsSubscriberID` obligatoire;
- `KeyUsage` obligatoire;
- `RotationType` obligatoire;
- `Status` obligatoire;
- `CurrentCredentialID` obligatoire;
- `NextCredentialID` optionnel selon etat;
- au plus un lifecycle actif par `(subscriber, key_usage)`.

## 5.4 `pkg/model/ebics_initialization_workflow.go`

Responsabilite:

- porter le workflow d'initialisation EBICS avec rupture d'automatisme;
- tracer les etapes `INI`, `HIA`, `H3K`, lettre EBICS, retour banque,
  activation;
- fournir un pivot propre pour REST/CLI/UI.

Invariants:

- `EbicsSubscriberID` obligatoire;
- `Status` obligatoire;
- `CurrentStep` obligatoire;
- les references d'operations EBICS associees doivent etre conservables;
- la confirmation externe ne doit jamais etre deduite sans evidence.

## 5.5 `pkg/protocols/modules/ebics/runtime/key_lifecycle_runner.go`

Responsabilite:

- appliquer les transitions techniques du lifecycle;
- verifier la coherence des ordres emis;
- refuser les transitions interdites;
- produire des mises a jour d'etat et d'evidence propres.

## 5.6 `pkg/protocols/modules/ebics/runtime/initialization_runner.go`

Responsabilite:

- piloter l'enchainement technique des ordres d'initialisation;
- s'arreter explicitement lors des ruptures d'automatisme;
- reprendre uniquement sur action ou evidence autorisee;
- preparer les transitions `WAITING_EXTERNAL_CONFIRMATION` /
  `WAITING_BANK_CONFIRMATION` / `ACTIVATED`.

## 5.7 `pkg/protocols/modules/ebics/runtime/signature_state.go`

Responsabilite:

- centraliser les etats de signature protocolaire;
- deriver le `signatureState` a partir des ordres et retours EBICS;
- preparer la restitution REST/CLI/UI et le passe-plat metier technique.

## 5.8 `pkg/admin/rest/api/ebics_key_lifecycles.go`

Responsabilite:

- DTO `In/Out` du lifecycle;
- actions `confirm`, `cancel`, `activate`, `retire` selon les cas;
- exposition lisible des credentials courant/futur.

## 5.9 `pkg/admin/rest/api/ebics_initializations.go`

Responsabilite:

- DTO `In/Out` du workflow d'initialisation;
- actions `send-ini`, `send-hia`, `send-h3k`, `confirm-letter`,
  `confirm-bank-activation`, `cancel`.

## 5.10 `pkg/admin/rest/api/ebics_operations.go`

Responsabilite additionnelle en `Phase D`:

- exposer `signatureState` quand applicable;
- exposer les blocs `technical/business` sans ambiguite;
- permettre une lecture claire des ordres sensibles.

## 6. Axes de modelisation prioritaires

## 6.1 Rotation des cles

La ligne fixe est:

- `Credential` stocke;
- `EbicsKeyLifecycle` gouverne;
- `EbicsOperation` trace l'ordre;
- `Gateway` protege les operations destructives sur les credentials references.

## 6.2 Initialisation

La ligne fixe est:

- le workflow d'initialisation est un objet durable;
- la lettre EBICS introduit un arret operateur reel;
- la confirmation banque est un fait externe a tracer, jamais une deduction;
- l'activation finale doit etre visible comme transition explicite.

## 6.3 Signatures

La ligne fixe est:

- `EbicsOperation.status` reste l'etat principal;
- `signatureState` est un sous-etat technique dedie;
- Gateway expose les faits techniques;
- le metier decide de l'action humaine.

## 7. Ordre de pose

Ordre recommande:

1. `table_names.go` et `display_names.go`
2. `ebics_key_lifecycle.go`
3. `ebics_initialization_workflow.go`
4. `runtime/signature_state.go`
5. `runtime/key_lifecycle_runner.go`
6. `runtime/initialization_runner.go`
7. DTO `api`

## 8. Definition de done de la Phase D

La `Phase D` est terminee si:

- les workflows sensibles existent comme objets explicites;
- les transitions principales sont connues et bornees;
- les credentials references sont proteges correctement;
- l'initialisation ne peut pas franchir ses ruptures d'automatisme sans action
  explicite;
- `signatureState` est stable, lisible et distinct du `status`;
- les runners techniques savent refuser un etat ou une action incoherente.

## 9. Point de vigilance principal

Le risque central de la `Phase D` est double:

- soit trop simplifier et perdre la rigueur protocolaire;
- soit trop en faire et glisser vers du workflow metier bancaire.

La bonne ligne est:

- workflow technique complet;
- gouvernance d'exploitation complete;
- mais aucune prise de decision metier a la place du client.
