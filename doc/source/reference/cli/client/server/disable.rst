==================================
Désactiver un serveur au démarrage
==================================

.. program:: waarp-gateway server disable

.. describe:: waarp-gateway server disable <SERVER>

Désactive le serveur donné, signifiant que celui-ci ne sera pas démarré automatiquement
lors du prochain lancement de la *gateway*. Le serveur restera cependant actif
jusqu'à l'arrêt de la *gateway*.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' disable 'serveur_sftp'