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

   L'adresse complete (IP + port) à laquelle le serveur HTTP va écouter les
   requêtes faites à l'interface d'administration. Si le port est mis à 0,
   le programme choisira un port libre au hasard.

   Valeur par défaut : ``127.0.0.1:8080``

.. confval:: TLSCert

   Le chemin du certificat TLS pour le serveur HTTP.
   Si ce paramètre n'est pas définit, le serveur utilisera HTTP à la place de
   HTTPS.

.. confval:: TLSKey

   Le chemin de la clé du certificat TLS.
   Si ce paramètre n'est pas définit, le serveur utilisera HTTP à la place de
   HTTPS.



