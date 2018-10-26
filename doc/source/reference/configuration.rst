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
