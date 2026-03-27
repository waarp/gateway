# Specifications fonctionnelles

## 1. Objet

Definir les besoins fonctionnels pour integrer EBICS dans Waarp Gateway en
capitalisant sur les concepts existants de Gateway:

- gestion de protocoles multiples;
- exploitation en mode client et serveur;
- gestion des partenaires, comptes, regles et transferts;
- historique, supervision et administration centralisee;
- execution de traitements automatiques autour des transferts.

## 2. Perimetre

Le perimetre vise une integration EBICS 3.0.2 (`H005`) basee sur la librairie
WAARP EBICS.

Sont inclus:

- exposition d'un endpoint serveur EBICS dans Gateway;
- emission de flux EBICS vers une banque ou un partenaire distant;
- gestion des ordres administratifs, de telechargement et de depot;
- automatisation des traitements protocolaires;
- administration des identites, cles, certificats, habilitations et flux.
- automatisation de la rotation des cles relevant du protocole;
- recuperation automatisee des reports et notifications RTN.

Ne sont pas inclus en premiere intention:

- la logique bancaire metier propre a chaque etablissement;
- les workflows de validation metier hors protocole;
- les circuits metier de signature et de decision;
- les politiques PKI d'entreprise non directement necessaires au protocole.

## 3. Vision fonctionnelle

Gateway doit devenir une plateforme MFT capable d'orchestrer EBICS comme les
autres protocoles supportes, sans casser ses mecanismes existants de pilotage.

L'integration doit permettre:

- d'operer EBICS en mode serveur pour exposer un service a des clients EBICS;
- d'operer EBICS en mode client pour se connecter a une banque distante;
- d'automatiser les ordres EBICS repetitifs et leur post-traitement;
- de conserver une tracabilite complete par transaction, ordre et transfert;
- d'offrir un niveau de securite au moins equivalent aux protocoles deja geres.
- de servir de passe-plat fiable entre le monde protocolaire EBICS et
  l'application metier.

## 4. Acteurs

- Administrateur Gateway: configure les endpoints, les comptes, les cles, les
  certificats, les politiques et la supervision.
- Operateur d'exploitation: suit les traitements, rejoue, diagnostique et
  supervise.
- Systeme client EBICS distant: consomme le serveur EBICS expose par Gateway.
- Banque ou serveur EBICS distant: cible des traitements EBICS sortants.
- Moteur de traitements Gateway: enchaine les pre-tasks, post-tasks,
  transformations et routages.

## 5. Capacites fonctionnelles attendues

### 5.1 Gestion des identites EBICS

Le systeme doit permettre de gerer:

- les `HostID`;
- les couples `PartnerID` / `UserID`;
- les cles et certificats d'authentification, chiffrement et signature;
- l'etat d'initialisation et d'activation EBICS d'un abonne;
- le cycle de vie des cles et certificats;
- les cles publiques banque et leurs digests.
- la projection technique du contrat banque/client utile a l'execution EBICS.

### 5.1.1 Focus sur le contrat technique EBICS

Le systeme doit stocker une vue technique exploitable du contrat publie par la
banque via les ordres administratifs EBICS.

Cette vue doit au minimum couvrir:

- les capacites protocolaire annoncees par `HPD`;
- les ordres administratifs et BTF telechargeables annonces par `HAA`;
- les ordres, services, niveaux d'autorisation, comptes et seuils annonces par
  `HKD` / `HTD`;
- la date de rafraichissement, la source et l'etat de validite de cette vue.

Le systeme doit utiliser cette vue pour:

- assister la configuration;
- prevenir l'emission d'ordres manifestement hors contrat;
- reduire les erreurs d'exploitation et les rejets evitables.

Cette vue ne remplace pas:

- le contrat juridique ou commercial;
- les regles metier internes de l'entreprise;
- une politique applicative plus restrictive que celle de la banque.

### 5.2 Gestion des roles serveur et client

Le systeme doit permettre de declarer:

- un serveur EBICS local expose par Gateway;
- un client EBICS local pilote par Gateway;
- des partenaires EBICS distants;
- des comptes locaux et distants relies a ces entites.

### 5.3 Gestion des ordres EBICS

Le systeme doit permettre de prendre en charge au minimum:

- ordres d'initialisation et de gestion de cles: `HEV`, `INI`, `HIA`, `HPB`,
  `H3K`, `HCA`, `HCS`, `HSA`, `SPR`, `PUB`;
- ordres administratifs et de consultation: `HPD`, `HTD`, `HKD`, `HAA`;
- ordres de depot et de telechargement: `BTU`, `BTD`; `FUL` et `FDL` restent
  seulement des alias de compatibilite normalises vers ces ordres en cible
  `EBICS 3.0.2`;
- ordres de reporting ou de signature, selon priorisation metier: `HAC`, `HVD`,
  `HVE`, `HVT`, `HVU`, `HVZ`, `HVS`.

Le systeme doit distinguer fonctionnellement:

- les ordres EBICS conduisant a un flux fichier, suivis dans `Transfer`;
- les ordres EBICS non payload, suivis dans un objet d'exploitation dedie de
  type `EbicsOperation`.

### 5.4 Automatisation attendue

Le systeme doit automatiser autant que possible:

- la validation XSD et syntaxique des messages EBICS;
- la verification `AuthSignature`;
- le chiffrement/dechiffrement E002 si necessaire;
- la gestion des segments et de la reprise de transaction;
- la protection anti-rejeu par `Nonce` / `Timestamp`;
- le mapping des return codes EBICS dans les statuts d'exploitation Gateway;
- le declenchement des traitements de pre/post transfert;
- l'archivage et la mise a disposition des traces d'execution.
- la recuperation automatisee des reports;
- le declenchement automatise des pulls a partir des notifications RTN.

### 5.6 Focus sur la rotation des cles

Le systeme doit permettre de gerer le cycle de vie des cles EBICS sans rupture
de service non maitrisee.

Attendus:

- visualisation de la cle active, de la future cle et de l'historique;
- lancement controle des ordres de rotation (`PUB`, `HCA`, `HCS`, `HSA`, `SPR`,
  `H3K` selon le profil banque);
- coexistence temporaire entre ancien et nouveau materiel quand le protocole ou
  la banque l'impose;
- tracabilite des dates de generation, d'envoi, d'activation et de retrait;
- blocage ou alerte si une cle expire ou devient non conforme;
- evidence operateur des validations associees.

Cette automatisation est dans le perimetre car elle releve directement du cycle
de vie protocolaire EBICS.

### 5.7 Focus sur la rupture d'automatisme de l'initialisation

L'initialisation EBICS ne doit pas etre traitee comme un flux 100 % automatique
de bout en bout.

Attendus:

- distinction claire entre phase technique EBICS et phase d'activation hors
  bande;
- generation de la lettre EBICS apres `INI` / `HIA` ou `H3K` si necessaire;
- mise a disposition de la lettre pour telechargement, impression ou
  transmission selon le processus banque;
- etats explicites: `draft`, `technical-sent`, `letter-generated`,
  `letter-sent`, `awaiting-bank-activation`, `activated`, `rejected`;
- capacite operateur de suspendre, reprendre, invalider ou rejouer la phase
  d'initialisation;
- interdiction des flux de production tant que l'activation n'est pas confirmee.

### 5.8 Focus sur les notifications RTN

Le systeme doit introduire une capacite de notifications temps reel, absente de
Gateway aujourd'hui.

Attendus:

- reception d'un signal temps reel indiquant qu'un contenu est disponible;
- validation, journalisation et deduplication de la notification;
- transformation de la notification en declenchement de flux EBICS standards
  (`FDL`, `BTD`, `HAC` ou autres profils supportes);
- configuration d'une politique par profil: manuel, automatique, filtre;
- capacite de reprise, rejeu et mise en quarantaine d'une notification;
- observabilite complete entre notification, ordre declenche et resultat.

Cette automatisation est dans le perimetre car elle releve d'une orchestration
technique de collecte EBICS.

### 5.8 bis Focus sur les return codes EBICS

Le systeme doit traiter les return codes EBICS comme une information a deux
dimensions:

- scope `technical`;
- scope `business`.

Attendus:

- conservation des deux scopes sans ecrasement dans un champ unique;
- restitution des codes bruts et des messages associes;
- derivation d'un statut d'exploitation Gateway lisible;
- derivation d'une politique `retry / replay / recovery` compatible avec le
  protocole;
- interdiction du retry automatique sur simple rejet `business`;
- exposition claire au SI metier des rejets fonctionnels publies par la
  banque.

### 5.9 Frontiere avec l'application metier

Le systeme doit exposer un passe-plat vers l'application metier pour tout ce qui
ne releve pas strictement du protocole.

Cela couvre:

- decisions de validation;
- arbitrages metier;
- workflow metier de signature;
- decisions de liberation ou rejet non protocolaires.

La frontiere inclut aussi le principe suivant:

- Gateway porte le contrat technique EBICS necessaire a l'execution securisee;
- l'application metier porte le contrat fonctionnel et les restrictions
  organisationnelles propres au client.

### 5.10 Modes d'integration avec l'application metier

Le systeme doit proposer plusieurs modes standards d'integration avec le SI
metier.

Modes minimaux cibles:

- filesystem;
- API REST;
- CLI;
- messagerie asynchrone.

La messagerie asynchrone doit etre consideree comme un axe de valeur ajoutee
important, en particulier via:

- `AMQP 0.9.1` pour les environnements de type RabbitMQ;
- `AMQP 1.0` pour les environnements plus orientés interoperabilite messaging.

Usages cibles:

- emission d'evenements techniques EBICS vers le metier;
- reception de demandes d'execution deja decidees par le metier;
- decouplage temporel entre collecte EBICS et consommation applicative;
- meilleure resilence et meilleure industrialisation des echanges.

### 5.10 bis Ergonomie des ordres payload

Le systeme doit permettre de soumettre un ordre payload EBICS:

- soit en renseignant explicitement tous les attributs protocolaires
  necessaires;
- soit en referencant une configuration predefinie.

Attendus:

- prise en charge de profils payload reutilisables;
- possibilite de surcharger un profil par des parametres explicites;
- articulation claire entre:
  - semantique EBICS du payload;
  - politique technique Gateway de routage/remise.

La separation recommandee est:

- un profil EBICS dedie pour `serviceName`, `serviceOption`, `scope`,
  `msgName` et parametres voisins;
- `Rule` pour les aspects techniques Gateway lies au fichier.

### 5.5 Exploitation et pilotage

L'administration doit permettre:

- la creation et modification des configurations EBICS via REST, CLI et UI;
- la consultation des transactions, ordres, operations EBICS, transferts et
  historiques;
- la supervision des erreurs protocolaires et metier;
- la reprise ou le rejeu de traitements supportes par le protocole;
- l'export d'elements de diagnostic.

## 6. Cas d'usage principaux

### 6.1 Serveur EBICS entrant

Un client EBICS distant depose ou consulte un flux via Gateway.

Attendus:

- authentification forte;
- controle des habilitations par ordre;
- validation stricte;
- persistence durable de la transaction;
- integration au pipeline de traitement Gateway.

### 6.2 Client EBICS sortant planifie

Gateway execute un ordre EBICS vers une banque distante de maniere planifiee ou
a la demande.

Attendus:

- selection du client, du partenaire et du compte EBICS;
- generation ou reutilisation des parametres d'ordre;
- telechargement ou depot;
- exploitation automatique du resultat dans Gateway.

### 6.3 Onboarding d'un abonne EBICS

Un administrateur initialise un nouvel abonne EBICS.

Attendus:

- creation des identites;
- chargement ou generation des cles/certificats;
- execution des ordres d'initialisation;
- production de la lettre EBICS quand requise;
- validation operateur de l'envoi hors bande;
- suivi des etats de mise en service.

### 6.4 Rotation de cles

Un administrateur ou un operateur renouvelle le materiel cryptographique d'un
abonne ou d'un host banque.

Attendus:

- preparation de la rotation;
- controle des dependances et de la fenetre de bascule;
- emission des ordres de mise a jour;
- verification du nouvel etat;
- conservation d'evidences et retour arriere si possible.

### 6.5 Notification RTN recue

Gateway recoit une notification RTN indiquant qu'un jeu de donnees est
disponible.

Attendus:

- validation du message et de sa source;
- calcul d'une cle d'idempotence;
- enregistrement de l'evenement;
- resolution du plan de pull a executer;
- execution automatique ou mise en attente operateur selon la politique.

### 6.6 Signature recue ou constatee

Gateway constate, verifie et expose les informations de signature, puis transmet
la situation a l'application metier si une decision non protocolaire est
necessaire.

## 7. Rapprochement fonctionnel avec l'existant Gateway

Correspondances directes envisageables:

- serveur EBICS local -> `LocalAgent`;
- client EBICS local -> `Client`;
- banque/partenaire EBICS distant -> `RemoteAgent`;
- utilisateur EBICS rattache -> `LocalAccount` ou `RemoteAccount` selon le sens;
- droits d'usage -> `Rule` + `RuleAccess`;
- execution unitaire -> `Transfer` + `TransferInfo`;
- traitements complementaires -> `Tasks`;
- exposition admin -> REST/CLI/UI existants.

Extensions fonctionnelles necessaires:

- persistance des concepts EBICS propres: `HostID`, `PartnerID`, `UserID`,
  etat d'activation, cles banque, transactions EBICS, segments, fenetre de
  rejeu;
- objets de consultation orientes ordre EBICS en plus de la notion de
  transfert de fichier;
- suivi des codes retour EBICS et des details de protocoles.
- gestion d'un workflow manuel/hors bande pour la lettre EBICS.
- gestion d'un cycle de vie de rotation de cles distinct du simple stockage des
  credentials.
- gestion d'un canal d'evenements RTN avec idempotence et orchestration de
  pulls.
- exposition d'interfaces de passe-plat vers l'application metier.
- stockage d'une vue technique des capacites et permissions publiees par la
  banque via `HPD` / `HKD` / `HTD` / `HAA`.

## 8. Exigences non fonctionnelles

### 8.1 Securite

- TLS obligatoire en production;
- mTLS selon les profils de contrepartie;
- validation stricte des signatures et certificats;
- gouvernance explicite des rotations et de l'obsolescence des cles;
- chiffrement des secrets au repos;
- protection anti-rejeu persistante;
- auditabilite complete des actions d'administration.
- blocage preventif des ordres manifestement hors contrat technique connu.

### 8.2 Performance

- pas de duplication inutile des flux;
- streaming privilegie pour les charges volumineuses;
- persistance optimisee pour la reprise et la segmentation;
- maintien des capacites de parallelisme de Gateway;
- maitrise des timeouts et des limites de taille.

### 8.3 Exploitabilite

- journaux exploitables par ordre et transaction;
- correlation entre identite EBICS, transfert Gateway et historique;
- suivi operateur des etapes manuelles de la lettre EBICS;
- supervision des evenements RTN et de leur transformation en pulls;
- supervision unifiee avec les autres protocoles;
- demarrage et arret coherents avec le cycle de vie Gateway.

## 9. Principes de priorisation

Priorite 1:

- endpoint serveur EBICS;
- client EBICS pour `HEV`, `INI`, `HIA`, `HPB`, `FUL`, `FDL`, `BTU`, `BTD`;
- persistance durable des transactions et nonces;
- administration de base des identites et cles;
- gestion explicite de la lettre EBICS et des etats d'initialisation;
- historique et supervision.

Priorite 2:

- ordres administratifs complets `HPD`, `HTD`, `HKD`, `HAA`;
- rotation de cles industrialisee;
- premiere capacite RTN pilotee;
- reporting `HAC`, `HVD`, `HVE`, `HVT`, `HVU`, `HVZ`;
- politiques de signature plus fines;
- ecrans UI dedies.

Priorite 3:

- industrialisation multi-banque avancee;
- optimisation fine des campagnes de reprise;
- outillage d'onboarding et d'audit avance.
