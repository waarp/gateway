# Frontiere protocole / metier pour EBICS dans Gateway

## 1. Objet

Ce document verrouille la frontiere entre:

- ce que Gateway doit porter pour EBICS;
- ce qui doit rester dans l'application metier ou dans l'organisation cliente.

Il sert de regle de conception pour les lots 1 et 2, puis pour la suite de
l'implementation.

## 2. Principe directeur

La frontiere retenue est la suivante:

- `Gateway = moteur protocolaire EBICS + automate technique + point
  d'integration fiable`
- `Application metier = moteur de decision, d'orchestration humaine et de
  traitement fonctionnel`

Corollaire:

- tout besoin EBICS n'est pas automatiquement du ressort de Gateway;
- Gateway n'embarque que ce qui est necessaire pour executer, tracer, securiser
  et exposer le protocole.

## 3. Ce qui est dans le perimetre Gateway

### 3.1 Transport, securite et validite protocolaire

Gateway doit porter:

- terminaison HTTP(S) / TLS / mTLS;
- validation H005, XSD et structure des messages;
- verification `AuthSignature`;
- verification des signatures EBICS au sens protocolaire;
- protection anti-rejeu `Nonce` / `Timestamp`;
- gestion des `ReturnCode`, `ReportText`, recovery et segmentation.

### 3.2 Identites et etats protocolaires

Gateway doit porter:

- `HostID`;
- `PartnerID` / `UserID`;
- etats techniques des abonnes EBICS;
- cles abonne et cles banque utiles au protocole;
- bank parameters et metadonnees necessaires a l'execution.
- vue technique du contrat EBICS delivre par la banque:
  - ordres autorises;
  - BTF autorises;
  - permissions par abonne, compte, niveau d'autorisation et seuils;
  - options supportees (`Recovery`, `PreValidation`, `HKD`/`HTD`, `HAA`,
    details EDS, etc.).

### 3.3 Transactions et ordres EBICS

Gateway doit porter:

- transactions EBICS et leurs etats;
- ordres administratifs EBICS;
- flux fichiers EBICS;
- EDS au sens protocolaire:
  - consultation (`HVU`, `HVZ`, `HVD`, `HVT`);
  - ajout de signature (`HVE`);
  - annulation (`HVS`);
- `HAC` et l'exploitation technique associee;
- RTN et l'auto-pull qui en decoule.

### 3.4 Automatisation technique

Gateway doit porter:

- automatisation de la rotation des cles relevant du protocole;
- generation de lettre EBICS;
- rupture d'automatisme et attente de confirmation externe;
- collecte automatisee des reports;
- publication d'evenements techniques vers le SI metier;
- rejeu, reprise et observabilite.

## 4. Ce qui reste hors Gateway

### 4.1 Decision metier

Gateway ne doit pas porter:

- decision d'accepter, rejeter ou liberer un ordre pour raison metier;
- arbitrage de montant, de delegation ou de role metier;
- circuits d'approbation bancaire internes au client;
- politiques fonctionnelles specifiques a une organisation.

En particulier, Gateway ne doit pas porter comme source d'autorite:

- le contrat juridique ou commercial entre banque et client;
- la politique interne d'habilitation metier du client.

### 4.2 Workflow humain de signature

Gateway ne doit pas porter:

- la liste des signataires a designer selon les regles internes du client;
- le pilotage humain de collecte des signatures;
- la logique de relance, d'escalade ou d'approbation metier;
- les decisions de cosignature qui depassent les faits exposes par EBICS.

Gateway peut seulement:

- exposer les ordres en attente de signature;
- exposer les hash, etats et details utiles;
- accepter une demande technique de signature ou d'annulation deja decidee.

### 4.3 Consommation des payloads metier

Gateway ne doit pas porter:

- l'interpretation fonctionnelle des payloads bancaires;
- la comptabilisation, la reconciliation ou les traitements financiers internes;
- la transformation des reports en decisions metier.

Gateway peut seulement:

- collecter le payload;
- le tracer;
- le pousser ou l'exposer a l'application metier.

## 5. Regles de conception concretes

### 5.1 Regle de test de frontiere

Avant d'ajouter une fonctionnalite dans Gateway, il faut se poser la question:

- "si cette logique etait executee sans application metier derriere, aurait-elle
  encore un sens purement protocolaire ou technique ?"

Si la reponse est non, la fonctionnalite est probablement hors Gateway.

### 5.2 Regle sur les etats

Gateway ne doit stocker que:

- des etats techniques;
- des etats protocolaires;
- des etats d'exploitation.

Gateway ne doit pas stocker comme etat de reference:

- des decisions metier finales;
- des validations fonctionnelles;
- des statuts organisationnels propres a un client.

En revanche, Gateway peut stocker des "vues techniques" de reference quand
elles sont publiees par la banque via EBICS et directement utiles a
l'execution.

### 5.3 Regle sur les API

Les API Gateway doivent:

- exposer des faits techniques;
- exposer des commandes protocolaires;
- exposer des evenements techniques.

Les API Gateway ne doivent pas:

- embarquer des verbes metier propres a un domaine bancaire client;
- imposer un workflow humain particulier.

### 5.4 Regle sur `Transfer`

`Transfer` ne doit etre utilise que si:

- un vrai fichier est manipule;
- le pipeline standard Gateway apporte de la valeur.

Les ordres administratifs et EDS non orientes fichier doivent rester dans une
historisation EBICS dediee.

### 5.5 Regle sur les integrations externes

Toute decision non protocolaire doit etre prise:

- avant l'appel a Gateway;
- ou apres reception d'un evenement technique emis par Gateway.

Gateway doit donc fournir:

- une outbox technique fiable;
- des identifiants de correlation;
- des capacites de rejeu.
- plusieurs modes de remise au SI metier:
  - filesystem;
  - REST/API;
  - CLI;
  - messagerie asynchrone, en particulier `AMQP 0.9.1` et `AMQP 1.0`.

## 6. Application aux sujets les plus sensibles

### 6.1 Initialisation

Dans Gateway:

- emission `INI`, `HIA`, `H3K`, `HPB`;
- generation de lettre;
- suivi technique de l'etat;
- attente de confirmation externe.

Hors Gateway:

- pilotage humain complet de la relation banque;
- decision organisationnelle de mise en service.

### 6.2 Rotation de cles

Dans Gateway:

- preparation technique;
- emission des ordres;
- bascule technique;
- tracabilite et alertes.

Hors Gateway:

- politique de renouvellement PKI d'entreprise;
- validation de conformite et gouvernance securite interne.

### 6.3 Signatures distribuees

Dans Gateway:

- consultation des ordres a signer;
- recuperation des details;
- ajout technique de signature;
- annulation technique.

Hors Gateway:

- determination des signataires;
- orchestration humaine de signature;
- politique metier d'approbation.

### 6.4 Reports et RTN

Dans Gateway:

- reception RTN;
- idempotence;
- mapping vers les pulls;
- telechargement EBICS;
- historisation;
- emission vers le SI metier.

Hors Gateway:

- interpretation fonctionnelle des reports;
- decisions a partir du contenu.

### 6.5 Contrat technique et permissions EBICS

Dans Gateway:

- stockage d'une projection technique du contrat publie par la banque via
  `HPD`, `HKD`, `HTD`, `HAA`;
- controle preventif des ordres et BTF manifestement hors contrat;
- exposition de la date de rafraichissement, de la source et de la version de
  cette projection;
- eventuel durcissement local plus restrictif pour securiser l'exploitation.

Hors Gateway:

- contrat commercial de service;
- arbitrage fonctionnel sur les flux effectivement utilises par l'entreprise;
- politique metier plus restrictive que le minimum bancaire.

## 7. Criteres d'acceptation de la frontiere

La frontiere est consideree tenue si:

- aucune fonctionnalite des lots 1 et 2 n'exige de modele metier bancaire
  specifique;
- les nouveaux etats introduits restent techniques;
- aucune API ne force un workflow humain propre a un client;
- les integrations externes peuvent rester generiques;
- `Transfer` n'est pas detourne pour porter des ordres administratifs.

## 8. Motif de remise en cause

Il faudra reouvrir cette frontiere si l'on decouvre que:

- les clients attendent de Gateway qu'elle pilote directement leurs circuits de
  signature;
- les payloads metier doivent etre interpretes dans Gateway pour que la chaine
  rende un vrai service;
- les ordres EDS ne peuvent pas etre exploites utilement sans logique metier
  embarquee;
- l'administration EBICS exige des concepts fonctionnels propres a chaque
  etablissement.
