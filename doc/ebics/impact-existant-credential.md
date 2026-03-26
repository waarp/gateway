# Impact sur l'existant `Credential`

## 1. Objet

Ce document precise l'impact des besoins EBICS sur l'existant `Credential`.

L'objectif est de fixer clairement:

- ce qui ne change pas;
- ce qui doit etre enrichi;
- ce qu'il faut explicitement eviter.

## 2. Ce qui ne change pas

`Credential` conserve son role actuel:

- conteneur generique de materiel d'authentification;
- rattachement a un proprietaire Gateway existant;
- support des certificats et cles deja administres;
- administration REST/CLI actuelle des credentials et certificats.

Il reste donc un objet transverse a tous les protocoles.

## 3. Ce qui doit etre enrichi

Sans changer sa nature, l'existant devra permettre:

- de relier un credential a un `EbicsKeyLifecycle`;
- d'exposer en lecture un contexte EBICS derive:
  - `lifecycleId`
  - `lifecycleRole`
  - `ebicsUsage`;
- de proteger les operations destructives sur un credential utilise par un
  lifecycle actif;
- d'afficher, en UI/CLI/REST, qu'un credential participe a une rotation EBICS.

## 4. Ce qu'il faut explicitement eviter

Il ne faut pas:

- ajouter dans `Credential` des champs de cycle de vie EBICS comme:
  - `is_current`
  - `is_next`
  - `activated_at`
  - `retired_at`
  - `rotation_status`;
- faire de la rotation EBICS par simple mise a jour en place du credential;
- transformer `Credential` en objet specifique a EBICS.

## 5. Impacts techniques attendus

### 5.1 REST

Impacts acceptables:

- enrichissement des DTO de lecture;
- liens croises vers `EbicsKeyLifecycle`;
- refus de suppression si lifecycle actif.

### 5.2 CLI

Impacts acceptables:

- affichage d'un role EBICS derive;
- message explicite quand un credential ne peut pas etre supprime.

### 5.3 UI

Impacts acceptables:

- badge ou section de contexte EBICS;
- lien vers le lifecycle en cours;
- indication `current / next / historical`.

## 6. Niveau d'impact estime

Ma conclusion est:

- impact faible a modere sur le modele;
- impact modere sur l'administration;
- impact fort a eviter si on glisse vers une semantique EBICS directement dans
  `Credential`.

## 7. Decision recommandee

La bonne ligne est:

- `Credential` reste stable;
- `EbicsKeyLifecycle` porte la semantique EBICS;
- l'administration `Credential` est seulement enrichie par correlation.
