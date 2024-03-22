=================================
Désactiver un client au démarrage
=================================

.. program:: waarp-gateway client disable

.. describe:: waarp-gateway client disable <CLIENT>

Désactive le client donné, signifiant que celui-ci ne sera pas démarré automatiquement
lors du prochain lancement de la *gateway*. Le client restera cependant actif
jusqu'à ce qu'il soit arrêté.

|

**Exemple**

.. code-block:: shell

   waarp-gateway disable 'sftp_client'