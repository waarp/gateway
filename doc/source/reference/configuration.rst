Fichier de configuration ``waarp-gatewayd.ini``
#############################################


.. module:: waarp-gatewayd.ini
   :synopsis: fichier de configuration du démon waarp-gatewayd

Le fichier de configuration ``waarp-gatewayd.ini`` permet de contrôler et modifier
le comportement du démon ``waarp-gatewayd``.

Section ``[path]``
==================

La section ``[path]`` contient les différents chemins de la gateway.

.. confval:: GatewayHome

   Définit la racine de la *gateway*. Par défaut, il s'agit du *working directory*
   depuis lequel la *gateway* a été lancée. Pour cette raison il est impératif de
   changer cette valeur si la *gateway* n'est pas lancée depuis son dossier racine.

.. confval:: InDirectory

   Le dossier dans lequel sont déposés les fichiers reçus. Par défaut, la racine
   de la *gateway* est utilisée à la place.

.. confval:: OutDirectory

   Le dossier dans lequel les fichiers à envoyer sont cherchés. Par défaut,
   la racine de la *gateway* est utilisée à la place.

.. confval:: WorkDirectory

   Le dossier dans lequel les fichiers en cours de réception sont déposés avant
   d'être déplacés dans le ``InDirectory``. Par défaut, la racine de la *gateway*
   est utilisée à la place.

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
interfaces d'administration de la gateway. Cela comprend l'interface d'admin
et l'API REST.

.. confval:: Address

   L'adresse de l'interface sur laquelle le serveur HTTP va écouter les
   requêtes faites à l'interface d'administration.

   Valeur par défaut : ``localhost``

.. confval:: Port

   Le port sur lequel le serveur HTTP doit écouter. La valeur '0' est entrée,
   un port libre sera arbitrairement choisit.

   Valeur par défaut : ``8080``

.. confval:: TLSCert

   Le chemin du certificat TLS pour le serveur HTTP. Si ce paramètre n'est pas
   défini, le serveur utilisera du HTTP en clair à la place de HTTPS.

.. confval:: TLSKey

   Le chemin de la clé du certificat TLS. Si ce paramètre n'est pas défini,
   le serveur utilisera du HTTP en clair à la place de HTTPS.


Section ``[database]``
======================

La section ``[database]`` regroupe toutes les options de configuration de la
base de données de la gateway.

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

   Le nom d'utilisateur du SGBD utilisé par la gateway pour faire des requêtes.

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
:term:`contrôleur` de la *gateway*.

.. confval:: Delay

   La durée de l'intervalle entre chaque requête du contrôleur à la base de
   données. Les unités de temps acceptées sont : "ns", "us" (ou "µs"), "ms",
   "s", "m", "h".

   Valeur par défaut : ``5s``

.. confval:: R66Home

   Le dossier racine du serveur *Waarp-R66* associé à cette *gateway* (s'il y en
   a un).

.. confval:: MaxTransfersIn

   Le nombre maximum autorisé de transferts entrants simultanés. Illimité par défaut.

.. confval:: MaxTransfersOut

   Le nombre maximum autorisé de transferts sortants simultanés. Illimité par défaut.