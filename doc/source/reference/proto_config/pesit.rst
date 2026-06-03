.. _proto-config-pesit:

Configuration PeSIT & PeSIT-TLS
###############################

Configuration client
====================

Le configuration protocolaire d'un client affectera tous les transferts effectués
avec ce client. Il est possible d'écraser cette configuration au cas par cas via
la configuration des partenaires (voir ci-dessous). La structure de l'objet JSON
de configuration du protocole pour un client PeSIT est la suivante :

* **disableRestart** (*boolean*) - Désactive le "restart" pour tous les transferts
  effectués avec ce client. Par défaut, le "restart" est activé. Cette valeur
  peut être écrasée au cas par cas dans la configuration des partenaires
  (voir ci-dessous).
* **disableCheckpoints** (*boolean*) - Désactive les checkpoints pour tous les
  transferts effectués avec ce client. Par défaut, les checkpoints sont activés.
  Cette valeur peut être écrasée au cas par cas dans la configuration des
  partenaires (voir ci-dessous).
* **checkpointSize** (*integer*) - Spécifie la taille (en kilo-octets, PI 7) des blocs de
  données entre chaque checkpoint lors d'un transfert. N'a aucun effet si les
  checkpoints sont désactivés. Par défaut, les blocs entre checkpoints font
  32 Ko (32 768 octets).
* **checkpointWindow** (*integer*) - Spécifie le nombre de checkpoints pouvant
  rester sans réponse avant que le transfert soit stoppé. N'a aucun effet si
  les checkpoints sont désactivés. Par défaut, le transfert sera stoppé si 2
  checkpoints restent sans réponse du partenaire.
* **useCRC** (*boolean*) - Active le contrôle d'intégrité CRC-16 (PI 1) sur
  chaque paquet NSDU. Utile sur des liaisons réseau peu fiables. Par défaut :
  ``false``.

**Exemple**

.. code-block:: json

   {
     "disableRestart": false,
     "disableCheckpoints": false,
     "checkpointSize": 32,
     "checkpointWindow": 2,
     "useCRC": false
   }

Configuration partenaire
========================

La configuration protocolaire des partenaires PeSIT est identique à celle du
client. Cependant, si une option de la configuration du partenaire contredit la
configuration du client, la configuration du partenaire (plus spécifique)
est prioritaire. Si une option de la configuration du partenaire n'est pas
renseignée, la configuration du client est utilisée. La structure de l'objet
JSON de configuration du protocole pour un partenaire PeSIT est donc la suivante :

* **login** (*string*) - Le login du partenaire (optionnel). Le mot de passe du
  partenaire doit être renseigné via un :ref:`identifiant <reference-auth-methods>`
  de type *"password"* rattaché au partenaire.
* **disableRestart** (*boolean*) - Désactive le "restart" pour tous les transferts
  effectués avec ce partenaire. Par défaut, la valeur donnée dans la configuration
  du client est utilisée.
* **disableCheckpoints** (*boolean*) - Désactive les checkpoints pour tous les
  transferts effectués avec ce partenaire. Par défaut, la valeur donnée dans la
  configuration du client est utilisée.
* **checkpointSize** (*integer*) - Spécifie la taille (en kilo-octets, PI 7) des blocs de
  données entre chaque checkpoint lors d'un transfert. N'a aucun effet si les
  checkpoints sont désactivés. Par défaut, la valeur donnée dans la configuration
  du client est utilisée.
* **checkpointWindow** (*integer*) - Spécifie le nombre de checkpoints pouvant
  rester sans réponse avant que le transfert soit stoppé. N'a aucun effet si
  les checkpoints sont désactivés. Par défaut, la valeur donnée dans la
  configuration du client est utilisée.
* **useCRC** (*boolean*) - Active le contrôle d'intégrité CRC-16 (PI 1) sur
  chaque paquet NSDU pour ce partenaire. Par défaut : ``false``.
* **useNSDU** (*boolean*) - Spécifie si les méta-paquets NSDU du protocole PeSIT
  doivent être utilisés lors des transferts avec ce partenaire. Par défaut, les
  paquets NSDU sont utilisés.
* **compatibilityMode** (*string*) - Spécifie le mode de compatibilité à utiliser
  lors des communications avec le partenaire. Les valeurs autorisées sont :

  - ``standard`` (par défaut) : mode standard PeSIT. Le chemin du fichier distant
    contient le nom de la règle en préfixe (ex: ``regle/fichier.txt``). Le serveur
    identifie la règle à appliquer par correspondance de préfixe. Ce mode supporte
    également les **patterns glob** (``*``, ``?``) dans les requêtes de pull
    (voir :ref:`ref-proto-pesit-patterns`).
  - ``historique`` : mode de compatibilité avec les agents PeSIT utilisant la
    convention bancaire historique.
    Le champ *Filename* (PI 12) transmet le nom de la règle au lieu du chemin de
    fichier, et le nom de fichier est transmis via le champ *FileLabel* (PI 37).
* **maxMessageSize** (*integer*) - Spécifie la taille maximale (en octets) autorisée
  pour les paquets PeSIT envoyés à (et reçus depuis) ce partenaire. Le partenaire
  pourra unilatéralement décider d'utiliser une taille plus petite que celle-ci,
  mais jamais plus grande. La valeur par défaut est de 65535 octets.
* **disablePreConnection** (*boolean*) - Permet de désactiver le processus de
  pré-connexion (et la pré-authentification qui va avec) pour ce partenaire. Par
  défaut, un échange de pré-connexion est attendu à chaque nouvelle connexion.
* **articleFormat** (*string*) - Spécifie le format des articles pour les transferts
  sortants vers ce partenaire (PI 31). Les valeurs acceptées sont ``variable``
  (par défaut) et ``fixed``. Surcharge la configuration du client/serveur si renseigné.
  Peut également être surchargé par transfert via l'info de transfert
  ``__articleFormat__``.
* **compression** (*string*) - Spécifie l'algorithme de compression PeSIT (PI 21,
  Annexe A) pour les transferts avec ce partenaire. Les valeurs acceptées sont :
  ``none`` (par défaut), ``horizontal`` (RLE), ``vertical`` (inter-articles)
  et ``both`` (horizontale + verticale combinées).
* **maxConnections** (*integer*) - Limite le nombre de connexions PeSIT
  simultanées vers ce partenaire. Lorsque la limite est atteinte, les nouveaux
  transferts attendent qu'une connexion se libère. La valeur par défaut est 0
  (illimité). Une valeur de 4 est un compromis raisonnable entre parallélisme
  et protection du partenaire distant.
* **replyTo** (*string*) - Adresse de retour pour les acquittements F.MESSAGE
  (*Store and Forward*). Lorsque ce champ est renseigné, la Gateway ajoute
  automatiquement ``REPLY=<valeur>`` dans le texte libre PI 99 de chaque transfert
  sortant vers ce partenaire. Format : ``partenaire:compte`` ou ``partenaire``
  seul (le compte sera déduit). Voir :ref:`ref-proto-pesit` section F.MESSAGE.
* **minTLSVersion** (*string*) - [PeSIT-TLS uniquement] Spécifie la version
  minimale de TLS autorisée pour ce partenaire. Par défaut, la valeur "v1.2"
  (pour TLS 1.2) est utilisée.
* **cipherSuites** (*liste de strings*) - [PeSIT-TLS uniquement] Spécifie la
  liste des cipher suites TLS acceptées, par nom (ex: ``TLS_AES_128_GCM_SHA256``,
  ``TLS_RSA_WITH_AES_128_CBC_SHA256``). Si la liste est vide, les cipher suites
  par défaut de Go sont utilisées. Utile pour l'interopérabilité avec des
  partenaires mainframe ou legacy nécessitant des suites spécifiques.

**Exemple**

.. code-block:: json

   {
     "disableRestart": false,
     "disableCheckpoints": false,
     "checkpointSize": 32,
     "checkpointWindow": 2,
     "useNSDU": true,
     "compatibilityMode": "historique",
     "maxMessageSize": 65535,
     "maxConnections": 4,
     "minTLSVersion": "v1.2",
     "cipherSuites": ["TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"]
   }

Configuration serveur
=====================

La structure de l'objet JSON de configuration du protocole pour un serveur PeSIT
est la suivante :

* **disableRestart** (*boolean*) - Désactive le "restart" pour tous les transferts
  effectués avec ce serveur. Par défaut, le "restart" est activé.
* **disableCheckpoints** (*boolean*) - Désactive les checkpoints pour tous les
  transferts effectués avec ce serveur. Par défaut, les checkpoints sont activés.
* **checkpointSize** (*integer*) - Spécifie la taille maximale (en kilo-octets, PI 7) des 
  blocs de données entre chaque checkpoint lors d'un transfert. Si un client se
  connectant au serveur demande une taille plus grande, celle-ci sera rabaissée
  à ce maximum. N'a aucun effet si les checkpoints sont désactivés. Par défaut,
  les blocs entre checkpoints font 32 Ko (32 768 octets).
* **checkpointWindow** (*integer*) - Spécifie le nombre maximum de checkpoints 
  pouvant rester sans réponse avant que le transfert soit stoppé. Si un client se
  connectant au serveur demande un interval plus grand, celui-ci sera rabaissé
  à ce maximum. N'a aucun effet si les checkpoints sont désactivés. Par défaut,
  le transfert sera stoppé si 2 checkpoints restent sans réponse du récepteur.
* **useCRC** (*boolean*) - Active le contrôle d'intégrité CRC-16 (PI 1) sur
  chaque paquet NSDU pour ce serveur. Par défaut : ``false``.
* **maxMessageSize** (*integer*) - Spécifie la taille maximale (en octets) autorisée
  pour les paquets PeSIT envoyés à (et reçus depuis) ce serveur. Si un client se
  connectant au serveur demande une taille plus grande, celle-ci sera rabaissée
  à ce maximum. La valeur par défaut est de 65535 octets.
* **articleFormat** (*string*) - Spécifie le format des articles pour les transferts
  sortants (PI 31). Les valeurs acceptées sont ``variable`` (par défaut) et ``fixed``.
  Le format fixe est utilisé pour les fichiers à enregistrements de taille constante
  (ex: fichiers mainframe COBOL de 80 octets). Cette valeur peut être surchargée
  par transfert via l'info de transfert ``__articleFormat__``.
* **compression** (*string*) - Spécifie l'algorithme de compression PeSIT (PI 21,
  Annexe A). Les valeurs acceptées sont : ``none`` (par défaut), ``horizontal``
  (RLE), ``vertical`` (inter-articles) et ``both`` (combinées). La compression
  est négociée à l'ouverture du fichier (F.OPEN) et est transparente pour les
  données.
* **disablePreConnection** (*boolean*) - Permet de désactiver le processus de
  pré-connexion (et la pré-authentification qui va avec) si le partenaire client
  ne le supporte pas. Lorsque ce paramètre est activé, le serveur assume
  directement un cadrage NSDU sans tenter d'auto-détection. Par défaut, la
  pré-connexion est gérée par auto-détection.
* **relayMessages** (*boolean*) - Active le relais automatique des F.MESSAGE
  (*Store and Forward*). Lorsqu'un partenaire envoie un F.MESSAGE d'acquittement,
  le serveur retrouve le transfert d'origine et relaie automatiquement le message
  vers l'émetteur initial. Par défaut : ``false``. Voir :ref:`ref-proto-pesit`
  section F.MESSAGE.
* **minTLSVersion** (*string*) - [PeSIT-TLS uniquement] Spécifie la version
  minimale de TLS autorisée par ce serveur. Par défaut, la valeur "v1.2"
  (pour TLS 1.2) est utilisée.
* **cipherSuites** (*liste de strings*) - [PeSIT-TLS uniquement] Spécifie la
  liste des cipher suites TLS acceptées par ce serveur. Si la liste est vide,
  les cipher suites par défaut de Go sont utilisées.
* **tlsClientAuth** (*string*) - [PeSIT-TLS uniquement] Spécifie comment les
  certificats client TLS sont utilisés pour l'authentification. Valeurs :

  - ``none`` (par défaut) : le certificat client est optionnel ;
    l'authentification repose sur le login/mot de passe PeSIT (PI 3/PI 5).
  - ``optional`` : si le client présente un certificat valide et envoie un
    mot de passe vide, le CN ou SAN du certificat est utilisé pour identifier
    le compte local. Sinon, le login/mot de passe PeSIT est utilisé.
  - ``required`` : le client **doit** présenter un certificat valide. Le CN
    ou SAN identifie le compte local ; le mot de passe PeSIT est ignoré.

**Exemple**

.. code-block:: json

   {
     "disableRestart": false,
     "disableCheckpoints": false,
     "checkpointSize": 32,
     "checkpointWindow": 1,
     "maxMessageSize": 65535,
     "minTLSVersion": "v1.2"
   }
