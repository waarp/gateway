Fichier de configuration ``waarp-gatewayd.ini``
#############################################


.. module:: waarp-gatewayd.ini
   :synopsis: fichier de configuration du démon waarp-gatewayd

Le fichier de configuration ``waarp-gatewayd.ini`` permet de contrôler et modifier
le comportement du démon ``waarp-gatewayd``.

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

   L'adresse complète (IP + port) à laquelle le serveur HTTP va écouter les
   requêtes faites à l'interface d'administration. Si le port est mis à 0,
   le programme choisira un port libre au hasard.

   Valeur par défaut : ``127.0.0.1:8080``

.. confval:: TLSCert

   Le chemin du certificat TLS pour le serveur HTTP.
   Si ce paramètre n'est pas défini, le serveur utilisera HTTP à la place de
   HTTPS.

.. confval:: TLSKey

   Le chemin de la clé du certificat TLS.
   Si ce paramètre n'est pas défini, le serveur utilisera HTTP à la place de
   HTTPS.


Section ``[database]``
======================

La section ``[database]`` regroupe toutes les options de configuration de la
base de données de la gateway.

.. confval:: Type

   Le nom (en minuscules) du type de système de gestion de base de données utilisé.
   Les valeurs autorisées sont: ``postgresql``, ``mysql``, ``sqlite``.

.. confval:: Address

   L'adresse de la base de données.

   Valeur par défaut : ``localhost``

.. confval:: Port

   Le port sur lequel écoute le serveur de base de donnée.

   Valeur par défaut : dépend du type de base de donnée (``5432`` pour PostgreSQL,
   ``3306`` pour MySQL, aucun pour SQLite).

.. confval:: Name

   Le nom de la base de donnée utilisée.

.. confval:: User

   Le nom d'utilisateur du SGBD utilisé par la gateway pour faire des requêtes.

.. confval:: Password

   Le mot de passe de l'utilisateur du SGBD.

.. confval:: AESPassphrase

   Le chemin vers le fichier qui contient la clef AES utilisée pour chiffrer les
   mots de passes des comptes enregistrés dans la base de données.

   Si le fichier renseigné n'existe pas, une nouvelle clef est automatiquement
   générée et écrite à cet emplacement.
