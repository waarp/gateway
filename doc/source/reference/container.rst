.. _ref-config-container:

############################
Configuration des containers
############################


Lancement d'une image
=====================

Les images OCI sont stockées sur le registre ``code.waarp.fr``. Le Dockerfile
utilisé pour générer les images est consultable dans `le dépôt de l'application
<https://code.waarp.fr/apps/gateway/gateway/-/blob/master/Dockerfile>`__.

Le lancement peut se faire avec une des commandes suivantes :

.. code-block:: sh

   # avec docker
   docker run code.waarp.fr:5000/apps/gateway/gateway:latest

   # avec podman
   podman run code.waarp.fr:5000/apps/gateway/gateway:latest

Ports exposés
=============

L'image ``gateway`` expose seulement le port ``8080``,  c'est-à-dire celui de
l'interface REST qui permet d'administrer l'instance de Waarp Gateway, 

Les ports des serveurs protocolaires de Gateway étant :any:`définis par
paramétrage <user-server-management>`, il n'est pas possible de les exposer dans
l'image. N'oubliez pas de les exposer au lancement du container.

Volumes déclarés
================

L'image ``gateway`` déclare deux volumes :

* :file:`/app/etc`, qui contient la configuration de l'instance ;
* :file:`/app/data`, qui contient toutes les données (fichiers reçus, fichiers
  à envoyer, base de données sqlite, etc.

.. note:: 

   Tous les fichiers utilisés par Gateway sont modifiables par variable
   d'environnement au lancement du container.

Variables d'environnement
=========================

Configuration de :program:`waarp-gatewayd`
------------------------------------------

L'image ``gateway`` utilise plusieurs variables d'environnement pour configurer
l'instance exécutée dans le container.

Ces variables vont être utilisées pour réécrire le fichier de configuration
utilisé par :program:`waarp-gatewayd`.


.. envvar:: WAARP_GATEWAY_CONFIG

   Permet de spécifier le chemin vers le fichier de configuration à utiliser.
   Cette option permet de réutiliser un fichier de configuration ou de fournir à
   :program:`waarp-gatewayd` un fichier pré-paramétré.

   .. warning:: 

      Le fichier de configuration spécifié va être réécrit avec les les
      variables d'environnement ci-dessous si elles sont fournies

.. envvar:: WAARP_GATEWAY_NODE_ID

   Le nom unique de l'instance de :program:`waarp-gatewayd` lors d'un
   fonctionnement en grappe. Correspond à l'argument :option:`waarp-gatewayd
   server --instance` 


.. envvar:: WAARP_GATEWAY_NAME

   Le nom de cette instance de :program:`waarp-gatewayd`.

   Correspond à l'option :confval:`GatewayName` du fichier de configuration.

.. envvar:: WAARP_GATEWAY_HOME

   Définit la racine de :program:`waarp-gatewayd`.

   Correspond à l'option :confval:`GatewayHome` du fichier de configuration.

.. envvar:: WAARP_GATEWAY_IN_DIR

   Définit le dossier par défaut dans lequel sont déposés les fichiers reçus par
   :program:`waarp-gatewayd`.

   Correspond à l'option :confval:`DefaultInDirectory` du fichier de
   configuration.

.. envvar:: WAARP_GATEWAY_OUT_DIR

   Définit le dossier par défaut dans lequel sont lus les fichiers envoyés par
   :program:`waarp-gatewayd`.

   Correspond à l'option :confval:`DefaultOutDirectory` du fichier de
   configuration.

.. envvar:: WAARP_GATEWAY_TMP_DIR

   Définit le dossier par défaut dans lequel sont déposés les fichiers en cours
   de réception par :program:`waarp-gatewayd`.

   Correspond à l'option :confval:`DefaultTmpDirectory` du fichier de
   configuration.

.. envvar:: WAARP_GATEWAY_LOG_LEVEL

   Définit le niveau de verbosité des logs. Les valeurs possibles sont :
   ``DEBUG``, ``INFO``, ``WARNING``, ``ERROR`` et ``CRITICAL``.

   Correspond à l'option :confval:`Level` du fichier de
   configuration.

.. envvar:: WAARP_GATEWAY_LOG_TO

   Le chemin du fichier d'écriture des logs.
   Les valeurs spéciales ``stdout`` et ``syslog`` permettent de rediriger les
   logs respectivement vers la sortie standard et vers un démon syslog.
   
   Correspond à l'option :confval:`LogTo` du fichier de
   configuration.

.. envvar:: WAARP_GATEWAY_SYSLOG_FACILITY

   Quand :any:`LogTo` est défini à ``syslog``, cette option permet de définir
   l'origine (*facility*) du message.

   Correspond à l'option :confval:`SyslogFacility` du fichier de
   configuration.

.. envvar:: WAARP_GATEWAY_ADMIN_ADDRESS

   L'adresse de l'interface sur laquelle le serveur HTTP va écouter les
   requêtes faites à l'interface d'administration.

   Correspond à l'option :confval:`Host` du fichier de
   configuration.

.. envvar:: WAARP_GATEWAY_ADMIN_PORT

   Le port sur lequel le serveur HTTP doit écouter. La valeur '0' est entrée,
   un port libre sera arbitrairement choisit.

   Correspond à l'option :confval:`Port` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_ADMIN_TLS_KEY

   Le chemin de la clé TLS pour le serveur HTTP. Si ce paramètre n'est pas
   défini, le serveur utilisera du HTTP en clair à la place de HTTPS.

   Correspond à l'option :confval:`SyslogFacility` du fichier de
   configuration.

   .. note:: 
      
      La clé est requise pour une utilisation avec Waarp Manager. Si aucune clé
      n'est fournie avec cette variable d'environnement, une clef sera générée
      au lancement du container.


.. envvar:: WAARP_GATEWAY_ADMIN_TLS_CERT

   Le chemin du certificat TLS pour le serveur HTTP. Si ce paramètre n'est pas
   défini, le serveur utilisera du HTTP en clair à la place de HTTPS.

   Correspond à l'option :confval:`SyslogFacility` du fichier de
   configuration.

   .. note:: 
      
      Le certificat est requis pour une utilisation avec Waarp Manager. Si aucun
      certificat n'est fourni avec cette variable d'environnement, un certificat
      auto-signé sera généré au lancement du container.


.. envvar:: WAARP_GATEWAY_DB_TYPE

   Le nom (en minuscules) du type de système de gestion de base de données utilisé.
   Les valeurs autorisées sont: ``postgresql``, ``mysql``, ``sqlite``.

   Correspond à l'option :confval:`Type` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_DB_ADDRESS

   L'adresse complète (URL + Port) de la base de données. Le port par défaut
   dépend du type de base de données utilisé (``5432`` pour PostgreSQL, ``3306``
   pour MySQL, aucun pour SQLite).

   Correspond à l'option :confval:`Address` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_DB_NAME

   Le nom de la base de donnée utilisée.

   Correspond à l'option :confval:`Name` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_DB_USER

   Le nom d'utilisateur du SGBD utilisé par la gateway pour faire des requêtes.

   Correspond à l'option :confval:`User` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_DB_PASSWORD

   Le mot de passe de l'utilisateur du SGBD.

   Correspond à l'option :confval:`Password` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_DB_TLS_KEY

   La clé du certificat TLS de la base de données.

   Correspond à l'option :confval:`TLSKey` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_DB_TLS_CERT

   Le certificat TLS de la base de données. Par défaut, les requêtes n'utilisent
   pas TLS.

   Correspond à l'option :confval:`TLSCert` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_DB_AES_PASSPHRASE

   Le chemin vers le fichier qui contient la clef AES utilisée pour chiffrer les
   mots de passes des comptes enregistrés dans la base de données.

   Correspond à l'option :confval:`AESPassphrase` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_MAX_IN

   Le nombre maximum autorisé de transferts entrants simultanés. Illimité par
   défaut.

   Correspond à l'option :confval:`MaxTransfersIn` du fichier de
   configuration.


.. envvar:: WAARP_GATEWAY_MAX_OUT

   Le nombre maximum autorisé de transferts sortants simultanés. Illimité par
   défaut.

   Correspond à l'option :confval:`MaxTransfersOut` du fichier de
   configuration.



Synchronisation depuis manager
------------------------------

.. envvar:: WAARP_GATEWAY_MANAGER_URL

   URL à utiliser pour ce connecter à Waarp Manager. Si cette variable
   d'environnement est renseignée, la configuration de ``Gateway`` (partenaires,
   règles de transfert, etc...) est téléchargée depuis Manager au lancement du
   container.

   Si l'instance de Gateway n'est pas déclarée dans Manager, Elle sera
   automatiquement déclarée et un flux de configuration est créé.

   L'URL doit contenir les identifiants d'un utilisateur ayant le droit de
   déployer la configuration, et de créer des partenaires et des flux si
   l'instance de Gateway n'est pas déclarée préalablement dans Manager ::

     https://USER:PASSWORD@manager.tld:8080


.. envvar:: WAARP_GATEWAY_MANAGER_SITE

   *Cette variable d'environnement est requise si l'instance de Gateway n'est
   pas déclarée préalablement dans Manager, et elle est ignorée sinon.*

   Défini le site dans lequel le partenaire doit être créé.


.. envvar:: WAARP_GATEWAY_MANAGER_IP

   *Cette variable d'environnement est requise si l'instance de Gateway n'est
   pas déclarée préalablement dans Manager, et elle est ignorée sinon.*

   Défini l'adresse ou le domaine du partenaire créé dans Manager.


.. envvar:: WAARP_GATEWAY_MANAGER_PASSWORD

   *Cette variable d'environnement est requise si l'instance de Gateway n'est
   pas déclarée préalablement dans Manager, et elle est ignorée sinon.*

   Défini le mot de passe du partenaire créé dans Manager. S'il n'est pas fourni
   avec cette variable d'environnement, un mot de passe aléatoire est généré.


.. envvar:: WAARP_GATEWAY_MANAGER_R66_PORT

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*

   Défini le port de communication R66 pour la Gateway définie dans Manager. Par
   défaut: ``6666``.


.. envvar:: WAARP_GATEWAY_MANAGER_R66TLS_PORT

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*

   Défini le port de communication R66 TLS pour la Gateway définie dans Manager.
   Par défaut: ``6667``.


.. .. envvar:: WAARP_GATEWAY_MANAGER_SFTP_PORT

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*

   Défini le port de communication SFTP pour la Gateway définie dans Manager.
   Par défaut: ``6622``.


.. envvar:: WAARP_GATEWAY_MANAGER_REST_USERNAME

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*

   Défini le nom d'utilisateur que Manager doit utiliser pour se connecter en
   REST à Gateway lors de la création du partenaire. Par défaut : ``admin``.


.. envvar:: WAARP_GATEWAY_MANAGER_REST_PASSWORD

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*

   Défini le mot de passe que Manager doit utiliser pour se connecter en
   REST à Gateway lors de la création du partenaire. Par défaut : ``admin``.


.. .. envvar:: WAARP_GATEWAY_MANAGER_SSH_PUBLIC_KEY_PATH

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*

   Définit le chemin vers la clé publique pour les communications SFTP de la
   Gateway défini dans Manager. Si aucun chemin n'est fourni avec cette
   variable d'environnement, une clé sera générée lors du démarrage du
   container.


.. .. envvar:: WAARP_GATEWAY_MANAGER_SSH_PRIVATE_KEY_PATH

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*

   Définit le chemin vers la clé privée pour les communications SFTP de la
   Gateway défini dans Manager. Si aucun chemin n'est fourni avec cette
   variable d'environnement, une clé sera générée lors du démarrage du
   container.


.. envvar:: WAARP_GATEWAY_MANAGER_R66_TLS_CERT_PATH

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*

   Définit le chemin vers le certificat pour les communications R66 TLS de la
   Gateway défini dans Manager. Si aucun chemin n'est fourni avec cette variable
   d'environnement, un certificat sera généré lors du démarrage du container.


.. envvar:: WAARP_GATEWAY_MANAGER_R66_TLS_KEY_PATH

   *Cette variable d'environnement est ignorée si l'instance de Gateway est déjà
   déclarée dans Manager.*


   Définit le chemin vers a clef privée pour les communications R66 TLS de la
   Gateway défini dans Manager. Si aucun chemin n'est fourni avec cette variable
   d'environnement, un certificat sera généré lors du démarrage du container.


Séquence de démarrage
=====================

Lors du lancement du container, plusieurs vérifications et opérations de
configurations sont réalisées avant le lancement de Waarp Gateway.

Voici le processus suivi lors du démarrage :


.. uml::

   start

   if (Fichier de configuration existe) then (oui)
      :Lecture du fichier de configuration;
   else (non)
      :Configuration par défaut;
   endif

   :Mise à jour de la configuration
   à partir de l'environnement;

   :Écriture du fichier de configuration;

   if (Utilisation de Manager) then (oui)

      if (Certificats présents) then (oui)
      else (non)
         :Génération de certificats
         auto-signés;
      endif

      if (Gateway est déclarée dans Manager) then (oui)
         :Téléchargement de la configuration
         depuis Manager;

         :Import de la configuration
         dans Gatewayd;
      else (non)
      endif

   else (non)
   endif

   :Lancement de Gatewayd;

   if (Gateway est déclarée dans Manager) then (oui)
   else (non)
      :Vérification des variables
      d'environnement requises
      pour l'enregistrement du partenaire;

      :Enregistrement dans Manager;

      :Création du flux de
      configuration;

      :Téléchargement de la configuration
      depuis Manager;

      :Import de la configuration
      dans Gatewayd;
         
      :Redémarrage de Gatewayd;
   endif

   stop


