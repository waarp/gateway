# Backlog et evaluation des librairies AMQP pour Gateway

## 1. Objet

Ce document precise:

- le backlog de mise en oeuvre des protocoles `AMQP 0.9.1` et `AMQP 1.0`
  dans Gateway;
- les librairies Go open source candidates;
- la recommandation de selection pour un usage robuste dans Gateway.

## 2. Clarification importante

Il faut distinguer:

- un protocole de messaging;
- un broker;
- une API d'acces.

En particulier:

- `AMQP 0.9.1` et `AMQP 1.0` sont des protocoles;
- `RabbitMQ`, `ActiveMQ Artemis`, `Amazon MQ for RabbitMQ` sont des brokers ou
  services de brokers;
- `JMS` n'est pas un protocole broker, mais une API Java.

Donc la bonne formulation est:

- `AMQP 1.0` est souvent plus present dans des environnements enterprise ou
  proches `JMS`;
- mais `JMS` n'est pas lui-meme l'equivalent d'`AMQP 1.0`.

## 3. Pourquoi les deux protocoles ont du sens

### 3.1 `AMQP 0.9.1`

Positionnement:

- tres naturel pour les ecosystèmes `RabbitMQ`;
- modele exchange / queue / binding familier;
- forte disponibilite de clients.

### 3.2 `AMQP 1.0`

Positionnement:

- protocole standardise OASIS / ISO;
- plus inter-operable entre brokers et outils enterprise;
- bien adapte a des environnements comme `ActiveMQ Artemis` et a certains
  services cloud ou offres managed compatibles.

### 3.3 Point de marche utile

Au 25 mars 2026, les sources que j'ai verifiees confirment notamment:

- `RabbitMQ` supporte `AMQP 0.9.1` comme protocole historique et `AMQP 1.0`
  nativement depuis la serie `4.x`;
- `ActiveMQ Artemis` supporte `AMQP 1.0`;
- `Amazon MQ for RabbitMQ` documente le support de `AMQP 0.9.1` et
  `AMQP 1.0`.

Cela renforce la pertinence d'un support double dans Gateway.

## 4. Candidats librairies Go

## 4.1 AMQP 0.9.1

### Candidat principal: `github.com/rabbitmq/amqp091-go`

Forces:

- maintenu par l'equipe RabbitMQ;
- derive du client historique `streadway/amqp`;
- mature, largement deploye, API connue;
- aucun besoin apparent de dependance native externe.

Points d'attention:

- centré sur le modele `AMQP 0.9.1`;
- certaines limites du modele et du client restent inherentes a ce protocole.

Verdict:

- `candidat recommande`.

### Candidat a eviter en base: `github.com/streadway/amqp`

Constat:

- le depot se presente lui-meme comme non activement maintenu;
- il recommande d'utiliser `rabbitmq/amqp091-go`.

Verdict:

- `a ne pas choisir pour un nouveau socle Gateway`.

## 4.2 AMQP 1.0

### Candidat principal: `github.com/Azure/go-amqp`

Forces:

- client Go AMQP 1.0 pur Go;
- projet actif et clairement positionne comme client AMQP 1.0;
- pas de dependance C annoncee dans le chemin nominal;
- orientation pratique pour des usages broker / service bus.

Points d'attention:

- projet historiquement tres visible dans l'ecosysteme Azure, meme si la
  bibliotheque vise un broker AMQP 1.0 conforme de maniere generale;
- a valider par interop sur les brokers cibles hors Azure.

Verdict:

- `candidat recommande en premiere intention`.

### Candidat secondaire: bindings Go de `Apache Qpid Proton`

Forces:

- projet AMQP 1.0 de reference cote Apache Qpid;
- forte legitimite protocolaire AMQP 1.0;
- bon alignement conceptuel avec l'ecosysteme enterprise.

Points d'attention:

- les packages Go visibles reposent sur la librairie `proton-C`;
- cela introduit une dependance native externe moins naturelle pour Gateway;
- cout d'exploitation, de packaging et de CI plus eleve.

Verdict:

- `candidat de reserve`, pas ideal comme premier choix pour Gateway.

### Candidats a eviter pour le socle

Exemple:

- anciennes librairies Go AMQP 1.0 en statut alpha ou peu actives.

Verdict:

- `non recommande` pour un protocole natif de Gateway.

## 5. Recommandation de selection

### 5.1 Selection recommandee

- `AMQP 0.9.1` -> `github.com/rabbitmq/amqp091-go`
- `AMQP 1.0` -> `github.com/Azure/go-amqp`

### 5.2 Pourquoi ce duo est defendable

- il evite une dependance native C pour la premiere version;
- il s'appuie sur des projets visibles et activement utilisables;
- il couvre deux mondes d'usage differents;
- il reste coherent avec un projet Go pur comme Gateway.

## 6. Criteres de validation avant adoption definitive

Avant de figer ce choix, il faudra verifier:

- compatibilite avec la version de Go du projet;
- qualite de la gestion TLS / SASL / auth;
- semantique ack/nack / settlement;
- gestion des reconnexions et erreurs reseau;
- comportement en backpressure;
- capacite a integrer observabilite et correlation;
- licence compatible avec Gateway;
- simplicite d'empaquetage et de CI.

## 7. Backlog de mise en oeuvre

### Lot A - Etude et spike librairies

Objectif:

- confirmer le choix des librairies sur des preuves courtes.

Taches:

- spike `amqp091-go` sur publication / consommation / TLS;
- spike `Azure/go-amqp` sur connexion, sender, receiver, settlement;
- test d'integration simple sur brokers cibles;
- note de choix finale.

Livrable:

- recommendation de librairie figee.

### Lot B - Protocole `amqp091`

Objectif:

- introduire `AMQP 0.9.1` comme protocole natif Gateway.

Taches:

- module `pkg/protocols/modules/amqp091/`;
- `ProtoConfig` client / serveur / partenaire;
- publication, consommation, ack, retry;
- administration REST/CLI minimale;
- observabilite.

Livrable:

- protocole `amqp091` exploitable.

### Lot C - Protocole `amqp10`

Objectif:

- introduire `AMQP 1.0` comme protocole natif Gateway.

Taches:

- module `pkg/protocols/modules/amqp10/`;
- `ProtoConfig` client / serveur / partenaire;
- sender / receiver / settlement;
- administration REST/CLI minimale;
- observabilite.

Livrable:

- protocole `amqp10` exploitable.

### Lot D - Socle mutualise

Objectif:

- mutualiser ce qui doit l'etre entre les deux protocoles.

Taches:

- outbox / inbox;
- contrats de message;
- correlation IDs;
- idempotence;
- retry / dead-letter;
- journalisation des livraisons.

Livrable:

- socle messaging transverse.

### Lot E - Integration EBICS

Objectif:

- brancher EBICS sur le socle AMQP.

Taches:

- publication d'evenements EBICS;
- reception de commandes techniques EBICS;
- correlation avec `ebics_event_outbox`;
- runbooks d'exploitation.

Livrable:

- passe-plat EBICS sur socle AMQP natif.

## 8. Decision provisoire

La decision la plus defendable a ce stade est:

- `oui` a `AMQP 0.9.1` et `AMQP 1.0` comme protocoles natifs Gateway;
- `oui` a un prealable AMQP avant EBICS;
- `oui` au duo de librairies
  `rabbitmq/amqp091-go` + `Azure/go-amqp` comme base de travail.
