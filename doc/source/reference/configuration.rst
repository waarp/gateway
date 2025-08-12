.. _configuration-file:

Fichier de configuration ``waarp-gatewayd.ini``
###############################################


.. module:: waarp-gatewayd.ini
   :synopsis: fichier de configuration du démon :program:`waarp-gatewayd`

Le fichier de configuration ``waarp-gatewayd.ini`` permet de contrôler et modifier
le comportement du démon ``waarp-gatewayd``.

.. confval:: GatewayName

   Définit le nom de Gateway. Par défaut, le nom ``waarp-gateway`` est utilisé.
   Il est cependant recommandé de donner un nom unique à chaque nouvelle instance
   pour éviter les confusions.

   Il est également possible de donner le même nom à plusieurs instances partageant
   une même base de données. Dans ce cas, les instances seront des copies les unes
   des autres (elles auront la même configuration et le même historique).

   .. warning::
      Dans une configuration avec plusieurs instances identiques, il est
      très fortement déconseillé de laisser la valeur par défaut de ``MaxTransfersOut``,
      et de fixer une limite au nombre de transferts client autorisés en parallèle
      pour une instance. En l'absence de limite, les transferts clients seront tous
      exécutés par la même instance au lieu d'être répartis sur les différentes
      instances.

Section ``[path]``
==================

La section ``[path]`` contient les différents chemins de Gateway.

.. confval:: GatewayHome

   Définit la racine de Waarp Gateway. Par défaut, il s'agit du dossier
   depuis lequel Waarp Gateway a été lancée. Pour cette raison il est impératif de
   changer cette valeur si Waarp Gateway n'est pas lancée depuis son dossier racine.

.. confval:: InDirectory

   .. deprecated:: 0.4.0

      Remplacé par ``DefaultInDir``

   Le dossier dans lequel sont déposés les fichiers reçus. Par défaut, la racine
   de Waarp Gateway est utilisée à la place.

.. confval:: OutDirectory

   .. deprecated:: 0.4.0

      Remplacé par ``DefaultOutDir``

   Le dossier dans lequel les fichiers à envoyer sont cherchés. Par défaut,
   la racine de Waarp Gateway est utilisée à la place.

.. confval:: WorkDirectory

   .. deprecated:: 0.4.0 

      Remplacé par ``DefaultTmpDir``

   Le dossier dans lequel les fichiers en cours de réception sont déposés avant
   d'être déplacés dans le ``InDirectory``. Par défaut, la racine de Waarp Gateway
   est utilisée à la place.

.. confval:: DefaultInDirectory

   Le dossier par défaut dans lequel sont déposés les fichiers reçus si le serveur
   et la règle concernés ne spécifient pas ce dossier de réception. Par défaut,
   un dossier 'in' est créé à cet effet sous la racine de Waarp Gateway.

.. confval:: DefaultOutDirectory

   Le dossier par défaut depuis lequel sont récupérés les fichiers à envoyer si
   le serveur et la règle concernés ne spécifient pas ce dossier d'envoi. Par
   défaut, un dossier 'out' est créé à cet effet sous la racine de Waarp Gateway.

.. confval:: DefaultTmpDirectory

   Le dossier par défaut dans lequel sont déposés les fichiers en cours de réception
   (avant dépôt dans le dossier de réception *in*) si le serveur et la règle
   concernés ne spécifient pas ce dossier temporaire. Par défaut, un dossier
   :file:`tmp` est créé à cet effet sous la racine de Waarp Gateway.

.. confval:: FilePermissions

   Les permissions appliquées à tous les fichiers de transferts créés par la
   Gateway. Les permissions doivent être en format octal (*0777*). Par défaut,
   les fichiers seront créés avec les permissions *0640*. Cette option est
   ineffective sous Windows, car les permissions y sont gérées différemment.

.. confval:: DirectoryPermissions

   Les permissions appliquées à tous les dossiers de transferts créés par la
   Gateway. Les permissions doivent être en format octal (*0777*). Par défaut,
   les dossiers seront créés avec les permissions *0750*. Cette option est
   ineffective sous Windows, car les permissions y sont gérées différemment.

Section ``[log]``
=================

La section ``[log]`` regroupe toutes les options qui permettent d'ajuster la
génération des traces du démon.

.. confval:: Level

   Définit le niveau de verbosité des logs. Les valeurs possibles sont :
   ``DEBUG``, ``INFO``, ``WARNING``, ``ERROR`` et ``CRITICAL``.

   Valeur par défaut : ``INFO``

.. confval:: LogTo

   Le chemin du fichier d'écriture des logs.
   Les valeurs spéciales ``stdout`` et ``syslog`` permettent de rediriger les
   logs respectivement vers la sortie standard et vers un démon syslog.

   Valeur par défaut : ``stdout``

.. confval:: SyslogFacility

   Quand :any:`LogTo` est défini à ``syslog``, cette option permet de définir
   l'origine (*facility*) du message.

   Valeur par défaut : ``local0``


Section ``[admin]``
===================

La section ``[admin]`` regroupe toutes les options de configuration des
interfaces d'administration de Gateway. Cela comprend l'interface
d'administration et l'API REST.

.. confval:: Host

   L'adresse de l'interface sur laquelle le serveur HTTP va écouter les
   requêtes faites à l'interface d'administration.

   Valeur par défaut : ``localhost``

.. confval:: Port

   Le port sur lequel le serveur HTTP doit écouter. La valeur ``0`` est entrée,
   un port libre sera arbitrairement choisit.

   Valeur par défaut : ``8080``

.. confval:: TLSCert

   Le chemin du certificat TLS pour le serveur HTTP. Si ce paramètre n'est pas
   défini, le serveur utilisera du HTTP en clair à la place de HTTPS.

.. confval:: TLSKey

   Le chemin de la clé du certificat TLS. Si ce paramètre n'est pas défini,
   le serveur utilisera du HTTP en clair à la place de HTTPS.

.. confval:: TLSPassphrase

   Le mot de passe de la clé du certificat (si la clé est chiffrée).
   Waarp Gateway supporte uniquement les clés privées en format PEM et chiffrées
   via la méthode décrite dans la :rfc:`1423`. Une clé chiffrée avec cette méthode
   doit avoir un entête PEM ``DEK-Info``.


Section ``[database]``
======================

La section ``[database]`` regroupe toutes les options de configuration de la
base de données de Gateway.

.. confval:: Type

   Le nom (en minuscules) du type de système de gestion de base de données utilisé.
   Les valeurs autorisées sont: ``postgresql``, ``mysql``, ``sqlite``.

.. confval:: Address

   L'adresse complète (URL + Port) de la base de données. Le port par défaut
   dépend du type de base de données utilisé (``5432`` pour PostgreSQL, ``3306``
   pour MySQL, aucun pour SQLite).

   Valeur par défaut : ``localhost``

.. confval:: Name

   Le nom de la base de donnée utilisée.

.. confval:: User

   Le nom d'utilisateur du SGBD utilisé par Gateway pour faire des requêtes.

.. confval:: Password

   Le mot de passe de l'utilisateur du SGBD.

.. confval:: TLSCert

   Le certificat TLS de la base de données. Par défaut, les requêtes n'utilisent
   pas TLS.

.. confval:: TLSKey

   La clé du certificat TLS de la base de données.

.. confval:: AESPassphrase

   Le chemin vers le fichier qui contient la clef AES utilisée pour chiffrer les
   mots de passes des comptes enregistrés dans la base de données.

   Si le fichier renseigné n'existe pas, une nouvelle clef est automatiquement
   générée et écrite à cet emplacement.

   Valeur par défaut : ``passphrase.aes``


Section ``[controller]``
========================

La section ``[controller]`` regroupe toutes les options de configuration du
:term:`contrôleur` de Waarp Gateway.

.. confval:: Delay

   La durée de l'intervalle entre chaque requête du contrôleur à la base de
   données. Les unités de temps acceptées sont : "ns", "us" (ou "µs"), "ms",
   "s", "m", "h".

   Valeur par défaut : ``5s``

.. confval:: MaxTransfersIn

   Le nombre maximum autorisé de transferts entrants simultanés. Illimité par défaut.

.. confval:: MaxTransfersOut

   Le nombre maximum autorisé de transferts sortants simultanés. Illimité par défaut.
