# Aptitude Gateway - rotation des cles EBICS

## 1. Objet

Ce document evalue si Gateway dispose deja des sous-jacents techniques
necessaires pour porter correctement la rotation des cles EBICS.

Le but est de distinguer:

- ce qui existe deja dans Gateway;
- ce qui peut etre reutilise tel quel;
- ce qu'il faut encore construire pour un workflow EBICS propre.

## 2. Constats sur l'existant Gateway

L'existant apporte deja des briques utiles:

- stockage durable de credentials rattaches a `LocalAgent`, `RemoteAgent`,
  `LocalAccount`, `RemoteAccount`;
- support de certificats et cles TLS via `Credential`, avec chiffrement de la
  cle privee en base;
- administration REST/CLI des certificats et credentials;
- gestion d'autorites et de certificats de confiance;
- persistance de cles cryptographiques generiques via `CryptoKey`.

Concretement, cela signifie que Gateway sait deja:

- stocker du materiel cryptographique;
- le rattacher a des objets d'administration existants;
- l'exposer et le mettre a jour via la filiere admin standard;
- journaliser et securiser ce stockage a un niveau satisfaisant pour une base
  de travail.

## 3. Ce que l'existant ne couvre pas encore

L'existant ne couvre pas, a lui seul:

- la notion de paire `courante / future` propre au cycle de rotation EBICS;
- la coexistence gouvernee de plusieurs materiels EBICS pour un meme abonne;
- la liaison explicite entre un credential et un ordre EBICS de rotation
  (`PUB`, `HCA`, `HCS`, `HSA`, `SPR`, `H3K`);
- les etats de workflow:
  - prepare
  - sent
  - waiting-confirmation
  - activated
  - retired;
- les evidences operateur associees a la rotation;
- le rejet, l'annulation ou le retrait gouverne d'un materiel futur.

Autrement dit:

- Gateway sait stocker la matiere;
- Gateway ne sait pas encore porter nativement le cycle de vie EBICS de cette
  matiere.

## 4. Lecture d'aptitude

Ma conclusion est:

- `GO sous reserve` pour la rotation des cles dans Gateway.

Pourquoi:

- le socle de stockage et d'administration existe deja;
- il n'est pas necessaire de repartir de zero pour les certificats et cles;
- mais il faut ajouter un vrai workflow durable, pas juste mettre a jour un
  `Credential`.

## 5. Sous-jacents a reutiliser

Peuvent etre reutilises directement:

- `Credential` pour stocker certificats et cles;
- `CryptoKey` si un besoin de cle non-certificat apparait cote integration;
- routes REST/CLI d'admin de credentials comme base ergonomique;
- permissions admin existantes;
- audit et historisation Gateway existants.

## 6. Sous-jacents a completer

Doivent etre ajoutes ou precises:

- `EbicsKeyLifecycle`;
- rattachement entre credentials et workflow de rotation;
- regles d'activation logique de la nouvelle cle;
- traces d'evidence et confirmations;
- validation de compatibilite banque/profil avant emission;
- politiques de replay strictes pour les ordres sensibles.

## 7. Decision d'architecture

La bonne approche n'est pas:

- de faire de la rotation EBICS par simple `PATCH` de credential.

La bonne approche est:

- stocker le materiel dans les objets existants;
- piloter la rotation dans un workflow EBICS dedie;
- exposer a l'operateur un etat explicite du cycle de vie.

## 8. Implication pour la suite

Avant un squelette technique complet, il est pertinent de figer encore:

- le mapping exact `Credential <-> EbicsKeyLifecycle`;
- les regles d'activation et de coexistence;
- les preuves/evidences minimales;
- la strategie de rollback ou de retrait.
