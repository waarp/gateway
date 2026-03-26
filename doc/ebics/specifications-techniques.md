# Specifications techniques

## 1. Constat sur l'existant Gateway

L'existant apporte deja les briques structurantes suivantes:

- framework de protocoles modulaires via `pkg/protocols`;
- cycle de vie des services via `pkg/gatewayd/server.go`;
- modele de configuration protocolaire via `ProtoConfig` mappe et valide;
- persistance centralisee pour agents, comptes, credentials, regles et
  transferts;
- orchestration d'execution via `controller`, `pipeline` et `tasks`;
- administration via REST, CLI et interfaces web;
- chiffrement des secrets dans la base et gestion des certificats/cles.

Conclusion:

- EBICS doit etre integre comme protocole natif supplementaire;
- il ne faut pas construire un sous-systeme parallele decouple du coeur
  Gateway;
- les extensions doivent rester concentrees sur les besoins imposes par le
  protocole EBICS.

## 2. Constat sur la librairie WAARP EBICS

La librairie EBICS couvre deja:

- les flux H005;
- parsing XML, XSD, `AuthSignature`, E002;
- segmentation, recovery, anti-replay;
- mapping des return codes;
- facades serveur `provider/server` et client `client.NewProductionProfile`.

La librairie ne couvre volontairement pas:

- le modele de persistance applicative;
- les politiques metier;
- l'hebergement et l'observabilite de production.

Conclusion:

- Gateway doit reutiliser la librairie comme coeur protocolaire;
- Gateway doit implementer les stores durables et le modele d'administration.

## 3. Strategie d'integration

### 3.1 Principe

Introduire un nouveau module protocolaire `ebics` dans Gateway.

Ce module sera responsable de:

- l'instanciation des services EBICS serveur et client;
- la traduction entre le modele Gateway et les contrats de la librairie EBICS;
- le raccordement avec la persistance, l'observabilite et l'administration.

### 3.2 Positionnement

Ce module devra etre enregistre dans `pkg/protocols/modules.go` comme les autres
protocoles.

Il devra fournir:

- `NewServer(db, localAgent)`;
- `NewClient(db, client)`;
- `MakeServerConfig()`;
- `MakeClientConfig()`;
- `MakePartnerConfig()`.

## 4. Mapping cible avec le modele Gateway

### 4.1 Elements reutilisables sans changement majeur

- `LocalAgent`: serveur EBICS local;
- `Client`: client EBICS local;
- `RemoteAgent`: banque ou serveur EBICS distant;
- `LocalAccount`: utilisateur EBICS local attache a un serveur;
- `RemoteAccount`: utilisateur EBICS distant attache a une banque;
- `Credential`: support des secrets, cles et certificats;
- `Rule` et `RuleAccess`: habilitations et autorisations d'usage;
- `Transfer` et `TransferInfo`: orchestration et traque d'execution.

### 4.2 Extensions de modele necessaires

Le modele courant ne suffit pas pour porter durablement:

- les tuples d'identite EBICS `(HostID, PartnerID, UserID)`;
- l'etat d'initialisation des abonnes;
- les cles publiques banque et digests par `HostID`;
- les operations EBICS non payload;
- le `TxStore` EBICS;
- le `NonceStore` EBICS;
- le suivi fin par ordre EBICS et segment.
- le workflow d'initialisation avec rupture d'automatisme et lettre EBICS.
- le cycle de rotation de cles.
- les evenements RTN et leur etat d'execution.
- la vue technique du contrat et des permissions publies par la banque.

Il est recommande d'ajouter des tables dediees plutot que de surcharger
massivement `TransferInfo`.

`TransferInfo` peut rester utilise pour des metadonnees locales de transfert,
mais il ne doit pas devenir le support principal des correlations structurelles
EBICS, car son exploitation hors Gateway Waarp n'est pas garantie.

Un modele SQL detaille de reference est maintenu separement pour figer les
cles, relations et index avant implementation.

## 5. Extensions de persistance recommandees

### 5.1 Table logique `ebics_hosts`

Objectif:

- porter les `HostID` exposes ou cibles;
- associer les metadonnees d'exploitation et la strategie de securite.

Champs cibles:

- `id`
- `owner`
- `name`
- `host_id`
- `mode` (`server` / `remote`)
- `gateway_entity_type`
- `gateway_entity_id`
- `status`
- `options`

### 5.2 Table logique `ebics_subscribers`

Objectif:

- porter le tuple metier `HostID` / `PartnerID` / `UserID`;
- relier ce tuple a un compte Gateway.

Champs cibles:

- `id`
- `owner`
- `host_id`
- `partner_id`
- `user_id`
- `account_role`
- `local_account_id` ou `remote_account_id`
- `state`
- `permissions`
- `options`

### 5.3 Table logique `ebics_bank_keys`

Objectif:

- persister les cles publiques banque et digests par `HostID`.

### 5.4 Table logique `ebics_transactions`

Objectif:

- porter la transaction EBICS durable au-dela du seul transfert Gateway.

Champs cibles:

- `tx_id`
- `host_id`
- `partner_id`
- `user_id`
- `order_type`
- `transfer_id`
- `status`
- `segment_count`
- `created_at`
- `updated_at`

### 5.4 bis Table logique `ebics_operations`

Objectif:

- porter les ordres EBICS non payload et les operations purement
  protocolaires sans les forcer dans `Transfer`.

Champs cibles:

- `id`
- `host_id`
- `partner_id`
- `user_id`
- `operation_type`
- `order_type`
- `transaction_id`
- `request_id`
- `transfer_id`
- `status`
- `technical_return_code`
- `technical_return_message`
- `business_return_code`
- `business_return_message`
- `gateway_outcome`
- `retry_decision`
- `manual_action_required`
- `metadata`
- `created_at`
- `updated_at`

Principe important:

- Gateway doit stocker les return codes `technical` et `business`
  separement;
- le statut derive Gateway ne doit jamais remplacer ces champs sources.

### 5.4 ter Table logique `ebics_payload_profiles`

Objectif:

- porter des configurations reutilisables pour les ordres payload EBICS;
- simplifier REST, CLI, UI et `updateconf`;
- eviter d'enfouir la semantique EBICS dans `Rule`.

Champs cibles:

- `id`
- `owner`
- `name`
- `order_type`
- `service_name`
- `service_option`
- `scope`
- `msg_name`
- `container`
- `requires_declared_amount`
- `default_currency`
- `allowed_directions`
- `default_rule_id`
- `description`
- `created_at`
- `updated_at`

Principe:

- un profil payload peut referencer une `Rule` technique par defaut;
- une `Rule` ne porte pas seule la semantique EBICS du payload;
- le profil payload ne doit pas embarquer une seconde copie de la vue
  contractuelle;
- la validation contractuelle se fait contre la vue contractuelle active.

### 5.5 Table logique `ebics_transaction_segments`

Objectif:

- persister les metadonnees et eventuellement les donnees des segments.

### 5.6 Table logique `ebics_nonce_window`

Objectif:

- persister les nonces vus dans la fenetre anti-rejeu.

### 5.7 Table logique `ebics_key_lifecycles`

Objectif:

- tracer les rotations de cles et certificats EBICS.

Champs cibles:

- `id`
- `subscriber_id` ou `host_id`
- `key_usage`
- `current_credential_id`
- `next_credential_id`
- `rotation_type`
- `status`
- `requested_at`
- `sent_at`
- `activated_at`
- `retired_at`
- `operator`
- `evidence`

### 5.8 Table logique `ebics_initialization_workflows`

Objectif:

- porter le workflow d'initialisation et la rupture d'automatisme.

Champs cibles:

- `id`
- `subscriber_id`
- `init_mode`
- `status`
- `ini_transfer_id`
- `hia_transfer_id`
- `h3k_transfer_id`
- `letter_type`
- `letter_generated_at`
- `letter_delivery_mode`
- `letter_sent_at`
- `bank_feedback`
- `activated_at`
- `operator`
- `evidence`

### 5.9 Table logique `ebics_rtn_events`

Objectif:

- persister les notifications RTN et leur traitement.

Champs cibles:

- `id`
- `source`
- `event_id`
- `correlation_id`
- `idempotence_key`
- `host_id`
- `partner_id`
- `user_id`
- `order_type_hint`
- `profile_id`
- `payload`
- `status`
- `attempts`
- `next_retry_at`
- `received_at`
- `processed_at`

### 5.10 Table logique `ebics_contract_views`

Objectif:

- persister la projection technique du contrat EBICS publie par la banque via
  `HPD`, `HKD`, `HTD` et `HAA`.

Contenu cible:

- capacites protocolaire annoncees par `HPD`:
  `Recovery`, `PreValidation`, `ClientDataDownload`,
  `DownloadableOrderData`, versions supportees;
- liste des ordres et BTF recuperables;
- permissions par `PartnerID` / `UserID`, compte, `AdminOrderType`,
  `Service`, `AuthorisationLevel`, `MaxAmount`;
- metadonnees de provenance:
  ordre source, date de collecte, empreinte ou version logique, statut.

Principe:

- cette table ne porte pas le contrat juridique;
- elle porte uniquement la vue executable utile aux controles techniques;
- elle ne doit pas se resumer a quelques colonnes `TEXT` contenant du JSON
  libre.

Recommendation:

- utiliser `ebics_contract_views` comme entete de snapshot;
- ajouter des lignes structurees `ebics_contract_view_items` pour exposer
  simplement en REST/UI les services, permissions, ordres admin et limites.

## 6. Convention de responsabilites

La librairie EBICS doit rester responsable de:

- validation protocolaire;
- cryptographie EBICS;
- etat protocolaire et return codes;
- formatage XML et ordres.

Gateway doit rester responsable de:

- exposition des services et cycle de vie;
- persistance durable;
- administration;
- observabilite;
- droits et regles d'utilisation;
- orchestration avec les autres traitements MFT.
- prevention de l'emission d'ordres manifestement hors contrat technique connu.
- exposition de connecteurs standards vers le SI metier, y compris
  synchrones et asynchrones.

## 7. Integration au pipeline Gateway

### 7.1 Serveur EBICS

Pour les ordres impliquant un flux fichier:

- une requete EBICS aboutie doit pouvoir creer ou completer un `Transfer`;
- le pipeline Gateway doit rester le point d'execution des pre/post tasks;
- l'historique final doit rester aligne sur les autres protocoles.

Pour les ordres non fichier:

- ne pas forcer artificiellement un `Transfer` complet;
- preferer un objet de journalisation EBICS dedie avec lien eventuel vers un
  `Transfer` lorsque pertinent.

### 7.3 Workflow d'initialisation avec rupture d'automatisme

L'initialisation doit etre modelee comme un workflow explicite.

Principe:

- les ordres `INI` / `HIA` / `H3K` peuvent etre emis automatiquement;
- la generation de lettre repose sur les helpers `RenderINILetter`,
  `RenderHIALetter`, `RenderH3KLetter` de la librairie;
- l'envoi de la lettre et la confirmation banque restent hors bande et donc
  hors automatisme strict;
- Gateway doit conserver l'etat operatoire de cette attente.

Implication:

- une API de workflow est necessaire, distincte du simple lancement d'ordre;
- l'activation de l'abonne doit etre un pre-requis technique a la production.

### 7.4 Rotation de cles

La rotation doit etre modelee comme une operation de cycle de vie, pas comme un
simple remplacement de credential.

Principe:

- les credentials actifs et futurs doivent pouvoir coexister;
- le choix de la cle active doit etre pilotable par etat;
- les ordres de rotation doivent etre historises avec leurs preuves;
- la bascule ne doit intervenir qu'apres validation des retours banque.

### 7.5 Notifications RTN

RTN ne doit pas etre integre dans le coeur `Transfer` lui-meme, mais comme une
nouvelle capacite d'ingestion d'evenements.

Principe:

- intake RTN dedie;
- validation et parsing via `ebics/rtnspec` ou `provider/rtn`;
- stockage idempotent des evenements;
- mapping vers un plan de pulls EBICS;
- execution par le client EBICS Gateway.

### 7.6 Vue technique du contrat banque

Gateway doit integrer un sous-composant de collecte et d'exploitation du
"contrat technique" publie par la banque.

Principe:

- `HPD` fournit les capacites et options supportees;
- `HKD` / `HTD` fournissent les permissions et autorisations visibles via
  EBICS;
- `HAA` fournit la vue des BTF et services recuperables;
- ces informations doivent etre stockees, historisees et exploitees en amont
  de l'emission d'ordres.

Usage:

- aide a la configuration;
- controle preventif operateur;
- documentation de l'etat autorise cote banque;
- correlation entre rejet banque et contrat technique connu.

### 7.7 Connecteurs d'integration metier

La couche de passe-plat ne doit pas rester limitee a:

- filesystem;
- REST;
- CLI.

Il est recommande d'introduire une famille de connecteurs d'integration
asynchrone, avec comme priorite:

- `AMQP 0.9.1`
- `AMQP 1.0`

Responsabilites attendues:

- publication d'evenements techniques sortants;
- consommation de commandes techniques entrantes;
- correlation, idempotence et rejeu;
- gestion des erreurs de livraison et dead-letter si necessaire.

Positionnement:

- cette capacite est transverse a EBICS;
- EBICS en est un premier cas d'usage fort;
- la valeur depasse le seul perimetre EBICS et renforce globalement Gateway.

Important:

- les attributs tels que `AuthorisationLevel`, `NumSigRequired` ou
  `MaxAmount` doivent etre lus d'abord comme des faits techniques publies par
  la banque;
- ils servent a prevenir, expliquer et documenter;
- ils n'imposent pas, par eux-memes, une ouverture systematique du payload
  metier par Gateway.

### 7.2 Client EBICS

Le client EBICS doit permettre deux modes:

- mode transfert fichier, aligne sur les `Transfer` existants;
- mode commande administrative, pilote par API/CLI et historise hors pipeline
  lourd si aucun fichier n'est manipule.

## 8. Configuration protocolaire cible

### 8.1 ServerConfig EBICS

Le `ProtoConfig` serveur devra porter au minimum:

- `hostId`
- `endpointPath`
- `requireTLS`
- `requireMTLS`
- `strictXSD`
- `strictOrderDataXSD`
- `requireNonceTimestamp`
- `nonceTTL`
- `nonceMaxSkew`
- `maxBodyBytes`
- `maxSegmentBytes`
- `maxSegments`
- `handlerTimeout`
- `validateServiceCodeList`

### 8.2 ClientConfig EBICS

Le `ProtoConfig` client devra porter au minimum:

- `endpointURL`
- `serverName`
- `requireMTLS`
- `requestTimeout`
- `retryPolicy`
- `recoveryEnabled`
- `validateCodeList`
- `defaultHostId`
- `rtnEnabled`
- `rtnAutoPullPolicy`
- `rtnProfileCatalog`
- `businessIntegrationProfile`

### 8.3 PartnerConfig EBICS

Le `ProtoConfig` partenaire devra porter au minimum:

- `hostId`
- `bankURL`
- `tlsPolicy`
- `serviceCapabilities`
- `schemaPolicy`
- `contractRefreshPolicy`
- `businessConnectorProfile`

## 9. Gestion des credentials

Le systeme de `Credential` existant peut etre reutilise pour:

- certificats TLS;
- cles privees ou publiques EBICS;
- references de certificats de confiance.

Il est recommande d'introduire des types de credentials EBICS explicites, par
exemple:

- `ebics_auth_private_key`
- `ebics_auth_certificate`
- `ebics_enc_private_key`
- `ebics_enc_certificate`
- `ebics_sig_private_key`
- `ebics_sig_certificate`
- `ebics_bank_auth_public_key`
- `ebics_bank_enc_public_key`

Pour la rotation, il est recommande de completer ce modele par:

- notion de version logique;
- dates de validite;
- flag d'etat (`draft`, `active`, `retired`);
- rattachement a un workflow de rotation.

## 10. Journalisation, supervision et audit

A minima, il faut tracer:

- `HostID`, `PartnerID`, `UserID`, `OrderType`, `TransactionID`;
- correlation entre transaction EBICS et `Transfer.ID`;
- return code EBICS, phase, duree, taille, segmentation;
- cause racine technique ou fonctionnelle.
- etapes du workflow d'initialisation et evidence de lettre;
- cycle de rotation de cles;
- `event_id`, `idempotence_key` et statut RTN.
- version ou date de rafraichissement de la vue contractuelle technique.
- connecteur d'integration utilise et etat de livraison vers le SI metier.

L'observabilite devra s'integrer aux mecanismes existants:

- logs Gateway;
- analytics;
- SNMP si pertinent;
- historique de transferts;
- futur metriqueur si present.

## 11. Exigences de securite

- activer le profil strict H005 par defaut;
- imposer `TxStore` durable pour les flux segmentes;
- imposer `NonceStore` durable si anti-rejeu active;
- ne jamais activer automatiquement un abonne apres envoi technique des ordres
  d'initialisation sans preuve de validation banque;
- gouverner la rotation de cles par workflow et evidence;
- valider et dedupliquer strictement les notifications RTN;
- utiliser la vue contractuelle technique connue pour bloquer les ordres
  manifestement non autorises avant emission;
- stocker les secrets via les mecanismes chiffres existants;
- borner tailles, segments, timeouts et rejeter les charges malforme.
- securiser les remises vers le SI metier, y compris sur connecteurs de type
  AMQP.

## 12. Exigences de performance

- utiliser les facades de production de la librairie EBICS;
- eviter les copies memoire inutiles;
- privilegier des stores SQL indexes et purgables;
- separer transaction EBICS et historique long terme;
- conserver les limites de parallelisme du `controller`.

## 13. Proposition de phasage technique

Phase 1:

- ajout du protocole `ebics` dans le registre Gateway;
- configs serveur/client/partenaire;
- service serveur EBICS minimal;
- stores SQL pour `SubscriberStore`, `KeyStore`, `TxStore`, `NonceStore`;
- workflow d'initialisation avec lettre EBICS;
- administration CLI/REST de base.

Phase 2:

- creation des transferts Gateway pour `FUL`, `FDL`, `BTU`, `BTD`;
- liaison avec le pipeline et les tasks;
- gestion de rotation de cles;
- historique et supervision enrichis.

Phase 3:

- intake RTN, event store et auto-pull;
- ordres administratifs et reporting;
- premiers connecteurs de passe-plat asynchrones;
- UI dediee;
- outillage d'industrialisation, purge et runbooks.

Phase 4:

- `AMQP 0.9.1`;
- `AMQP 1.0`;
- outbox / consumer workers mutualises;
- supervision et runbooks d'exploitation messaging.
