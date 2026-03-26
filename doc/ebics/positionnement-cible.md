# Positionnement cible de l'integration EBICS

## 1. Ligne directrice

L'integration EBICS dans Gateway doit rester une integration **protocolaire**,
pas une integration de workflow bancaire metier.

Gateway doit prendre en charge:

- l'exposition serveur et client EBICS;
- les controles et automatismes relevant du protocole;
- la rotation des cles lorsqu'elle releve d'ordres et d'etats EBICS;
- la recuperation automatisee des reports;
- les notifications RTN et leur transformation en pulls EBICS;
- la verification et la collecte des signatures quand cela releve du protocole;
- l'historisation, la tracabilite et l'administration technique.

Gateway ne doit pas prendre en charge:

- les decisions metier de validation;
- les circuits d'approbation metier;
- les regles metier de signature complexes;
- les arbitrages fonctionnels hors protocole;
- la logique bancaire metier specifique a une organisation.

La bonne architecture est donc:

- `Gateway = moteur protocolaire et automate EBICS`
- `Application metier = moteur de decision`

## 2. Ce qui reste dans Gateway

### 2.1 Automatisations conservees

- rotation des cles et certificats quand elle passe par les ordres EBICS:
  `PUB`, `HCA`, `HCS`, `HSA`, `SPR`, `H3K`;
- recuperation automatisee des reports:
  `FDL`, `BTD`, `HAC`, `HVD`, `HVE`, `HVT`, `HVU`, `HVZ`, selon priorisation;
- ingestion et traitement des notifications RTN;
- mapping d'un evenement RTN vers un ou plusieurs pulls;
- verification protocolaire des signatures et pieces EBICS;
- controles `AuthSignature`, anti-rejeu, segmentation, recovery, return codes;
- mise a disposition d'un historique technique fiable.

### 2.2 Frontieres fonctionnelles

Gateway peut:

- constater qu'une signature est presente, absente, invalide ou suffisante au
  sens strict du protocole;
- telecharger des reports ou des ordres en attente;
- exposer les informations techniques a une application tierce;
- declencher un traitement externe.

Gateway ne doit pas:

- decider qu'un ordre peut etre libere d'un point de vue metier;
- reconstituer un workflow d'approbation bancaire interne;
- porter les arbitrages de role, de montant ou de delegation hors protocole.

## 3. Cas particulier de l'initialisation

L'initialisation EBICS doit rester traitee avec prudence.

Position retenue:

- Gateway peut automatiser la partie purement technique (`INI`, `HIA`, `H3K`,
  `HPB`, generation de lettre);
- Gateway ne doit pas devenir le pilote metier du workflow de validation hors
  bande;
- la validation finale et les decisions associees peuvent etre faites dans
  l'application metier ou via un simple changement d'etat externe.

Autrement dit:

- on peut conserver un support technique de la lettre EBICS;
- on ne doit pas transformer Gateway en outil de pilotage humain bancaire.

## 4. Notifications et reports

Ce point entre clairement dans le bon perimetre Gateway.

Pourquoi:

- RTN est un mecanisme de signalisation technique;
- la transformation RTN -> ordre EBICS de pull est purement protocolaire;
- la recuperation des reports est une orchestration technique de collecte.

Ce que Gateway doit faire:

- recevoir la notification;
- la valider;
- la dedoublonner;
- la mapper sur un plan de pull;
- executer le ou les pulls EBICS;
- historiser le resultat;
- pousser ensuite le resultat vers l'application metier.

## 5. Signatures

Position retenue:

- Gateway gere les signatures **au sens protocolaire**;
- Gateway ne gere pas le **workflow metier de signature**.

Cela implique:

- verification des signatures presentes dans les flux EBICS;
- collecte des informations de signature;
- exposition d'un etat exploitable;
- eventuel telechargement des ordres et rapports lies aux signatures;
- passe-plat vers l'application metier pour la decision.

## 6. Mode d'integration avec l'application metier

Le couplage recommande est un couplage de type passe-plat:

- Gateway recoit, verifie, recupere, historise;
- Gateway expose ou pousse des evenements techniques;
- l'application metier decide;
- Gateway execute ensuite les ordres purement protocolaires demandes.

Mecanismes possibles:

- REST sortant/inbound;
- file de messages;
- webhook;
- publication d'evenements techniques;
- reprise de traitement par commande API.

Le positionnement retenu ne suppose pas l'ajout d'un filewatcher integre ni
d'un client lourd dans le coeur Gateway.

La priorite reste une integration asynchrone, serveur et protocolaire.

## 7. Consequence sur la decision Gateway vs from scratch

Avec ce recentrage, Gateway devient un candidat plus pertinent.

Pourquoi:

- le coeur reste MFT/protocolaire;
- les automatismes demandes sont surtout techniques;
- la logique metier reste hors de Gateway;
- on evite de faire de Gateway une application bancaire metier.

La condition reste cependant la meme:

- les lots 1 et 2 doivent confirmer que cette separation est techniquement
  propre et peu intrusive.
