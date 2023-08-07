==================================
Désactiver un serveur au démarrage
==================================

.. program:: waarp-gateway server disable

Désactive le serveur donné, signifiant que celui-ci ne sera pas démarré automatiquement
lors du prochain lancement de Waarp Gateway. Le serveur restera cependant actif
jusqu'à l'arrêt de Waarp Gateway.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' disable 'serveur_sftp'
