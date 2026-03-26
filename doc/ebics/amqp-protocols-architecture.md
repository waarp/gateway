# Architecture des protocoles AMQP dans Gateway

## 1. Objet

Ce document cadre l'introduction de:

- `AMQP 0.9.1`
- `AMQP 1.0`

comme deux protocoles Gateway a part entiere, independants d'EBICS.

L'objectif est double:

- renforcer Gateway comme plateforme d'integration;
- fournir un socle de decouplage robuste avant l'integration EBICS.

## 2. Pourquoi proposer deux versions

Ce n'est pas intuitif, mais le choix est defendable.

Les deux versions ne recouvrent pas exactement les memes ecosystems ni les
memes habitudes d'integration.

En premiere approximation:

- `AMQP 0.9.1` est tres souvent associe a des brokers de type `RabbitMQ`;
- `AMQP 1.0` est davantage positionne sur l'interoperabilite messaging
  d'entreprise et des environnements de type `JMS`, brokers d'entreprise,
  services cloud et integration heterogene.

Autrement dit:

- proposer seulement `AMQP 0.9.1` reviendrait a coller surtout a l'ecosysteme
  RabbitMQ;
- proposer seulement `AMQP 1.0` ne couvrirait pas naturellement certains
  usages RabbitMQ historiques ou les habitudes d'integration les plus courantes
  dans certains SI.

Conclusion:

- il est plus juste de les penser comme deux protocoles distincts plutot que
  comme deux variantes d'un meme detail technique.

## 3. Position produit recommande

La position recommande est:

- `AMQP 0.9.1` = protocole Gateway natif
- `AMQP 1.0` = protocole Gateway natif

et non:

- "une simple option de sortie d'EBICS".

Pourquoi:

- ces protocoles ont une valeur propre hors EBICS;
- ils peuvent servir d'autres flux et usages de Gateway;
- ils suivent naturellement le fonctionnement nominal de Gateway:
  - `LocalAgent`
  - `Client`
  - `RemoteAgent`
  - `ProtoConfig`
  - cycle de vie `Start` / `Stop` / `State`
  - administration REST / CLI / UI
  - observabilite et historique.

## 4. Place d'AMQP par rapport a EBICS

Le bon ordre conceptuel est:

1. Gateway sait parler `AMQP 0.9.1`
2. Gateway sait parler `AMQP 1.0`
3. EBICS utilise ensuite ce socle pour le passe-plat metier

Donc:

- AMQP n'est pas subordonne a EBICS;
- EBICS devient un consommateur privilegie de cette capacite d'integration.

Cela rend l'architecture plus propre:

- EBICS reste concentre sur le protocole bancaire;
- Gateway fournit deja les mecanismes de remise asynchrone;
- le SI metier peut s'integrer sans coupling fort au flux EBICS lui-meme.

## 5. Difference de positionnement entre 0.9.1 et 1.0

## 5.1 `AMQP 0.9.1`

Profil cible:

- SI deja equipes autour de `RabbitMQ`;
- integration applicative pragmatique;
- topologies classiques `exchange` / `queue` / `binding`;
- besoins rapides de publication/consommation.

Valeur pour Gateway:

- adoption probable plus simple dans de nombreux contextes techniques;
- bon candidat pour un premier protocole messaging fortement demandable.

## 5.2 `AMQP 1.0`

Profil cible:

- SI d'entreprise cherchant davantage d'interoperabilite;
- environnements proches des usages `JMS` ou d'infrastructures messaging plus
  standards enterprise;
- integration avec certains brokers et services cloud supportant AMQP 1.0.

Valeur pour Gateway:

- meilleure couverture des architectures enterprise heterogenes;
- positionnement plus large que le seul monde RabbitMQ.

## 5.3 Consequence d'architecture

Il ne faut pas chercher a masquer les differences sous un faux protocole
unique.

Il faut plutot:

- partager un socle commun de concepts;
- assumer deux modules protocolaires distincts.

## 6. Architecture cible dans Gateway

## 6.1 Modules protocolaires

Modules cibles:

- `pkg/protocols/modules/amqp091/`
- `pkg/protocols/modules/amqp10/`

Chaque module doit:

- implementer l'interface standard de protocole Gateway;
- avoir son propre `ProtoConfig`;
- fournir ses services client et serveur si pertinents;
- s'integrer au registre de protocoles comme FTP, SFTP, HTTP, etc.

## 6.2 Socle commun a mutualiser

Ce qui peut etre mutualise:

- modele d'outbox / inbox;
- correlation IDs;
- idempotence keys;
- retry policy;
- dead-letter policy;
- observabilite de livraison;
- administration des profils de connecteurs;
- contrats de message techniques.

Ce qui doit rester specifique:

- details de connexion;
- semantique de publication / consommation;
- mapping fin des options de broker;
- gestion des particularites 0.9.1 vs 1.0.

## 6.3 Objets d'exploitation

Les objets Gateway existants peuvent etre reutilises de maniere naturelle:

- `LocalAgent` pour un endpoint ou un service local expose;
- `Client` pour un producteur / consommateur sortant;
- `RemoteAgent` pour un broker ou un endpoint distant;
- `Credential` pour secrets, certificats, SASL, TLS;
- `Transfer` seulement quand un vrai fichier circule;
- historique dedie pour les evenements purement messages.

## 7. Cas d'usage cibles

## 7.1 Hors EBICS

- publication d'evenements de transfert;
- consommation de commandes techniques;
- integration avec des applications internes;
- orchestration asynchrone de traitements Gateway.

## 7.2 Avec EBICS

- emission d'evenements EBICS vers le metier;
- reception de commandes d'execution deja decidees;
- decouplage entre collecte RTN / reports et consommation applicative;
- limitation du coupling REST synchrone.

## 8. Pourquoi cela peut etre un prealable a EBICS

Ce n'est pas un prealable strictement obligatoire au sens technique.

En revanche, c'est un excellent prealable:

- d'architecture;
- de produit;
- de valeur percue.

Pourquoi:

- cela evite que la valeur ajoutee de Gateway repose uniquement sur EBICS;
- cela dote d'abord Gateway d'une vraie capacite d'integration moderne;
- cela simplifie ensuite le passe-plat EBICS vers le metier;
- cela rend la story produit plus credible.

La formulation la plus juste est donc:

- `AMQP 0.9.1` et `AMQP 1.0` sont des prealables architecturaux pertinents a
  EBICS.

## 9. Strategie de mise en oeuvre

### 9.1 Option recommandee

1. introduire `amqp091`;
2. introduire `amqp10`;
3. valider l'outbox/inbox mutualisee;
4. brancher ensuite EBICS dessus.

### 9.2 Variante acceptable

Si le budget impose une marche plus prudente:

1. introduire `amqp091` en premier;
2. garder `amqp10` cadre mais differe;
3. brancher EBICS sur ce premier socle;
4. ajouter `amqp10` ensuite.

Cette variante est acceptable, mais moins aboutie produit.

## 10. Recommandation finale

La recommendation est:

- oui a `AMQP 0.9.1` et `AMQP 1.0` comme protocoles Gateway autonomes;
- oui a leur traitement comme capacite transverse de la plateforme;
- oui a leur positionnement comme prealable architectural pertinent avant
  EBICS;
- non a une implementation AMQP uniquement vue comme "sortie technique EBICS".
