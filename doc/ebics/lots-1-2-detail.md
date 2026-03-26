# Detail des lots 1 et 2

## 1. Objet

Ce document detaille les deux premiers lots, non pour lancer le dev
immediatement, mais pour servir de phase de preuve architecturale.

Ces deux lots doivent etre evalues avec la ligne directrice suivante:

- Gateway porte les automatismes protocolaires;
- l'application metier porte les decisions non protocolaires.

Le document [frontiere-protocole-metier.md](c:\MonProjet\Waarp-Gateway\doc\ebics\frontiere-protocole-metier.md)
fait foi pour arbitrer les cas ambigus.

## 1.1 Resultat attendu

Les lots 1 et 2 ne doivent pas seulement montrer que "cela fonctionne".

Ils doivent montrer que:

- l'integration EBICS reste dans le bon perimetre de Gateway;
- la valeur apportee par Gateway est reelle;
- le cout d'intrusion dans le coeur de Gateway reste borne.

## 1.2 Methode de validation

Chaque sous-lot doit etre evalue selon quatre axes:

- `adequation protocolaire`
- `intrusion coeur Gateway`
- `respect de la frontiere protocole / metier`
- `valeur d'exploitation`

La decision se prend ensuite avec:

- preuves observees;
- ecarts identifies;
- decision `GO`, `GO sous reserve`, ou `NO-GO`.

## 2. Lot 1 - Socle protocolaire EBICS

### 2.1 Objectif

Verifier qu'EBICS peut s'inserer proprement dans le cadre de module protocolaire
de Gateway.

### 2.2 Sous-lot 1A - Design du module

Taches:

- definir la structure cible `pkg/protocols/modules/ebics/`;
- figer les fichiers minimaux:
  - `module.go`
  - `config.go`
  - `server.go`
  - `client.go`;
- verifier l'alignement avec [modules.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\modules.go).

Livrable:

- structure de package validee.

### 2.3 Sous-lot 1B - Contrats de configuration

Taches:

- definir `ServerConfig`, `ClientConfig`, `PartnerConfig`;
- fixer les champs minimaux de `ProtoConfig`;
- distinguer ce qui reste en `ProtoConfig` de ce qui doit sortir en tables
  dediees;
- definir `ValidServer`, `ValidClient`, `ValidPartner`.

Livrable:

- contrat de configuration EBICS stable.

### 2.4 Sous-lot 1C - Registration et validation

Taches:

- enregistrer `ebics` dans [modules.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\modules.go);
- verifier l'impact sur [config.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\config.go);
- verifier que `LocalAgent`, `RemoteAgent` et `Client` acceptent le protocole.

Livrable:

- protocole reconnu par les modeles et la validation.

### 2.5 Sous-lot 1D - Squelettes serveur et client

Taches:

- definir les structures des services;
- definir `Start`, `Stop`, `State`;
- preparer les dependances futures vers `lib-ebics`;
- raccorder le cycle de vie a
  [server.go](c:\MonProjet\Waarp-Gateway\pkg\gatewayd\server.go).

Livrable:

- services instanciables, meme incomplets.

### 2.6 Sous-lot 1E - Administration minimale

Taches:

- verifier l'effet de bord sur le routeur REST
  [router.go](c:\MonProjet\Waarp-Gateway\pkg\admin\rest\router.go);
- verifier ce que les routes existantes apportent deja pour:
  - serveurs
  - clients
  - partenaires
  - comptes
  - credentials;
- identifier les enrichissements minimaux.

Livrable:

- liste de ce qui est reutilisable tel quel.

### 2.7 Sous-lot 1F - Tests de faisabilite

Taches:

- test de registration du protocole;
- test de validation des `ProtoConfig`;
- test de creation d'un `LocalAgent` EBICS;
- test de creation d'un `Client` EBICS;
- test de demarrage/arret d'un service EBICS minimal.

Livrable:

- premiere preuve d'integration modulaire.

### 2.8 Question de decision a l'issue du lot 1

Le lot 1 doit permettre de repondre a:

- "EBICS rentre-t-il proprement dans le framework de protocoles Gateway ?"
- "Peut-on le faire sans embarquer de logique metier bancaire dans Gateway ?"

Si la reponse est non, il faut stopper avant le lot 2.

### 2.9 Grille de validation du lot 1

#### Criteres `GO`

- le module `ebics` s'insere via les mecanismes standards de registre
  protocolaire;
- les `ProtoConfig` restent lisibles et limites a la configuration technique;
- aucun concept metier bancaire n'est necessaire pour demarrer un serveur ou un
  client minimal;
- l'administration existante peut porter le minimum viable sans filiere
  parallele;
- les premiers tests montrent une integration modulaire et non une exception
  structurelle.

#### Signaux d'alerte

- besoin d'ajouter des branches conditionnelles EBICS dans des composants coeur
  non prevus pour cela;
- besoin de modeles metier supplementaires juste pour enregistrer le protocole;
- besoin d'une API d'administration totalement separee;
- impossibilite de representer proprement serveur, client, partenaire et
  comptes avec les objets Gateway existants.

#### Preuves attendues

- schema cible du package `pkg/protocols/modules/ebics/`;
- contrat `ProtoConfig` commente avec repartition `config` vs `tables`;
- liste des points de contact exacts avec le coeur Gateway;
- tableau des reutilisations REST/CLI existantes;
- tests de faisabilite documentes.

#### Decision lot 1

- `GO` si EBICS entre proprement comme protocole natif supplementaire;
- `GO sous reserve` si quelques enrichissements coeur sont necessaires mais
  restent localises;
- `NO-GO` si le module ne peut exister qu'en deformant le framework de
  protocoles.

## 3. Lot 2 - Stores durables EBICS

### 3.1 Objectif

Verifier que la persistance specifique EBICS reste modulaire et non intrusive.

Le lot 2 doit aussi verifier que la persistance sert uniquement:

- les identites et etats protocolaires;
- les transactions EBICS;
- les nonces et la reprise;

et non une logique metier applicative cachee dans Gateway.

### 3.2 Sous-lot 2A - Design du modele persistant

Taches:

- finaliser les tables:
  - `ebics_hosts`
  - `ebics_subscribers`
  - `ebics_bank_keys`
  - `ebics_transactions`
  - `ebics_transaction_segments`
  - `ebics_nonce_window`
  - `ebics_contract_views`;
- fixer colonnes, indexes et contraintes;
- fixer les liens avec les objets Gateway existants.

Livrable:

- modele logique final.

### 3.3 Sous-lot 2B - Migrations et structs

Taches:

- ajouter les structs dans `pkg/model/`;
- raccorder l'initialisation de tables;
- ajouter les migrations dans
  [list.go](c:\MonProjet\Waarp-Gateway\pkg\database\migrations\list.go);
- verifier SQLite, PostgreSQL, MySQL.

Livrable:

- schema migrable.

### 3.4 Sous-lot 2C - Adapter `KeyStore`

Taches:

- definir le mapping entre `Credential` et le materiel cryptographique EBICS;
- separer ce qui reste dans `Credential` et ce qui va dans `ebics_bank_keys`;
- implementer le stockage des cles abonne et banque.

Livrable:

- adapter `KeyStore`.

### 3.5 Sous-lot 2D - Adapter `SubscriberStore`

Taches:

- implementer le mapping `(HostID, PartnerID, UserID)` -> compte Gateway;
- definir les etats subscriber minimaux;
- definir les metadonnees d'habilitation minimales.

Livrable:

- adapter `SubscriberStore`.

### 3.6 Sous-lot 2E - Adapter `TxStore`

Taches:

- implementer la persistance transactionnelle EBICS hors `Transfer`;
- definir le lien optionnel avec `Transfer.ID`;
- stocker `tx_id`, `order_type`, `status`, `segment_count`,
  `recovery_counter`, `recovery_point`.

Livrable:

- adapter `TxStore`.

### 3.7 Sous-lot 2F - Adapter `NonceStore`

Taches:

- implementer la fenetre anti-rejeu;
- definir TTL et purge;
- verifier l'impact volumetrique attendu.

Livrable:

- adapter `NonceStore`.

### 3.8 Sous-lot 2G - Couche d'acces

Taches:

- definir une couche d'acces dediee aux objets EBICS;
- eviter de disperser la logique de store dans les handlers REST;
- preparer une reutilisation par les futurs workflows.

Livrable:

- couche d'acces coherente.

### 3.9 Sous-lot 2H - Tests de faisabilite

Taches:

- tests CRUD;
- tests contraintes;
- tests purge nonce;
- tests reprise transactionnelle;
- tests de correlation avec agents et comptes Gateway.

Livrable:

- preuve que les stores sont durables et stables.

### 3.10 Sous-lot 2I - Mesure d'impact architecture

Taches:

- mesurer le nombre de nouvelles tables et repositories;
- mesurer le nombre de points de contact avec le coeur Gateway;
- verifier que `Transfer` n'a pas besoin d'etre refondu a ce stade;
- verifier que les routes REST existantes ne doivent pas etre cassees.

Livrable:

- conclusion argumentee sur la viabilite de l'option Gateway.

### 3.11 Question de decision a l'issue du lot 2

Le lot 2 doit permettre de repondre a:

- "Le cout de persistance specifique EBICS reste-t-il borne et compatible avec
  Gateway ?"
- "La persistance EBICS reste-t-elle technique/protocolaire et non metier ?"

Si oui, Gateway reste un bon candidat.

Si non, il faut reconsiderer serieusement l'option `from scratch`.

### 3.12 Grille de validation du lot 2

#### Criteres `GO`

- les nouvelles tables restent majoritairement techniques et protocolaires;
- `TxStore` et `NonceStore` vivent hors `Transfer` sans contorsion;
- la vue contractuelle technique publiee par la banque peut etre stockee sans
  devenir une source d'autorite metier;
- le lien avec les objets Gateway existants reste simple et explicable;
- la persistence des identites, transactions et nonces reste exploitable par
  les canaux d'administration standard;
- aucun workflow metier n'apparait dans le schema.

#### Signaux d'alerte

- besoin de surcharger `Transfer`, `TransferInfo` ou `Pipeline` pour des ordres
  non fichier;
- multiplication de tables d'etat fonctionnel non directement justifiees par la
  norme EBICS;
- besoin d'ouvrir systematiquement les payloads metier juste pour appliquer des
  regles de contrat;
- dependance forte de la persistance EBICS a des decisions metier propres au
  client;
- explosion des points de contact avec le coeur de Gateway.

#### Preuves attendues

- modele logique final annote par objet et responsabilite;
- justification de chaque table EBICS dediee;
- distinction explicite entre vue contractuelle technique et contrat metier;
- mapping explicite `objet Gateway <-> objet EBICS`;
- liste des cas ou `Transfer` reste volontairement absent;
- tests de faisabilite SQL et contraintes critiques.

#### Decision lot 2

- `GO` si la persistance EBICS reste un sous-modele technique borne;
- `GO sous reserve` si quelques ajustements coeur sont necessaires mais sans
  refonte de `Transfer` ni du `Pipeline`;
- `NO-GO` si la persistance EBICS devient dominante et deforme le modele
  Gateway.

## 4. Sortie attendue des lots 1 et 2

Les lots 1 et 2 ne doivent pas produire un produit fini.

Ils doivent produire une decision:

- `GO Gateway` si l'integration reste modulaire;
- `NO-GO Gateway` si l'integration force deja une refonte structurelle.

## 5. Livrables de decision recommandes

Pour que la decision soit exploitable, les lots 1 et 2 devraient produire en
sortie:

- une note d'ecarts entre besoin EBICS et existant Gateway;
- une matrice `reutilisation / extension / nouveau composant`;
- une carte des points d'intrusion dans le coeur Gateway;
- une liste de reserves a lever avant squelette de code;
- une recommandation finale explicite `GO`, `GO sous reserve`, `NO-GO`.
