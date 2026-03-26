# Aptitude Gateway - signatures protocolaires EBICS

## 1. Objet

Ce document evalue si Gateway dispose deja des sous-jacents techniques
necessaires pour porter les signatures EBICS au sens protocolaire.

La question n'est pas de reconstruire le workflow metier de signature, mais de
verifier si Gateway est un bon support pour:

- constater;
- verifier;
- journaliser;
- exposer;
- et, selon les cas, emettre techniquement une signature deja decidee.

## 2. Constats sur l'existant Gateway

Gateway dispose deja de briques utiles:

- stockage de credentials et de materiel cryptographique;
- administration standard des credentials/certificats;
- pipeline, historique et supervision;
- objets d'exploitation et de corrrelation;
- connecteurs de passe-plat vers le metier dans la cible d'architecture.

La librairie EBICS apporte de son cote:

- la validation protocolaire;
- les signatures et controles EBICS;
- les ordres lies a la signature distribuee;
- les return codes et etats techniques.

## 3. Limite importante

L'existant Gateway ne porte pas nativement:

- le workflow humain de collecte des signataires;
- la determination des roles de signature metier;
- la logique de validation interne avant ajout de signature;
- la politique organisationnelle du client.

Ce point est normal et doit le rester hors Gateway.

## 4. Lecture d'aptitude

Ma conclusion est:

- `GO` pour les signatures EBICS au sens protocolaire;
- `NO-GO` si l'objectif derive vers un moteur complet de workflow humain de
  signature.

Gateway est donc un bon candidat si son role reste:

- exposer les ordres a signer;
- recuperer les informations de signature;
- tracer les etats techniques;
- accepter une demande technique `HVE/HVS` deja decidee ailleurs;
- remettre l'information au SI metier.

## 5. Sous-jacents a reutiliser

Peuvent etre reutilises:

- `EbicsOperation` pour les ordres `HVD`, `HVE`, `HVT`, `HVU`, `HVZ`, `HVS`;
- la persistance dediee EBICS;
- le passe-plat metier;
- la CLI/UI d'exploitation;
- la journalisation et l'observabilite Gateway.

## 6. Sous-jacents a ajouter

Doivent etre clarifies ou ajoutes:

- representation explicite des etats techniques de signature dans
  `EbicsOperation`;
- vues REST/CLI dediees aux ordres et etats de signature;
- policy de retry/replay tres stricte pour `HVE/HVS`;
- passage propre au metier des faits techniques:
  - ordre a signer
  - signature ajoutee
  - signature refusee
  - signature annulee.

## 7. Decision d'architecture

La bonne frontiere est:

- Gateway porte la signature distribuee comme mecanisme protocolaire;
- l'application metier porte la decision humaine et l'orchestration
  organisationnelle.

## 8. Implication pour la suite

Avant un squelette technique complet, il reste utile de figer:

- la modelisation fine des etats de signature protocolaire;
- la restitution REST/CLI/UI de ces etats;
- la nature exacte des messages de passe-plat vers le metier.
