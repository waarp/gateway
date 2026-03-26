# Impacts CLI et UI de l'integration EBICS

## 1. Objet

Ce document fige les impacts attendus de l'integration EBICS sur:

- la CLI;
- l'UI;
- les ecrans et commandes existants de Gateway.

Son objectif est de servir de garde-fou pendant le developpement.

## 2. Principe directeur

La regle de lecture doit rester simple:

- `transfer` = flux fichier;
- `ebics operation` = traitement protocolaire EBICS;
- une correlation visible doit exister entre les deux quand ils coexistent.

Il ne faut pas:

- afficher un ordre `HPD`, `INI` ou `HAC` comme un faux transfert;
- surcharger les commandes `transfer` avec des concepts EBICS non fichier;
- obliger l'exploitant a deviner si un objet releve du protocole ou du
  transport de fichier.

## 3. Impacts CLI

## 3.1 Nouvelles familles de commandes

Familles minimales a ajouter:

- `waarp-gateway ebics operation ...`
- `waarp-gateway ebics contract-view ...`
- `waarp-gateway ebics initialization ...`
- `waarp-gateway ebics key-lifecycle ...`
- `waarp-gateway ebics rtn ...`

La famille la plus structurante est `ebics operation`.

## 3.2 Commandes a ne pas detourner

Les commandes existantes suivantes doivent rester orientees transfert:

- `waarp-gateway transfer ...`
- `waarp-gateway history ...`

Ces commandes peuvent afficher un lien vers une `EbicsOperation`, mais ne
doivent pas devenir le point d'entree principal pour administrer:

- `HPD`, `HKD`, `HTD`, `HAA`;
- `INI`, `HIA`, `H3K`;
- RTN;
- signatures protocolaires non projetees en fichier.

## 3.3 Filtres nouveaux ou adaptes

Filtres CLI a prevoir pour `ebics operation list`:

- `--status`
- `--operation-type`
- `--order-type`
- `--partner-id`
- `--user-id`
- `--transaction-id`
- `--request-id`
- `--correlation-id`
- `--transfer-id`
- `--date`

## 3.4 Affichage CLI

Une `EbicsOperation` doit afficher au minimum:

- son ID;
- son ordre EBICS;
- son type;
- sa direction;
- son statut;
- son return code;
- ses identifiants EBICS;
- son `Transfer` associe si present.

L'affichage doit explicitement differencier:

- succes protocolaire;
- attente de confirmation externe;
- echec technique;
- existence ou non d'un `Transfer` associe.

## 4. Impacts UI

## 4.1 Nouvelles vues a ajouter

Vues minimales a ajouter:

- `Operations EBICS`
- `Detail operation EBICS`
- `Vue contrat technique`
- `Workflow d'initialisation EBICS`
- `Cycles de vie des cles`
- `Supervision RTN`

## 4.2 Vues existantes a ajuster

Vues existantes a adapter:

- supervision des transferts;
- vues client/partner/account quand protocole = `ebics`;
- ecrans de creation/modification de `Client`, `RemoteAgent`, `LocalAgent`;
- ecrans de credentials si des usages EBICS specifiques apparaissent.

## 4.3 Liens croises a rendre visibles

L'UI doit rendre visible la correlation:

- `EbicsOperation` -> `Transfer`
- `EbicsOperation` -> `contract view`
- `EbicsOperation` -> `initialization workflow`
- `EbicsOperation` -> `key lifecycle`
- `EbicsOperation` -> `RTN event`

L'utilisateur ne doit pas avoir a croiser plusieurs ecrans manuellement pour
comprendre un enchainement fonctionnel.

## 5. Impacts sur la navigation

La navigation produit doit expliciter deux axes:

- l'axe `transport / transfert`;
- l'axe `protocole / operation EBICS`.

Recommendation:

- entree principale `EBICS` dans la navigation;
- sous-entrees:
  - `Operations`
  - `Contrat technique`
  - `Initialisation`
  - `Cles`
  - `RTN`

Les transferts EBICS restent consultables depuis la vue transferts generale,
mais ne doivent pas absorber toute l'admin EBICS.

## 6. Risques UX a eviter

Risques principaux:

- faux positifs: un exploitant pense qu'un ordre protocolaire est un transfert;
- faux negatifs: un exploitant ne voit pas une operation EBICS car elle n'est
  pas dans la vue transferts;
- confusion entre `retry transfer` et `retry ebics operation`;
- confusion entre reprise fichier et recovery/segmentation EBICS;
- masquage des etats `WAITING_EXTERNAL_CONFIRMATION` ou `WAITING_BANK`.

## 7. Regles de developpement

Pendant le developpement, appliquer les regles suivantes:

- ne jamais ajouter un ordre EBICS non payload dans la CLI `transfer`;
- ne jamais ajouter un ecran EBICS non payload dans la seule supervision des
  transferts;
- afficher les identifiants EBICS sans les confondre avec les identifiants
  Gateway;
- si une action existe sur `Transfer` et sur `EbicsOperation`, documenter et
  afficher clairement la difference.

## 8. Aides memoire pour les devs

Question a se poser avant toute commande ou tout ecran:

1. est-ce un transfert de fichier ou une operation protocolaire ?
2. l'exploitant doit-il agir sur `Transfer` ou sur `EbicsOperation` ?
3. faut-il afficher une correlation avec un autre objet ?
4. l'action demandee est-elle un retry Gateway ou un replay/recovery EBICS ?

## 9. Decision recommandee

L'impact CLI/UI est reel, mais il est justifie et preferable a une modelisation
fausse.

La bonne ligne est:

- assumer une famille `ebics` visible;
- garder `transfer` centree sur le fichier;
- ajouter des correlations claires;
- ne pas sacrifier la lisibilite d'exploitation pour economiser quelques ecrans
  ou commandes.
